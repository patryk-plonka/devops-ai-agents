package health

import (
	"context"
	"errors"
)

// Check owns source state and checkpoints. Adapters neither persist data nor
// choose the run outcome. A Save failure is fatal and prevents further reads.
func Check(ctx context.Context, run *Run, inventory, resourceHealth Collector, save func(*Run) error) (int, error) {
	for _, item := range []struct {
		name    string
		collect Collector
	}{{"inventory", inventory}, {"resource_health", resourceHealth}} {
		if ctx.Err() != nil {
			run.Outcome = "cancelled"
			run.RebuildCoverage()
			return 130, nil
		}
		run.Sources[item.name] = Source{Status: "in_progress"}
		run.RebuildCoverage()
		if err := save(run); err != nil {
			return 1, err
		}
		if ctx.Err() != nil {
			run.Sources[item.name] = Source{Status: "not_checked", Reason: "not_attempted"}
			run.Outcome = "cancelled"
			run.RebuildCoverage()
			return 130, nil
		}
		var storageErr error
		err := item.collect(ctx, run.Scope.SubscriptionID, func(page Page) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			run.NormalizePage(item.name, page)
			storageErr = save(run)
			return storageErr
		})
		if storageErr != nil {
			return 1, storageErr
		}
		source := run.Sources[item.name]
		if ctx.Err() != nil {
			source.Status, source.Reason = "partial", "interrupted"
		} else if err != nil {
			source.Status = "failed"
			if source.Pages > 0 {
				source.Status = "partial"
			}
			source.Reason = collectionReason(err)
			run.Diagnostics = append(run.Diagnostics, item.name+":"+source.Reason)
		} else if source.Reason != "" {
			source.Status = "partial"
		} else {
			source.Status = "completed"
		}
		run.Sources[item.name] = source
		run.RebuildCoverage()
		if err := save(run); err != nil {
			return 1, err
		}
	}
	if ctx.Err() != nil {
		run.Outcome = "cancelled"
		return 130, nil
	}
	if run.Sources["inventory"].Status == "completed" && run.Sources["resource_health"].Status == "completed" {
		run.Outcome = "completed"
		return 0, nil
	}
	if len(run.Evidence) > 0 || run.Sources["inventory"].Status == "completed" || run.Sources["resource_health"].Status == "completed" {
		run.Outcome = "partial"
		return 3, nil
	}
	run.Outcome = "failed"
	return 1, nil
}
func collectionReason(err error) string {
	var diagnostic interface{ Diagnostic() string }
	if errors.As(err, &diagnostic) {
		// Only category identifiers cross the boundary, never arbitrary error text.
		switch reason := diagnostic.Diagnostic(); reason {
		case "login_required", "access_denied", "source_unavailable", "unsupported", "timeout", "throttled", "invalid_response", "unsafe_continuation", "repeated_continuation", "page_limit", "read_boundary_rejected":
			return reason
		}
	}
	return "collection_failed"
}
