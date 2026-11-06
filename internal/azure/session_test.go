package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const selectedID = "11111111-1111-1111-1111-111111111111"
const otherID = "22222222-2222-2222-2222-222222222222"
const guestTenant = "33333333-3333-3333-3333-333333333333"

func account(id, name string) Account {
	a := Account{ID: id, Name: name, TenantID: guestTenant, Environment: "AzureCloud"}
	a.User.Type = "user"
	return a
}

func TestMatches(t *testing.T) {
	a, b := account(selectedID, "Client"), account(otherID, "client")
	service := a
	service.User.Type = "servicePrincipal"
	sovereign := a
	sovereign.Environment = "AzureUSGovernment"
	placeholder := a
	placeholder.ID = guestTenant
	conflicting := a
	conflicting.TenantID = otherID
	for _, tc := range []struct {
		name, input string
		accounts    []Account
		ids         []string
		category    string
	}{
		{"ID", strings.ToUpper(selectedID), []Account{b, a}, []string{selectedID}, ""},
		{"name", "CLIENT", []Account{b, a, a}, []string{selectedID, otherID}, ""},
		{"no substring", "Cli", []Account{a}, nil, "subscription_not_in_session"},
		{"missing", "Other", []Account{a}, nil, "subscription_not_in_session"},
		{"no cache", "Client", nil, nil, "subscription_not_in_session"},
		{"service principal", "Client", []Account{service}, nil, "unsupported_account"},
		{"cloud", "Client", []Account{sovereign}, nil, "unsupported_account"},
		{"placeholder", "Client", []Account{placeholder}, nil, "subscription_not_in_session"},
		{"conflicting duplicates", "Client", []Account{a, conflicting}, nil, "invalid_session"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Matches(tc.accounts, tc.input)
			if tc.category != "" {
				assertFailure(t, err, tc.category)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			ids := []string{}
			for _, a := range got {
				ids = append(ids, a.ID)
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("IDs %v, want %v", ids, tc.ids)
			}
		})
	}
}

func assertFailure(t *testing.T, err error, category string) {
	t.Helper()
	var f *Failure
	if !errors.As(err, &f) || f.Category != category {
		t.Fatalf("failure %v, want %s", err, category)
	}
}

func TestDiscoverUsesOnlyCachedReadCommands(t *testing.T) {
	var calls [][]string
	s := Session{Command: func(ctx context.Context, args ...string) ([]byte, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded discovery")
		}
		calls = append(calls, args)
		if args[0] == "cloud" {
			return []byte(`{"name":"AzureCloud"}`), nil
		}
		return json.Marshal([]Account{account(selectedID, "client")})
	}}
	got, err := s.Discover(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	want := [][]string{{"cloud", "show", "--output", "json", "--only-show-errors"}, {"account", "list", "--all", "--output", "json", "--only-show-errors"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("commands %v", calls)
	}
}

func TestDiscoverDecodesAzureCLIAccountSchema(t *testing.T) {
	// Keep this independent of Account's JSON tags: az account list emits
	// cloudName; serializing Account here would reproduce a wrong tag in fixtures.
	public := fmt.Sprintf(`[{"id":%q,"name":"Client","tenantId":%q,"cloudName":"AzureCloud","user":{"type":"user","name":"profile-canary"}}]`, selectedID, guestTenant)
	for _, tc := range []struct {
		name, body, category string
	}{
		{"public user", public, ""},
		{"sovereign cloud", strings.Replace(public, `"cloudName":"AzureCloud"`, `"cloudName":"AzureUSGovernment"`, 1), "unsupported_account"},
		{"missing cloud", strings.Replace(public, `"cloudName":"AzureCloud",`, "", 1), "unsupported_account"},
		{"profile field is not CLI output", strings.Replace(public, `"cloudName"`, `"environmentName"`, 1), "unsupported_account"},
		{"service principal", strings.Replace(public, `"type":"user"`, `"type":"servicePrincipal"`, 1), "unsupported_account"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := Session{Command: func(_ context.Context, args ...string) ([]byte, error) {
				if args[0] == "cloud" {
					return []byte(`{"name":"AzureCloud"}`), nil
				}
				return []byte(tc.body), nil
			}}
			accounts, err := s.Discover(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, requested := range []string{selectedID, "CLIENT"} {
				matches, err := Matches(accounts, requested)
				if tc.category != "" {
					assertFailure(t, err, tc.category)
					continue
				}
				if err != nil || len(matches) != 1 || matches[0].ID != selectedID || matches[0].Environment != "AzureCloud" || matches[0].TenantID != guestTenant {
					t.Fatalf("CLI account did not resolve by %q: %v", requested, err)
				}
			}
		})
	}
}

func TestDiscoveryFailuresAreSanitized(t *testing.T) {
	for _, tc := range []struct {
		name, cloud, list, category string
		processError                bool
	}{
		{"no login", "", "", "session_unavailable", true},
		{"invalid cloud", "token-canary", "", "invalid_session", false},
		{"unsupported cloud", `{"name":"AzureChinaCloud"}`, "", "unsupported_cloud", false},
		{"invalid accounts", `{"name":"AzureCloud"}`, "token-canary", "invalid_session", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := Session{Command: func(_ context.Context, args ...string) ([]byte, error) {
				if tc.processError {
					return nil, errors.New("token-canary")
				}
				if args[0] == "cloud" {
					return []byte(tc.cloud), nil
				}
				return []byte(tc.list), nil
			}}
			_, err := s.Discover(context.Background())
			assertFailure(t, err, tc.category)
			if strings.Contains(err.Error(), "token-canary") {
				t.Fatal("raw error leaked")
			}
		})
	}
}
