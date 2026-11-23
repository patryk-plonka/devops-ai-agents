package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
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

func TestProcessDraftPathPipe(t *testing.T) {
	const childEnv = "DEVOPS_TEST_DRAFT_PATH_PIPE"
	if mode := os.Getenv(childEnv); mode != "" {
		dir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		start := slices.Index(os.Args, "--")
		if start < 0 {
			t.Fatal("missing draft command")
		}
		var diagnostics io.Writer = os.Stderr
		if mode == "stderr" {
			diagnostics = &draftDiagnosticProbe{writer: diagnostics}
		}
		os.Exit(runApp(cli.App{Directory: dir, Out: os.Stdout, Err: diagnostics}, os.Args[start+1:]))
	}
	if runtime.GOOS == "windows" {
		t.Skip("SIGPIPE is a POSIX signal")
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		code int
	}{
		{"closed pipe", 1},
		{"interrupt with a full pipe", 130},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, path, findingIDs := copyDraftReport(t)
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer read.Close()
			defer write.Close()
			if tc.code == 130 {
				fillDraftPathPipe(t, write)
			} else if err := read.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, "-test.run", "^TestProcessDraftPathPipe$", "--",
				"health", "draft", "--report", path, "--type", "ticket",
				"--finding", findingIDs[0], "--finding", findingIDs[1])
			cmd.Dir = root
			cmd.Env = append(os.Environ(), childEnv+"=1")
			cmd.Stdout = write
			stderr, err := cmd.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if cmd.ProcessState == nil {
					cancel()
					_ = cmd.Wait()
				}
			}()
			reader := bufio.NewReader(stderr)
			firstLine, err := reader.ReadString('\n')
			firstPath := filepath.Join(filepath.Dir(path), "drafts", "ticket-"+findingIDs[0]+".md")
			if err != nil || !strings.Contains(firstLine, "Draft created: "+firstPath) {
				t.Fatalf("first draft was not reported: %q (%v)", firstLine, err)
			}
			if tc.code == 130 {
				// Publication is complete and the pipe stays full until the child exits.
				if err := cmd.Process.Signal(os.Interrupt); err != nil {
					t.Fatal(err)
				}
			}
			remaining, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			var exit *exec.ExitError
			if err := cmd.Wait(); !errors.As(err, &exit) || exit.ExitCode() != tc.code {
				t.Fatalf("expected exit %d, got %v\n%s%s", tc.code, err, firstLine, remaining)
			}
			if !strings.Contains(string(remaining), "draft remains available at "+firstPath) ||
				!strings.Contains(string(remaining), "Pending selections:\n"+findingIDs[1]) {
				t.Fatalf("missing retained path or pending selection:\n%s", remaining)
			}
			body, err := os.ReadFile(firstPath)
			if err != nil || !bytes.Contains(body, []byte(findingIDs[0])) || !bytes.HasSuffix(body, []byte("\n")) {
				t.Fatalf("first draft is incomplete: %v", err)
			}
			secondPath := filepath.Join(filepath.Dir(firstPath), "ticket-"+findingIDs[1]+".md")
			if _, err := os.Stat(secondPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("pending draft was published: %v", err)
			}
		})
	}
	t.Run("interrupt with full stderr", func(t *testing.T) {
		root, path, findingIDs := copyDraftReport(t)
		read, write, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer read.Close()
		defer write.Close()
		fillDraftPathPipe(t, write)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "-test.run", "^TestProcessDraftPathPipe$", "--",
			"health", "draft", "--report", path, "--type", "ticket",
			"--finding", findingIDs[0], "--finding", findingIDs[1])
		cmd.Dir = root
		cmd.Env = append(os.Environ(), childEnv+"=stderr")
		cmd.Stderr = write
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if cmd.ProcessState == nil {
				cancel()
				_ = cmd.Wait()
			}
		}()
		// The probe signals at the first stderr write. The draft is already
		// published, and the stderr pipe stays full until the child exits.
		var ready [1]byte
		if _, err := io.ReadFull(stdout, ready[:]); err != nil || ready[0] != 1 {
			t.Fatalf("draft diagnostic did not start: %v", err)
		}
		dir := filepath.Join(filepath.Dir(path), "drafts")
		first := filepath.Join(dir, "ticket-"+findingIDs[0]+".md")
		original, err := os.ReadFile(first)
		if err != nil || !bytes.Contains(original, []byte(findingIDs[0])) || !bytes.HasSuffix(original, []byte("\n")) {
			t.Fatalf("published draft is incomplete: %v", err)
		}
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			t.Fatal(err)
		}
		var exit *exec.ExitError
		if err := cmd.Wait(); !errors.As(err, &exit) || exit.ExitCode() != 130 {
			t.Fatalf("expected exit 130 with stderr still full, got %v", err)
		}
		if body, err := os.ReadFile(first); err != nil || !bytes.Equal(body, original) {
			t.Fatalf("interruption changed the saved draft: %v", err)
		}
		second := filepath.Join(dir, "ticket-"+findingIDs[1]+".md")
		if _, err := os.Stat(second); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("pending draft was published: %v", err)
		}
	})
}

type draftDiagnosticProbe struct {
	writer io.Writer
	first  sync.Once
}

func (p *draftDiagnosticProbe) Write(data []byte) (int, error) {
	p.first.Do(func() { _, _ = os.Stdout.Write([]byte{1}) })
	return p.writer.Write(data)
}

func copyDraftReport(t *testing.T) (root, path string, findingIDs []string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+runstore.Module+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join("..", "..", "internal", "report", "testdata", "assessment")
	snapshot, err := os.ReadFile(filepath.Join(fixture, "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	var run health.Run
	if err := json.Unmarshal(snapshot, &run); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "runs", run.ID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(filepath.Join(fixture, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(dir, "report.md")
	for name, body := range map[string][]byte{filepath.Join(dir, "run.json"): snapshot, path: report} {
		if err := os.WriteFile(name, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range run.Findings {
		findingIDs = append(findingIDs, f.ID)
	}
	if len(findingIDs) < 2 {
		t.Fatal("expected at least two saved findings")
	}
	return root, path, findingIDs
}

func fillDraftPathPipe(t *testing.T, file *os.File) {
	t.Helper()
	fd := int(file.Fd())
	if err := syscall.SetNonblock(fd, true); err != nil {
		t.Fatal(err)
	}
	padding := make([]byte, 4096)
	for _, size := range []int{len(padding), 1} {
		for {
			if _, err := syscall.Write(fd, padding[:size]); errors.Is(err, syscall.EAGAIN) {
				break
			} else if err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := syscall.SetNonblock(fd, false); err != nil {
		t.Fatal(err)
	}
}
