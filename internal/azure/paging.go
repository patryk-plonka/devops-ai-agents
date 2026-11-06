package azure

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const maxPages = 10000

// Collection permits dependency injection only at Go boundaries, never via a
// production endpoint/token flag. Zero values select the fixed operational bounds.
type Collection struct {
	Base       http.RoundTripper
	Credential func(string) (azcore.TokenCredential, error)
	timeout    time.Duration
	pageLimit  int
	retryDelay time.Duration
}

func (c Collection) options(subscription, route string) (*arm.ClientOptions, azcore.TokenCredential, error) {
	newCredential := c.Credential
	if newCredential == nil {
		newCredential = subscriptionCredential
	}
	credential, err := newCredential(subscription)
	options := &arm.ClientOptions{DisableRPRegistration: true}
	options.Transport = collectionTransport{ReadTransport{Subscription: subscription, Route: route, Base: c.Base}}
	options.Retry.MaxRetries = 3
	options.Retry.RetryDelay = c.retryDelay
	// A nanosecond cap keeps retry tests controlled. This SDK version treats a
	// zero calculated delay as overflow; negative RetryDelay alone sleeps 60s.
	if c.retryDelay < 0 {
		options.Retry.RetryDelay = time.Nanosecond
		options.Retry.MaxRetryDelay = time.Nanosecond
	}
	return options, credential, err
}
func (c Collection) deadline(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := c.timeout
	if timeout == 0 {
		timeout = OperationTimeout
	}
	return context.WithTimeout(ctx, timeout)
}
func (e *Failure) Diagnostic() string { return e.Category }

// Application failures describe non-transient conditions. Network failures remain
// retriable transport errors; invalid pages and login failures must not be retried.
func (e *Failure) NonRetriable() {}

type boundaryError struct{ reason string }

func (e *boundaryError) Error() string      { return "ARM read boundary rejected the request" }
func (e *boundaryError) NonRetriable()      {}
func (e *boundaryError) Diagnostic() string { return e.reason }

func collectionFailure(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fail("timeout", "Collection page deadline exceeded.", 1)
	}
	// The selected SDK formats JSON decode errors with %s, losing their Go
	// types. Recognize only its fixed prefix; never copy its payload-bearing text.
	if strings.HasPrefix(err.Error(), "unmarshalling type ") {
		return fail("invalid_response", "Source returned invalid JSON data.", 1)
	}
	var boundary *boundaryError
	if errors.As(err, &boundary) {
		return boundary
	}
	var failure *Failure
	if errors.As(err, &failure) {
		return failure
	}
	var response *azcore.ResponseError
	if errors.As(err, &response) {
		switch response.StatusCode {
		case 401:
			return fail("login_required", "Collection authentication failed; sign in externally.", 1)
		case 403:
			return fail("access_denied", "Collection access denied.", 1)
		case 429:
			return fail("throttled", "Collection retries exhausted after throttling.", 1)
		case 404:
			return fail("source_unavailable", "Collection source is unavailable.", 1)
		}
		switch response.ErrorCode {
		case "UnsupportedResourceType", "ResourceTypeNotSupported":
			return fail("unsupported", "Source explicitly reported unsupported evidence.", 1)
		}
	}
	return fail("collection_failed", "Collection failed; inspect source coverage gaps.", 1)
}

// Validate continuations before passing them to the authenticated SDK pipeline.
// Restrict them to the same operation, not just any approved subscription read.
func continuation(link, subscription, route string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", &boundaryError{"unsafe_continuation"}
	}
	req := &http.Request{Method: http.MethodGet, URL: u}
	if !allowedRequest(req, subscription, route) {
		return "", &boundaryError{"unsafe_continuation"}
	}
	q, _ := url.ParseQuery(u.RawQuery)
	if route == healthRoute && q.Has("$skipToken") {
		q.Set("$skiptoken", q.Get("$skipToken"))
		q.Del("$skipToken")
	}
	// Canonical ordering detects repeated equivalent URLs without recording tokens.
	u.RawQuery = q.Encode()
	u.Path = strings.ToLower(u.Path)
	return u.String(), nil
}

func pages[T any](ctx context.Context, c Collection, subscription, route string, pager *runtime.Pager[T], accept func(T) (*string, error)) error {
	limit := c.pageLimit
	if limit == 0 {
		limit = maxPages
	}
	path, version := "/resources", "2025-04-01"
	if route == healthRoute {
		path, version = "/providers/Microsoft.ResourceHealth/availabilityStatuses", "2025-05-01"
	}
	initial, _ := continuation("https://management.azure.com/subscriptions/"+subscription+path+"?api-version="+version, subscription, route)
	seen := map[string]bool{initial: true}
	for number := 0; pager.More(); number++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if number >= limit {
			return fail("page_limit", "Collection page ceiling reached.", 1)
		}
		bounded, cancel := c.deadline(ctx)
		if err := bounded.Err(); err != nil {
			cancel()
			return collectionFailure(err)
		}
		page, err := pager.NextPage(bounded)
		deadlineErr := bounded.Err()
		cancel()
		if err != nil {
			if deadlineErr != nil {
				return collectionFailure(deadlineErr)
			}
			return collectionFailure(err)
		}
		next, err := accept(page) // normalization and checkpoint precede next request
		if err != nil {
			return err
		}
		if next != nil && *next != "" {
			key, err := continuation(*next, subscription, route)
			if err != nil {
				return err
			}
			if seen[key] {
				return fail("repeated_continuation", "Repeated continuation stopped collection.", 1)
			}
			seen[key] = true
		}
	}
	return nil
}
