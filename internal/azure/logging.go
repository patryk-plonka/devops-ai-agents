package azure

import (
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
)

var requestLog = regexp.MustCompile(`^==> OUTGOING REQUEST \(Try=([0-9]+)\)\n`)
var responseLog = regexp.MustCompile(`^==> REQUEST/RESPONSE \(Try=([0-9]+)/([^,\n]+), OpTime=([^\n)]+)\) -- (RESPONSE RECEIVED|REQUEST ERROR)\n`)
var statusLog = regexp.MustCompile(`^response ([1-5][0-9]{2})$`)
var delayLog = regexp.MustCompile(`^End Try #([0-9]+), Delay=([^\n]+)$`)

// ConfigureDebugLogging overrides the environment-installed SDK listener. It must
// be configured before SDK operations; the CLI supports one run at a time.
func ConfigureDebugLogging(enabled bool, output io.Writer) {
	azlog.SetListener(nil)
	if !enabled || output == nil {
		return
	}
	azlog.SetEvents(azlog.EventRequest, azlog.EventResponse, azlog.EventRetryPolicy)
	var mu sync.Mutex
	azlog.SetListener(func(event azlog.Event, message string) {
		if summary := debugSummary(event, message); summary != "" {
			mu.Lock()
			defer mu.Unlock()
			_, _ = io.WriteString(output, summary)
		}
	})
}

// Extract only SDK-generated numeric fields and fixed outcomes. Never forward a
// raw message: even Retry and Response events can contain bodies or credentials.
// Unknown formats fail closed, including after an SDK upgrade.
func debugSummary(event azlog.Event, message string) string {
	switch event {
	case azlog.EventRequest:
		if fields := requestLog.FindStringSubmatch(message); fields != nil {
			return fmt.Sprintf("debug: azure request attempt=%s\n", fields[1])
		}
	case azlog.EventResponse:
		if fields := responseLog.FindStringSubmatch(message); fields != nil {
			duration, err := time.ParseDuration(fields[2])
			if err != nil {
				return ""
			}
			total, err := time.ParseDuration(fields[3])
			if err != nil {
				return ""
			}
			outcome := "response_received"
			if fields[4] == "REQUEST ERROR" {
				outcome = "request_failed"
			}
			return fmt.Sprintf("debug: azure response attempt=%s duration=%s total=%s result=%s\n", fields[1], duration, total, outcome)
		}
	case azlog.EventRetryPolicy:
		if fields := statusLog.FindStringSubmatch(message); fields != nil {
			return fmt.Sprintf("debug: azure status=%s\n", fields[1])
		}
		if fields := delayLog.FindStringSubmatch(message); fields != nil {
			delay, err := time.ParseDuration(fields[2])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("debug: azure retry after_attempt=%s delay=%s\n", fields[1], delay)
		}
	}
	return ""
}
