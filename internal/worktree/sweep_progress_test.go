package worktree

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSweepProgressAnimatesSanitizesAndClears(t *testing.T) {
	var output bytes.Buffer
	progress := newSweepProgress(&output, true)
	progress.start("Discovering \x1b[31mroots")
	progress.update("Querying repository 1/2")
	time.Sleep(sweepProgressInterval * 2)
	progress.stopLine()
	result := output.String()
	if !strings.Contains(result, "Querying repository 1/2") || !strings.HasSuffix(result, "\r\x1b[2K") {
		t.Fatalf("progress = %q", result)
	}
	if strings.Contains(result, "\x1b[31m") {
		t.Fatal("progress retained untrusted terminal control characters")
	}
}

func TestDisabledSweepProgressWritesNothing(t *testing.T) {
	var output bytes.Buffer
	progress := newSweepProgress(&output, false)
	progress.start("hidden")
	progress.update("still hidden")
	progress.stopLine()
	if output.Len() != 0 {
		t.Fatalf("disabled progress = %q", output.String())
	}
}
