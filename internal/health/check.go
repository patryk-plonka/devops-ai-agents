package health

import (
	"context"
	"errors"
	"time"
)

type NamedCollector struct {
	Name    string
	Collect Collector
}

// Check owns source state and checkpoints. Adapters neither persist data nor
// choose the run outcome. A Save failure is fatal and prevents further reads.
func Check(ctx context.Context, run *Run, collectors []NamedCollector, save func(*Run) error) (int, error) {
	seen := make(map[string]bool, len(collectors))
	for _, item := range collectors {
		if item.Name == "" || item.Collect == nil || seen[item.Name] {
			run.Outcome = "failed"
			run.RebuildCoverage()
			return 1, errors.New("invalid collector configuration")
		}
		seen[item.Name] = true
	}
	for _, item := range collectors {
		source := run.Sources[item.Name]
		source.Status, source.Reason = "not_checked", "not_attempted"
		if q := source.Query; q != nil && q.RequestedAt != nil {
			configured := *q
			configured.RequestedAt = nil
			source.Query = &configured
		}
		run.Sources[item.Name] = source
	}
	for _, item := range collectors {
		if ctx.Err() != nil {
			run.Outcome = "cancelled"
			run.RebuildCoverage()
			return 130, nil
		}
		source := run.Sources[item.Name]
		source.Status, source.Reason = "in_progress", ""
		configured := source.Query
		if configured != nil {
			query := *configured
			now := time.Now().UTC()
			query.RequestedAt = &now
			source.Query = &query
		}
		run.Sources[item.Name] = source
		run.RebuildCoverage()
		if err := save(run); err != nil {
			return 1, err
		}
		if ctx.Err() != nil {
			source.Status, source.Reason = "not_checked", "not_attempted"
			source.Query = configured
			run.Sources[item.Name] = source
			run.Outcome = "cancelled"
			run.RebuildCoverage()
			return 130, nil
		}
		var storageErr error
		err := item.Collect(ctx, run.Scope.SubscriptionID, func(page Page) error {
			if storageErr != nil {
				return storageErr
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			run.NormalizePage(item.Name, page)
			storageErr = save(run)
			return storageErr
		})
		if storageErr != nil {
			return 1, storageErr
		}
		source = run.Sources[item.Name]
		if ctx.Err() != nil {
			source.Status, source.Reason = "partial", "interrupted"
		} else if err != nil {
			source.Status = "failed"
			if source.Pages > 0 {
				source.Status = "partial"
			}
			source.Reason = collectionReason(err)
			run.Diagnostics = append(run.Diagnostics, item.Name+":"+source.Reason)
		} else if source.RejectedRecords > 0 {
			source.Status, source.Reason = "partial", "invalid_records"
		} else {
			source.Status = "completed"
		}
		run.Sources[item.Name] = source
		run.RebuildCoverage()
		if err := save(run); err != nil {
			return 1, err
		}
	}
	if ctx.Err() != nil {
		run.Outcome = "cancelled"
		return 130, nil
	}
	completed := 0
	for _, item := range collectors {
		if run.Sources[item.Name].Status == "completed" {
			completed++
		}
	}
	if len(collectors) > 0 && completed == len(collectors) {
		run.Outcome = "completed"
		return 0, nil
	}
	if len(run.Evidence) > 0 || completed > 0 {
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
