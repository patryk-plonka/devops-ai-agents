// Package cli implements the operator interaction with injectable external boundaries.
package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

type Storage interface {
	Save(*health.Run) error
	Path() string
	PublishReport(context.Context, func(health.Run) ([]byte, error)) error
	ReportPath() string
	ReportExists() (bool, error)
	Close() error
}
type App struct {
	In                          io.Reader
	Out, Err                    io.Writer
	Terminal                    bool
	Directory, Version          string
	Discover                    func(context.Context) ([]azure.Account, error)
	Validate                    func(context.Context, azure.Account) error
	NewStore                    func(string, *health.Run) (Storage, error)
	Inventory, ResourceHealth   health.Collector
	ServiceHealth, AzureMonitor health.Collector
	Advisor                     health.Collector
	Render                      func(health.Run) ([]byte, error)
}

const commands = `Usage: devops-ai-agents health <check|draft> [flags]
Use health check --help or health draft --help for command details.
`

const usage = `Usage: devops-ai-agents health check --subscription <id-or-full-name> [--non-interactive] [--debug]
--debug: print filtered Azure SDK request, status, timing, and retry diagnostics to stderr.
Saves Markdown and evidence with coverage and resource counts. OK applies within the saved checks and query windows.
Exit codes: 0 all collectors completed and report delivered (or help), 3 partial execution with useful results,
            2 invalid input or subscription selection, 1 execution or delivery failure, 130 interrupted.
`

func (a App) Run(ctx context.Context, args []string) int {
	if a.In == nil {
		a.In = strings.NewReader("")
	}
	if a.Out == nil {
		a.Out = io.Discard
	}
	if a.Err == nil {
		a.Err = io.Discard
	}
	help := func() int { fmt.Fprint(a.Out, usage); return 0 }
	if len(args) == 1 && isHelp(args[0]) || len(args) == 2 && args[0] == "health" && isHelp(args[1]) {
		fmt.Fprint(a.Out, commands)
		return 0
	}
	if len(args) >= 2 && args[0] == "health" && args[1] == "draft" {
		return a.runDraft(ctx, args[2:])
	}
	if len(args) < 2 || args[0] != "health" || args[1] != "check" {
		fmt.Fprint(a.Err, commands)
		return 2
	}
	// SDK package initialization can enable logging from the environment. Override
	// it before collection, then require an explicit flag for filtered output.
	azure.ConfigureDebugLogging(false, nil)
	flags := flag.NewFlagSet("health check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	requested := flags.String("subscription", "", "subscription UUID or full name")
	nonInteractive := flags.Bool("non-interactive", false, "fail on ambiguity")
	debug := flags.Bool("debug", false, "print filtered Azure SDK diagnostics")
	if err := flags.Parse(args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return help()
		}
		fmt.Fprint(a.Err, "Invalid flags.\n", usage)
		return 2
	}
	if strings.TrimSpace(*requested) == "" || flags.NArg() != 0 {
		fmt.Fprint(a.Err, "Supply one subscription and no extra arguments.\n", usage)
		return 2
	}
	azure.ConfigureDebugLogging(*debug, a.Err)
	defer azure.ConfigureDebugLogging(false, nil)
	root, err := runstore.FindRepository(a.Directory)
	if err != nil {
		fmt.Fprintln(a.Err, "Repository context missing; run from this project's repository tree.")
		return 2
	}
	newStore := a.NewStore
	if newStore == nil {
		newStore = func(root string, run *health.Run) (Storage, error) { return runstore.New(root, run) }
	}
	run := health.NewRun(*requested, a.Version)
	store, err := newStore(root, &run)
	if err != nil {
		fmt.Fprintln(a.Err, "Cannot initialize local run storage; no artifact could be saved and no Azure requests were made.")
		if ctx.Err() != nil {
			return 130
		}
		return 1
	}
	defer store.Close()
	retained := func() {
		fmt.Fprintf(a.Err, "Last saved snapshot: %s (revision %d). ", store.Path(), run.Revision)
		exists, err := store.ReportExists()
		switch {
		case err != nil:
			fmt.Fprintln(a.Err, "Could not determine whether an older report exists; no current report was published.")
		case exists:
			fmt.Fprintf(a.Err, "An older report exists at %s; it is not current. Compare its run ID/revision with run.json.\n", store.ReportPath())
		default:
			fmt.Fprintln(a.Err, "No older report exists; no report was saved.")
		}
	}
	var finish func(string, string, string, int) int
	finish = func(outcome, diagnostic, message string, code int) int {
		if ctx.Err() != nil {
			outcome, diagnostic, message, code = "cancelled", "interrupted", "Run cancelled; overall coverage is incomplete.", 130
		}
		now := time.Now().UTC()
		run.EndedAt = &now
		run.Outcome = outcome
		if diagnostic != "" {
			run.Diagnostics = append(run.Diagnostics, diagnostic)
		}
		run.RebuildCoverage()
		if err := store.Save(&run); err != nil {
			fmt.Fprintln(a.Err, "Final run write failed; the previous checkpoint remains available.")
			retained()
			if code == 130 || ctx.Err() != nil {
				fmt.Fprintln(a.Err, "Run cancelled; finalization did not complete.")
				return 130
			}
			return 1
		}
		render := a.Render
		if render == nil {
			render = report.Markdown
		}
		publishCtx := ctx
		if run.Outcome == "cancelled" {
			publishCtx = context.WithoutCancel(ctx)
		}
		err := store.PublishReport(publishCtx, render)
		if err != nil && ctx.Err() != nil && run.Outcome != "cancelled" {
			return finish("cancelled", "interrupted", "Run cancelled; overall coverage is incomplete.", 130)
		}
		if err != nil {
			fmt.Fprintln(a.Err, "Report rendering/publication failed.")
			// Preserve the publication failure in JSON when storage is still usable.
			// An older report remains a separate, explicitly stale artifact.
			if ctx.Err() != nil {
				code = 130
			}
			run.Outcome = "failed"
			if code == 130 {
				run.Outcome = "cancelled"
			}
			run.Diagnostics = append(run.Diagnostics, "report_publication_failed")
			run.RebuildCoverage()
			saveErr := store.Save(&run)
			if saveErr == nil && run.Outcome != "cancelled" && ctx.Err() != nil {
				run.Outcome = "cancelled"
				run.Diagnostics = append(run.Diagnostics, "interrupted")
				run.RebuildCoverage()
				saveErr = store.Save(&run)
			}
			if saveErr != nil {
				fmt.Fprintln(a.Err, "Could not save the report-publication failure diagnostic.")
			}
			retained()
			if code == 130 || ctx.Err() != nil {
				fmt.Fprintln(a.Err, "Run cancelled; finalization did not complete.")
				return 130
			}
			return 1
		}
		fmt.Fprintln(a.Err, message)
		fmt.Fprintf(a.Err, "Execution: %s, overall coverage: %s. %s. Report saved: %s\nEvidence snapshot: %s (revision %d).\n",
			run.Outcome, run.Coverage, run.AssessmentSummary.ResourceCounts(), store.ReportPath(), store.Path(), run.Revision)
		if _, err := fmt.Fprintln(a.Out, store.ReportPath()); err != nil {
			fmt.Fprintf(a.Err, "Could not write the saved report path to stdout. The report remains available at %s.\n", store.ReportPath())
			if code == 130 || ctx.Err() != nil {
				fmt.Fprintln(a.Err, "Run cancelled; the saved report remains available.")
				return 130
			}
			return 1
		}
		if ctx.Err() != nil {
			fmt.Fprintln(a.Err, "Run cancelled; the saved report remains available.")
			return 130
		}
		return code
	}
	failure := func(err error) int {
		var f *azure.Failure
		if errors.As(err, &f) {
			return finish("failed", f.Category, f.Message, f.Code)
		}
		return finish("failed", "execution_failed", "Scope validation failed; check the existing Azure CLI login.", 1)
	}
	if ctx.Err() != nil {
		return finish("cancelled", "interrupted", "Run cancelled.", 130)
	}
	discover := a.Discover
	if discover == nil {
		discover = (azure.Session{}).Discover
	}
	accounts, err := discover(ctx)
	if err != nil {
		return failure(err)
	}
	matches, err := azure.Matches(accounts, *requested)
	if err != nil {
		return failure(err)
	}
	chosen := matches[0]
	if len(matches) > 1 {
		if *nonInteractive || !a.Terminal {
			return failure(&azure.Failure{Category: "ambiguous_subscription", Message: "Ambiguous subscription name; supply its UUID, or use terminal input to select.", Code: 2})
		}
		for i, account := range matches {
			scope, _ := account.Scope()
			fmt.Fprintf(a.Err, "%d) %s | %s | tenant %s\n", i+1, scope.Name, scope.SubscriptionID, scope.TenantID)
		}
		fmt.Fprint(a.Err, "Select subscription number (blank cancels): ")
		line, err := readSelection(ctx, a.In)
		if err != nil || strings.TrimSpace(line) == "" {
			return failure(&azure.Failure{Category: "selection_cancelled", Message: "Subscription selection cancelled; no subscription was selected.", Code: 2})
		}
		index, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || index < 1 || index > len(matches) {
			return failure(&azure.Failure{Category: "invalid_selection", Message: "Invalid subscription selection; retry with its UUID.", Code: 2})
		}
		chosen = matches[index-1]
	}
	if ctx.Err() != nil {
		return finish("cancelled", "interrupted", "Run cancelled.", 130)
	}
	scope, redactions := chosen.Scope()
	run.Scope = &scope
	run.Redactions += redactions
	fmt.Fprintln(a.Err, "Validating live access to the selected subscription...")
	validate := a.Validate
	if validate == nil {
		validate = (azure.Validator{}).Validate
	}
	if err := validate(ctx, chosen); err != nil {
		return failure(err)
	}
	if ctx.Err() != nil {
		return finish("cancelled", "interrupted", "Run cancelled.", 130)
	}
	serviceQuery := health.ServiceHealthQuery(run.StartedAt)
	for name, query := range map[string]health.SourceQuery{
		"service_health": serviceQuery,
		"azure_monitor":  health.MonitorQuery(run.StartedAt),
		"advisor":        health.AdvisorQuery(run.StartedAt),
	} {
		source := run.Sources[name]
		source.Query = &query
		run.Sources[name] = source
	}
	collection := azure.Collection{ServiceHealthQuery: serviceQuery}
	inventory, resourceHealth := a.Inventory, a.ResourceHealth
	serviceHealth, monitor := a.ServiceHealth, a.AzureMonitor
	advisor := a.Advisor
	if inventory == nil {
		inventory = collection.Inventory
	}
	if resourceHealth == nil {
		resourceHealth = collection.ResourceHealth
	}
	if serviceHealth == nil {
		serviceHealth = collection.ServiceHealth
	}
	if monitor == nil {
		monitor = collection.AzureMonitor
	}
	if advisor == nil {
		advisor = collection.Advisor
	}
	collectors := []health.NamedCollector{
		{Name: "inventory", Collect: inventory}, {Name: "resource_health", Collect: resourceHealth},
		{Name: "service_health", Collect: serviceHealth}, {Name: "azure_monitor", Collect: monitor},
		{Name: "advisor", Collect: advisor},
	}
	fmt.Fprintln(a.Err, "Collecting inventory, Resource Health, Service Health, Azure Monitor, and Azure Advisor evidence...")
	code, err := health.Check(ctx, &run, collectors, store.Save)
	if err != nil {
		fmt.Fprintln(a.Err, "Evidence checkpoint write failed; collection stopped.")
		retained()
		if ctx.Err() != nil {
			fmt.Fprintln(a.Err, "Run cancelled; finalization did not complete.")
			return 130
		}
		return 1
	}
	for _, item := range collectors {
		source := run.Sources[item.Name]
		fmt.Fprintf(a.Err, "%s: %s; pages=%d; rejected=%d; excluded=%d; unclassified=%d; reason=%s\n", item.Name, source.Status, source.Pages, source.RejectedRecords, source.ExcludedRecords, source.UnclassifiedRecords, source.Reason)
	}
	return finish(run.Outcome, "", "Evidence collection finished.", code)
}

// A buffered result lets cancellation finalize local artifacts even while the
// terminal reader is blocked. The reader never touches run state or output and
// exits on its next input (or with the process); it never chooses a default.
func readSelection(ctx context.Context, in io.Reader) (string, error) {
	type result struct {
		line string
		err  error
	}
	ready := make(chan result, 1)
	go func() { line, err := bufio.NewReader(in).ReadString('\n'); ready <- result{line, err} }()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ready:
		return r.line, r.err
	}
}

func isHelp(s string) bool { return s == "--help" || s == "-h" }
