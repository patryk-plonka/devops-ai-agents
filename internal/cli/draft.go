package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

const draftUsage = `Usage: devops-ai-agents health draft --report <path> --type ticket|recommendation --finding <id> [--finding <id> ...]
Reads a reviewed schema 4 report offline and prints one draft path per selected finding.
Drafts are saved beside the report in drafts/. Repeating a request preserves existing edits.
Exit codes: 0 drafts delivered (or help), 2 invalid request or reviewed report,
            1 storage, publication or output failure, 130 interrupted.
`

func (a App) runDraft(ctx context.Context, args []string) int {
	flags := flag.NewFlagSet("health draft", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("report", "", "path to the reviewed report.md")
	draftType := flags.String("type", "", "ticket or recommendation")
	var ids []string
	flags.Func("finding", "full finding ID from the report (repeatable)", func(id string) error {
		if !health.ValidFindingID(id) {
			return errors.New("invalid finding ID")
		}
		ids = append(ids, id)
		return nil
	})
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if _, err := fmt.Fprint(a.Out, draftUsage); err != nil {
				fmt.Fprintln(a.Err, "Could not write draft help to stdout.")
				if ctx.Err() != nil {
					return 130
				}
				return 1
			}
			return 0
		}
		fmt.Fprint(a.Err, "Invalid draft flags. Supply a full finding ID for each selection.\n", draftUsage)
		return 2
	}
	kind := report.DraftType(*draftType)
	if strings.TrimSpace(*path) == "" || !kind.Valid() || len(ids) == 0 || flags.NArg() != 0 {
		fmt.Fprint(a.Err, "Supply a reviewed report, an explicit draft type and at least one finding ID.\n", draftUsage)
		return 2
	}

	interrupted := func() bool {
		if ctx.Err() == nil {
			return false
		}
		fmt.Fprintln(a.Err, "Draft request interrupted.")
		return true
	}
	if interrupted() {
		return 130
	}
	store, run, err := runstore.OpenReviewedRun(a.Directory, *path)
	if err != nil {
		if interrupted() {
			return 130
		}
		if errors.Is(err, runstore.ErrInvalidReviewedRun) {
			fmt.Fprintln(a.Err, "Cannot use the reviewed report.", err)
			return 2
		}
		fmt.Fprintln(a.Err, "Could not read the reviewed run from local storage.")
		return 1
	}
	defer store.Close()
	selection, err := run.ResolveDraftSelection(ids)
	if interrupted() {
		return 130
	}
	if err != nil {
		fmt.Fprintln(a.Err, "Cannot select findings.", err)
		return 2
	}

	// Prepare the whole validated batch before touching the output directory.
	available := 0
	pending := func() {
		if available < len(selection.Findings) {
			fmt.Fprintln(a.Err, "Pending selections:")
			for _, item := range selection.Findings[available:] {
				fmt.Fprintln(a.Err, item.Finding.ID)
			}
		}
	}
	contents := make([][]byte, 0, len(selection.Findings))
	for _, item := range selection.Findings {
		if interrupted() {
			pending()
			return 130
		}
		markdown, err := report.DraftMarkdown(kind, selection, item)
		if err != nil {
			fmt.Fprintln(a.Err, "Cannot render selected finding.", err)
			return 2
		}
		contents = append(contents, markdown)
	}

	drafts, err := store.OpenDrafts(ctx)
	if err != nil {
		if interrupted() {
			pending()
			return 130
		}
		fmt.Fprintln(a.Err, "Could not open the private drafts directory.")
		pending()
		return 1
	}
	defer drafts.Close()
	for i, item := range selection.Findings {
		path, created, err := drafts.Publish(ctx, kind, item.Finding.ID, contents[i])
		if err != nil {
			if interrupted() {
				pending()
				return 130
			}
			fmt.Fprintf(a.Err, "Could not publish draft for %s.\n", item.Finding.ID)
			pending()
			return 1
		}
		available++
		if created {
			fmt.Fprintln(a.Err, "Draft created:", path)
		} else {
			fmt.Fprintln(a.Err, "Draft already exists:", path)
		}
		if _, err := fmt.Fprintln(a.Out, path); err != nil {
			fmt.Fprintf(a.Err, "Could not write the draft path to stdout. The draft remains available at %s.\n", path)
			pending()
			if interrupted() {
				return 130
			}
			return 1
		}
		if interrupted() {
			pending()
			return 130
		}
	}
	return 0
}
