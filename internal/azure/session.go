// Package azure contains the read-only public Azure adapters.
package azure

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

const OperationTimeout = 60 * time.Second

var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Failure struct {
	Category string
	Message  string
	Code     int
}

func (e *Failure) Error() string                    { return e.Message }
func fail(category, message string, code int) error { return &Failure{category, message, code} }

type Account struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TenantID    string `json:"tenantId"`
	Environment string `json:"cloudName"`
	User        struct {
		Type string `json:"type"`
	} `json:"user"`
}

type Command func(context.Context, ...string) ([]byte, error)
type Session struct{ Command Command }

func runAZ(ctx context.Context, args ...string) ([]byte, error) {
	// Output captures stdout only. Never forward child stderr or the CLI profile.
	return exec.CommandContext(ctx, "az", args...).Output()
}

func (s Session) Discover(ctx context.Context) ([]Account, error) {
	command := s.Command
	if command == nil {
		command = runAZ
	}
	call := func(args ...string) ([]byte, error) {
		bounded, cancel := context.WithTimeout(ctx, OperationTimeout)
		defer cancel()
		return command(bounded, args...)
	}
	b, err := call("cloud", "show", "--output", "json", "--only-show-errors")
	if err != nil {
		return nil, fail("session_unavailable", "Cannot read Azure CLI session; run az login externally and retry.", 1)
	}
	var cloud struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(b, &cloud) != nil || cloud.Name == "" {
		return nil, fail("invalid_session", "Azure CLI returned invalid cloud metadata.", 1)
	}
	if cloud.Name != "AzureCloud" {
		return nil, fail("unsupported_cloud", "Only the AzureCloud public cloud is supported.", 2)
	}
	b, err = call("account", "list", "--all", "--output", "json", "--only-show-errors")
	if err != nil {
		return nil, fail("session_unavailable", "Cannot read cached Azure CLI accounts; run az login externally and retry.", 1)
	}
	var accounts []Account
	if json.Unmarshal(b, &accounts) != nil {
		return nil, fail("invalid_session", "Azure CLI returned invalid account metadata.", 1)
	}
	return accounts, nil
}

// Matches uses original names for exact matching; presentation is filtered later.
func Matches(accounts []Account, requested string) ([]Account, error) {
	byID := map[string]Account{}
	unsupported := false
	for _, a := range accounts {
		if !uuid.MatchString(a.ID) || strings.EqualFold(a.ID, a.TenantID) {
			continue
		}
		matches := strings.EqualFold(a.Name, requested)
		if uuid.MatchString(requested) {
			matches = strings.EqualFold(a.ID, requested)
		}
		if !matches {
			continue
		}
		if a.Environment != "AzureCloud" || !strings.EqualFold(a.User.Type, "user") {
			unsupported = true
			continue
		}
		if !uuid.MatchString(a.TenantID) {
			return nil, fail("invalid_session", "Selected account has invalid tenant metadata; sign in again externally.", 1)
		}
		a.ID = strings.ToLower(a.ID)
		a.TenantID = strings.ToLower(a.TenantID)
		if previous, exists := byID[a.ID]; exists && previous != a {
			return nil, fail("invalid_session", "Conflicting cached subscription metadata; refresh the login externally.", 1)
		}
		byID[a.ID] = a
	}
	out := make([]Account, 0, len(byID))
	for _, a := range byID {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) == 0 {
		if unsupported {
			return nil, fail("unsupported_account", "The selected subscription requires an AzureCloud user login.", 2)
		}
		return nil, fail("subscription_not_in_session", "Subscription is missing from this CLI session; sign in or refresh externally, then supply its full ID or name.", 2)
	}
	return out, nil
}

func (a Account) Scope() (health.Scope, int) {
	name, n := health.SafeText(a.Name)
	return health.Scope{SubscriptionID: a.ID, Name: name, TenantID: a.TenantID, Cloud: "AzureCloud"}, n
}
