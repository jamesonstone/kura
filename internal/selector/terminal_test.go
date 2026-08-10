package selector

import (
	"bytes"
	"context"
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

func TestTerminalMultiSelectAndRestoration(t *testing.T) {
	script, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script command is required for pseudo-terminal coverage")
	}
	args := []string{"-q"}
	switch runtime.GOOS {
	case "darwin":
		args = append(args, "/dev/null", os.Args[0], "-test.run=^TestTerminalPTYHelper$")
	case "linux":
		command := shellQuote(os.Args[0]) + " -test.run=^TestTerminalPTYHelper$"
		args = append(args, "-c", command, "/dev/null")
	default:
		t.Skipf("script pseudo-terminal invocation is not configured for %s", runtime.GOOS)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, script, args...)
	command.Env = append(os.Environ(), "KURA_TEST_SELECTOR_PTY=1")
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := stdin.Write([]byte(" \r")); err != nil {
		t.Fatal(err)
	}
	_ = stdin.Close()
	if err := command.Wait(); err != nil {
		t.Fatalf("PTY helper failed: %v\n%s", err, output.String())
	}
	if ctx.Err() != nil {
		t.Fatalf("PTY helper timed out: %v\n%s", ctx.Err(), output.String())
	}
	for _, want := range []string{"\x1b[?1049h", "\x1b[?1049l", "selector PTY passed"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("PTY output missing %q: %q", want, output.Bytes())
		}
	}
}

func TestTerminalPTYHelper(t *testing.T) {
	if os.Getenv("KURA_TEST_SELECTOR_PTY") != "1" {
		return
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		t.Fatal("PTY helper streams are not terminals")
	}
	before, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := (Terminal{Input: os.Stdin, Output: os.Stdout}).Select(context.Background(), testItems()[:2])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(selected, ",") != "one" {
		t.Fatalf("selected = %v, want [one]", selected)
	}
	after, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normalizedState(before), normalizedState(after)) {
		t.Fatalf("terminal state was not restored:\nbefore: %#v\nafter: %#v", before, after)
	}
	fmt.Fprintln(os.Stderr, "selector PTY passed")
}

func normalizedState(state *term.State) []uint64 {
	var values []uint64
	var collect func(reflect.Value, string)
	collect = func(value reflect.Value, name string) {
		switch value.Kind() {
		case reflect.Pointer:
			collect(value.Elem(), name)
		case reflect.Struct:
			for index := 0; index < value.NumField(); index++ {
				collect(value.Field(index), value.Type().Field(index).Name)
			}
		case reflect.Array:
			for index := 0; index < value.Len(); index++ {
				collect(value.Index(index), name)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			fieldValue := value.Uint()
			if runtime.GOOS == "darwin" && name == "Lflag" {
				fieldValue &^= 0x20000000
			}
			values = append(values, fieldValue)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			values = append(values, uint64(value.Int()))
		}
	}
	collect(reflect.ValueOf(state), "")
	return values
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
