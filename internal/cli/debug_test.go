package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
)

type debugCredential struct{}

func (debugCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "token-secret-canary", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type debugTransport struct{}

func (debugTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusForbidden, Status: "403 Forbidden",
		Header: http.Header{"Content-Type": {"application/json"}, "X-Ms-Request-Id": {"header-secret-canary"}},
		Body:   io.NopCloser(strings.NewReader(`{"error":{"code":"AuthorizationFailed","message":"body-secret-canary"}}`)),
	}, nil
}

// A fresh process is required: the SDK reads its logging environment in init.
func TestDebugLoggingSubprocess(t *testing.T) {
	if os.Getenv("DEVOPS_DEBUG_TEST_CHILD") == "1" {
		a, _, _, _ := testApp(t)
		a.Out, a.Err = os.Stdout, os.Stderr
		a.Validate = (azure.Validator{
			Credential: func(string) (azcore.TokenCredential, error) { return debugCredential{}, nil },
			Base:       debugTransport{},
		}).Validate
		args := []string{"health", "check", "--subscription", "Client"}
		if os.Getenv("DEVOPS_DEBUG_TEST_FLAG") == "1" {
			args = append(args, "--debug")
		}
		fmt.Fprintf(os.Stdout, "app_exit=%d\n", a.Run(context.Background(), args))
		return
	}
	for _, tc := range []struct {
		name             string
		debug, inherited bool
	}{
		{"default", false, false},
		{"inherited SDK logging", false, true},
		{"debug flag", true, false},
		{"debug with inherited SDK logging", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DEVOPS_DEBUG_TEST_CHILD", "1")
			t.Setenv("DEVOPS_DEBUG_TEST_FLAG", "0")
			t.Setenv("AZURE_SDK_GO_LOGGING", "")
			if tc.debug {
				t.Setenv("DEVOPS_DEBUG_TEST_FLAG", "1")
			}
			if tc.inherited {
				t.Setenv("AZURE_SDK_GO_LOGGING", "all")
			}
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, "-test.run=^TestDebugLoggingSubprocess$")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("child failed: %v", err)
			}
			if !strings.Contains(stdout.String(), "app_exit=1") || !strings.Contains(stderr.String(), "Live access to the selected subscription was denied.") {
				t.Fatal("debug flag changed the access-failure result")
			}
			if strings.Contains(stdout.String()+stderr.String(), "secret-canary") {
				t.Fatal("SDK body, header, or token reached process output")
			}
			if strings.Contains(stdout.String(), "debug:") {
				t.Fatal("debug diagnostics reached stdout")
			}
			if tc.debug {
				for _, expected := range []string{"debug: azure request attempt=1", "debug: azure response attempt=1", "debug: azure status=403"} {
					if !strings.Contains(stderr.String(), expected) {
						t.Errorf("missing diagnostic %q", expected)
					}
				}
			} else if strings.Contains(stderr.String(), "debug:") || strings.Contains(stderr.String(), "ResponseError:") {
				t.Fatal("SDK logging enabled without --debug")
			}
		})
	}
}
