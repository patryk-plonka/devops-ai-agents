package azure

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type fakeCredential struct{}

func (fakeCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "synthetic-test-token", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

func TestLiveValidationAndReadBoundary(t *testing.T) {
	body := fmt.Sprintf(`{"id":"/subscriptions/%s","subscriptionId":"%s","tenantId":"%s","displayName":"ignored"}`, selectedID, selectedID, guestTenant)
	for _, tc := range []struct {
		name           string
		status         int
		body, category string
	}{
		{"success", 200, body, ""}, {"denied", 403, `{"error":{"code":"AuthorizationFailed","message":"secret-canary"}}`, "subscription_access_denied"},
		{"expired", 401, `{}`, "login_required"}, {"not found", 404, `{}`, "subscription_inaccessible"},
		{"registration disabled", 409, `{"error":{"code":"MissingSubscriptionRegistration","message":"secret-canary"}}`, "subscription_validation_failed"},
		{"wrong subscription", 200, strings.ReplaceAll(body, selectedID, otherID), "scope_mismatch"},
		{"wrong tenant", 200, strings.ReplaceAll(body, guestTenant, otherID), "scope_mismatch"},
		{"missing identity", 200, `{}`, "scope_mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			v := Validator{Credential: func(id string) (azcore.TokenCredential, error) {
				if id != selectedID {
					t.Fatal("wrong binding")
				}
				return fakeCredential{}, nil
			}, Base: roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/subscriptions/"+selectedID {
					t.Fatal("unexpected ARM operation")
				}
				if _, ok := r.Context().Deadline(); !ok {
					t.Fatal("unbounded validation")
				}
				return response(tc.status, tc.body), nil
			})}
			err := v.Validate(context.Background(), account(selectedID, "client"))
			if tc.category == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertFailure(t, err, tc.category)
				if strings.Contains(err.Error(), "secret-canary") {
					t.Fatal("leaked error")
				}
			}
			if calls != 1 {
				t.Fatalf("%d HTTP calls; unexpected retry/registration", calls)
			}
		})
	}
}

func TestAzureCLICredentialBindsGuestSubscriptionInsteadOfDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake az executable uses POSIX shell")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "args")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$FAKE_AZ_LOG"
printf '%s\n' '{"accessToken":"synthetic-guest-token","expires_on":4102444800}'
`
	if err := os.WriteFile(filepath.Join(dir, "az"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_AZ_LOG", log)
	t.Setenv("AZURE_CONFIG_DIR", dir)
	// The cached default account belongs to a different subscription and home tenant.
	if err := os.WriteFile(filepath.Join(dir, "azureProfile.json"), []byte(`{"subscriptions":[{"id":"`+otherID+`","isDefault":true,"tenantId":"`+otherID+`"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := subscriptionCredential(selectedID)
	if err != nil {
		t.Fatal(err)
	}
	token, err := c.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}})
	if err != nil || token.Token != "synthetic-guest-token" {
		t.Fatal("fake CLI credential failed")
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	args := string(b)
	if !strings.Contains(args, "--subscription\n"+selectedID+"\n") || strings.Contains(args, "--tenant") || strings.Contains(args, otherID) {
		t.Fatalf("wrong binding arguments: %q", args)
	}
	if err := os.Remove(log); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []policy.TokenRequestOptions{{TenantID: guestTenant}, {Claims: "secret-claims"}} {
		_, err := c.GetToken(context.Background(), opts)
		assertFailure(t, err, "login_required")
		if _, err := os.Stat(log); !os.IsNotExist(err) {
			t.Fatal("challenge executed CLI")
		}
	}
	// An expired login may write arbitrary sensitive stderr. Only fixed guidance escapes.
	if err := os.WriteFile(filepath.Join(dir, "az"), []byte("#!/bin/sh\nprintf '%s' 'Bearer secret-canary' >&2\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err = c.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}})
	assertFailure(t, err, "login_required")
	if strings.Contains(err.Error(), "secret-canary") {
		t.Fatal("CLI stderr leaked through credential failure")
	}
}
