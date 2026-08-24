package worktree

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"
)

func TestSelectSweepTerminalCancellationRestoresPTY(t *testing.T) {
	script, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script command is required for pseudo-terminal coverage")
	}
	args := []string{"-q"}
	switch runtime.GOOS {
	case "darwin":
		args = append(args, "/dev/null", os.Args[0], "-test.run=^TestSelectSweepTerminalCancellationPTYHelper$")
	case "linux":
		args = append(args, "-c", shellQuote(os.Args[0])+" -test.run=^TestSelectSweepTerminalCancellationPTYHelper$", "/dev/null")
	default:
		t.Skipf("script pseudo-terminal invocation is not configured for %s", runtime.GOOS)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, script, args...)
	command.Env = append(os.Environ(), "KURA_TEST_SWEEP_PTY=1")
	readyRead, readyWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = readyRead.Close() }()
	command.ExtraFiles = []*os.File{readyWrite}
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	_ = readyWrite.Close()
	ready := make(chan error, 1)
	go func() {
		_, readErr := bufio.NewReader(readyRead).ReadString('\n')
		ready <- readErr
	}()
	select {
	case readyErr := <-ready:
		if readyErr != nil {
			t.Fatalf("wait for sweep cancellation: %v\n%s", readyErr, output.String())
		}
	case <-ctx.Done():
		t.Fatalf("sweep did not signal cancellation: %v\n%s", ctx.Err(), output.String())
	}
	_, writeErr := fmt.Fprintln(stdin, "sentinel-after-sweep-cancel")
	err = command.Wait()
	_ = stdin.Close()
	if writeErr != nil || err != nil || ctx.Err() != nil {
		t.Fatalf("sweep PTY failed: write=%v wait=%v context=%v\n%s", writeErr, err, ctx.Err(), output.String())
	}
	if !bytes.Contains(output.Bytes(), []byte(hideCursor)) || !bytes.Contains(output.Bytes(), []byte(showCursor)) {
		t.Fatalf("sweep selector did not restore cursor: %q", output.Bytes())
	}
	if !strings.Contains(output.String(), "sweep PTY cancellation passed") {
		t.Fatalf("sweep helper did not confirm restoration: %q", output.Bytes())
	}
}

func TestSelectSweepTerminalCancellationPTYHelper(t *testing.T) {
	if os.Getenv("KURA_TEST_SWEEP_PTY") != "1" {
		return
	}
	inputFD := int(os.Stdin.Fd())
	if !term.IsTerminal(inputFD) || !term.IsTerminal(int(os.Stdout.Fd())) {
		t.Fatal("PTY helper streams are not terminals")
	}
	before, err := term.GetState(inputFD)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(os.Stdout, os.Stderr)
	app.stdin = os.Stdin
	app.isTerminal = func() bool { return true }
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	selected, err := app.selectSweepTerminal(ctx, []SweepCandidate{{
		ID: "one", Repository: "example/project", Branch: "GH-1", Path: "/tmp/GH-1",
		State: SweepRemoveReady, Selectable: true,
	}}, SweepOptions{Sort: "state"})
	if !errors.Is(err, context.Canceled) || len(selected) != 0 {
		t.Fatalf("selection=%#v error=%v", selected, err)
	}
	ready := os.NewFile(3, "sweep-ready")
	if ready == nil {
		t.Fatal("sweep readiness pipe is unavailable")
	}
	if _, err := fmt.Fprintln(ready, "ready"); err != nil {
		t.Fatal(err)
	}
	if err := ready.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := term.GetState(inputFD)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normalizedTerminalState(before), normalizedTerminalState(after)) {
		t.Fatalf("terminal state changed: before=%#v after=%#v", before, after)
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "sentinel-after-sweep-cancel" {
		t.Fatalf("post-cancel input=%q error=%v", line, err)
	}
	_, _ = fmt.Fprintln(os.Stderr, "sweep PTY cancellation passed")
}
