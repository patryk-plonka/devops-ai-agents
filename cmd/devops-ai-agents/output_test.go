package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
	"github.com/patryk-plonka/devops-ai-agents/internal/cli"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

func TestProcessClosedReportPathPipe(t *testing.T) {
	if root := os.Getenv("DEVOPS_TEST_CLOSED_PIPE_ROOT"); root != "" {
		account := azure.Account{ID: signalSub, Name: "Synthetic", TenantID: "22222222-2222-2222-2222-222222222222", Environment: "AzureCloud"}
		account.User.Type = "user"
		collect := func(_ context.Context, _ string, emit func(health.Page) error) error { return emit(health.Page{}) }
		app := cli.App{
			Directory: root, Out: os.Stdout, Err: os.Stderr,
			Discover:  func(context.Context) ([]azure.Account, error) { return []azure.Account{account}, nil },
			Validate:  func(context.Context, azure.Account) error { return nil },
			Inventory: collect, ResourceHealth: collect, ServiceHealth: collect, AzureMonitor: collect, Advisor: collect,
		}
		os.Exit(runApp(app, []string{"health", "check", "--subscription", signalSub}))
	}
	if runtime.GOOS == "windows" {
		t.Skip("SIGPIPE is a POSIX signal")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+runstore.Module+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer write.Close()
	if err := read.Close(); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-test.run", "^TestProcessClosedReportPathPipe$")
	cmd.Env = append(os.Environ(), "DEVOPS_TEST_CLOSED_PIPE_ROOT="+root)
	cmd.Stdout = write
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	var exit *exec.ExitError
	if err := cmd.Run(); !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("expected report delivery exit 1, got %v\n%s", err, diagnostics.String())
	}
	reports, err := filepath.Glob(filepath.Join(root, "runs", "*", "report.md"))
	if err != nil || len(reports) != 1 {
		t.Fatalf("expected one retained report, got %v, %v", reports, err)
	}
	path := reports[0]
	if !strings.Contains(diagnostics.String(), "Could not write the saved report path to stdout. The report remains available at "+path+".") {
		t.Fatalf("missing delivery diagnostic for retained report:\n%s", diagnostics.String())
	}
	snapshot, err := os.ReadFile(filepath.Join(filepath.Dir(path), "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved health.Run
	if err := json.Unmarshal(snapshot, &saved); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report := html.UnescapeString(string(b))
	if saved.Outcome != "completed" || saved.ID != filepath.Base(filepath.Dir(path)) ||
		!strings.Contains(report, "\nRun ID: "+saved.ID+"\n") ||
		!strings.Contains(report, fmt.Sprintf("\nRevision: %d\n", saved.Revision)) {
		t.Fatal("report path delivery changed the published revision pair")
	}
}
