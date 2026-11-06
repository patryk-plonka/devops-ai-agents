package azure

import (
	"strings"
	"testing"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
)

func TestDebugSummaryExcludesUntrustedSDKText(t *testing.T) {
	for _, tc := range []struct {
		name          string
		event         azlog.Event
		message, want string
	}{
		{"request", azlog.EventRequest, "==> OUTGOING REQUEST (Try=2)\n   GET https://example.com/token-canary\nAuthorization: Bearer token-canary", "debug: azure request attempt=2\n"},
		{"response", azlog.EventResponse, "==> REQUEST/RESPONSE (Try=2/1ms, OpTime=3ms) -- RESPONSE RECEIVED\nX-Ms-Request-Id: header-canary\nbody-canary", "debug: azure response attempt=2 duration=1ms total=3ms result=response_received\n"},
		{"request failure", azlog.EventResponse, "==> REQUEST/RESPONSE (Try=1/1ms, OpTime=1ms) -- REQUEST ERROR\nBearer token-canary", "debug: azure response attempt=1 duration=1ms total=1ms result=request_failed\n"},
		{"status", azlog.EventRetryPolicy, "response 403", "debug: azure status=403\n"},
		{"delay", azlog.EventRetryPolicy, "End Try #2, Delay=1.5s", "debug: azure retry after_attempt=2 delay=1.5s\n"},
		{"raw response error", azlog.EventResponseError, "body-canary", ""},
		{"raw retry error", azlog.EventRetryPolicy, "error Bearer token-canary", ""},
		{"invalid duration", azlog.EventResponse, "==> REQUEST/RESPONSE (Try=1/body-canary, OpTime=1ms) -- RESPONSE RECEIVED\n", ""},
		{"invalid retry delay", azlog.EventRetryPolicy, "End Try #2, Delay=token-canary", ""},
		{"status with extra content", azlog.EventRetryPolicy, "response 403\nbody-canary", ""},
		{"unknown format", azlog.EventRequest, "body-canary", ""},
		{"unknown event", azlog.Event("Authentication"), "Bearer token-canary", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := debugSummary(tc.event, tc.message)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			if strings.Contains(got, "canary") {
				t.Fatal("untrusted SDK text leaked")
			}
		})
	}
}
