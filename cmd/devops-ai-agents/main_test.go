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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
	"github.com/patryk-plonka/devops-ai-agents/internal/cli"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

const signalSub = "11111111-1111-1111-1111-111111111111"
const signalVM = "/subscriptions/" + signalSub + "/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/vm"

type signalCredential struct{}

func (signalCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "synthetic-token-canary", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type signalTransport struct {
	calls int
	mode  string
}

func (s *signalTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls++
	if s.calls > 2 {
		return nil, fmt.Errorf("unexpected third request")
	}
	if s.calls == 1 {
		body := fmt.Sprintf(`{"value":[{"id":%q,"name":"vm","tags":{"owner":"consultant","password":"secret-canary"}}],"nextLink":%q}`, signalVM, "https://management.azure.com/subscriptions/"+signalSub+"/resources?api-version=2025-04-01&$skiptoken=next")
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	// The next native SDK request proves the preceding page was checkpointed.
	if s.mode == "retry" {
		return &http.Response{StatusCode: 429, Header: http.Header{"Content-Type": {"application/json"}, "Retry-After": {"30"}}, Body: retryBody{io.NopCloser(strings.NewReader(`{"error":{"message":"body-canary"}}`))}}, nil
	}
	fmt.Fprintln(os.Stderr, "READY_AFTER_CHECKPOINT")
	<-req.Context().Done()
	return nil, req.Context().Err()
}

type retryBody struct{ io.ReadCloser }

func (b retryBody) Close() error {
	// The SDK drains the response after deciding to retry, before its delay.
	fmt.Fprintln(os.Stderr, "READY_AFTER_CHECKPOINT")
	return b.ReadCloser.Close()
}

type blockedFinalStore struct {
	cli.Storage
	fail bool
}

func (s blockedFinalStore) Save(r *health.Run) error {
	if r.EndedAt != nil {
		if s.fail {
			return fmt.Errorf("storage-canary")
		}
		fmt.Fprintln(os.Stderr, "FINALIZATION_BLOCKED")
		select {}
	}
	return s.Storage.Save(r)
}
func signalChild(t *testing.T) {
	mode := os.Getenv("DEVOPS_SIGNAL_MODE")
	root := os.Getenv("DEVOPS_SIGNAL_ROOT")
	account := azure.Account{ID: signalSub, Name: "Synthetic", TenantID: "22222222-2222-2222-2222-222222222222", Environment: "AzureCloud"}
	account.User.Type = "user"
	transport := &signalTransport{mode: mode}
	collection := azure.Collection{Base: transport, Credential: func(string) (azcore.TokenCredential, error) { return signalCredential{}, nil }}
	app := cli.App{Directory: root, In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Discover: func(context.Context) ([]azure.Account, error) { return []azure.Account{account}, nil }, Validate: func(context.Context, azure.Account) error { return nil }, Inventory: collection.Inventory, ResourceHealth: func(context.Context, string, func(health.Page) error) error {
		t.Fatal("second collector ran after interrupt")
		return nil
	}}
	if mode == "second interrupt" || mode == "failed final write" {
		app.NewStore = func(root string, r *health.Run) (cli.Storage, error) {
			s, err := runstore.New(root, r)
			return blockedFinalStore{Storage: s, fail: mode == "failed final write"}, err
		}
	}
	code := runApp(app, []string{"health", "check", "--subscription", signalSub})
	fmt.Fprintf(os.Stderr, "REQUESTS=%d\n", transport.calls)
	os.Exit(code)
}
func TestProcessInterruptPreservesCheckpoint(t *testing.T) {
	if os.Getenv("DEVOPS_SIGNAL_CHILD") == "1" {
		signalChild(t)
		return
	}
	if runtime.GOOS == "windows" {
		t.Skip("os.Interrupt delivery unavailable on Windows")
	}
	for _, mode := range []string{"next page", "retry", "failed final write", "second interrupt"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+runstore.Module+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, "-test.run=^TestProcessInterruptPreservesCheckpoint$")
			cmd.Env = append(os.Environ(), "DEVOPS_SIGNAL_CHILD=1", "DEVOPS_SIGNAL_ROOT="+root, "DEVOPS_SIGNAL_MODE="+mode)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			stderr, err := cmd.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			ready := make(chan string, 2)
			scanned := make(chan string, 1)
			go func() {
				var text strings.Builder
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					line := scanner.Text()
					text.WriteString(line + "\n")
					if line == "READY_AFTER_CHECKPOINT" || line == "FINALIZATION_BLOCKED" {
						ready <- line
					}
				}
				scanned <- text.String()
			}()
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			await := func(want string) {
				t.Helper()
				select {
				case line := <-ready:
					if line != want {
						t.Fatalf("signal synchronization %q want %q", line, want)
					}
				case <-ctx.Done():
					t.Fatal("child did not reach synchronized stage")
				}
			}
			await("READY_AFTER_CHECKPOINT")
			paths, err := filepath.Glob(filepath.Join(root, "runs", "*", "run.json"))
			if err != nil || len(paths) != 1 {
				t.Fatal("checkpoint missing before signal")
			}
			read := func() health.Run {
				t.Helper()
				b, err := os.ReadFile(paths[0])
				if err != nil {
					t.Fatal(err)
				}
				var r health.Run
				if json.Unmarshal(b, &r) != nil {
					t.Fatal("bad checkpoint")
				}
				return r
			}
			checkpoint := read()
			if len(checkpoint.Evidence) != 1 {
				t.Fatal("page not committed before next request")
			}
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				t.Fatal(err)
			}
			if mode == "second interrupt" {
				await("FINALIZATION_BLOCKED")
				if err := cmd.Process.Signal(os.Interrupt); err != nil {
					t.Fatal(err)
				}
			}
			// Drain stderr to EOF before Wait closes the pipe.
			var diagnostics string
			select {
			case diagnostics = <-scanned:
			case <-ctx.Done():
				t.Fatal("interruption did not stop child")
			}
			err = cmd.Wait()
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("expected nonzero process exit: %v", err)
			}
			saved := read()
			if len(saved.Evidence) != 1 || saved.ID != checkpoint.ID {
				t.Fatal("checkpoint evidence lost")
			}
			if strings.Contains(stdout.String()+diagnostics, "canary") {
				t.Fatal("sensitive output leaked")
			}
			if mode == "second interrupt" {
				if saved.Revision != checkpoint.Revision+1 || saved.EndedAt != nil {
					t.Fatal("forced exit damaged last checkpoint")
				}
				if stdout.Len() != 0 {
					t.Fatal("forced exit claimed report")
				}
				return
			}
			if exit.ExitCode() != 130 || !strings.Contains(diagnostics, "REQUESTS=2") {
				t.Fatalf("exit %d diagnostics %s", exit.ExitCode(), diagnostics)
			}
			if mode == "failed final write" {
				if stdout.Len() != 0 || saved.EndedAt != nil || !strings.Contains(diagnostics, "Final run write failed") || !strings.Contains(diagnostics, "Last saved snapshot:") {
					t.Fatalf("final write failure concealed: %s", diagnostics)
				}
				return
			}
			reportPath := strings.TrimSpace(stdout.String())
			b, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatal(err)
			}
			text := html.UnescapeString(string(b))
			if saved.Outcome != "cancelled" || saved.EndedAt == nil || saved.Sources["resource_health"].Reason != "not_attempted" || !strings.Contains(text, fmt.Sprintf("Revision: %d", saved.Revision)) || !strings.Contains(text, "Execution outcome: **cancelled**") || !strings.Contains(text, saved.Evidence[0].ID) {
				t.Fatalf("cancelled report disagrees with saved checkpoint: run=%+v diagnostics=%s report=%s", saved, diagnostics, text)
			}
		})
	}
}
