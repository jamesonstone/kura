package selector

import (
	"bufio"
	"strings"
	"testing"
)

func TestModelMovesWrapsAndPreservesCatalogOrder(t *testing.T) {
	model := NewModel(testItems())
	model.Move(-1)
	if model.Focused() != 2 {
		t.Fatalf("focused = %d, want 2", model.Focused())
	}
	model.Toggle()
	model.Move(1)
	model.Toggle()
	selected := model.Selected()
	if strings.Join(selected, ",") != "one,three" {
		t.Fatalf("selected = %v, want [one three]", selected)
	}
	model.Toggle()
	if strings.Join(model.Selected(), ",") != "three" {
		t.Fatalf("selected after toggle = %v", model.Selected())
	}
}

func TestReadKeyRecognizesSelectionControls(t *testing.T) {
	tests := []struct {
		input string
		want  key
	}{
		{" ", keyToggle}, {"x", keyToggle}, {"\r", keyConfirm}, {"\n", keyConfirm},
		{"\t", keyDown}, {"j", keyDown}, {"k", keyUp}, {"q", keyCancel},
		{string([]byte{3}), keyCancel}, {"\x1b[A", keyUp}, {"\x1b[B", keyDown},
		{"?", keyUnknown},
	}
	for _, test := range tests {
		got, err := readKey(bufio.NewReader(strings.NewReader(test.input)))
		if err != nil {
			t.Fatalf("readKey(%q): %v", test.input, err)
		}
		if got != test.want {
			t.Fatalf("readKey(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestRenderShowsFocusSelectionAndInstructions(t *testing.T) {
	model := NewModel(testItems()[:2])
	model.Move(1)
	model.Toggle()
	output := Render(model)
	for _, want := range []string{
		"Kura command storehouse",
		"Space to select",
		"  [ ] one",
		"› [x] two",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("render missing %q:\n%s", want, output)
		}
	}
}

func TestEmptyModelConfirmsNoSelection(t *testing.T) {
	model := NewModel(nil)
	model.Move(1)
	model.Toggle()
	if len(model.Selected()) != 0 || !strings.Contains(Render(model), "No tools are available") {
		t.Fatalf("empty model was not stable: %q", Render(model))
	}
}

func testItems() []Item {
	return []Item{
		{ID: "one", Name: "one", Description: "first"},
		{ID: "two", Name: "two", Description: "second"},
		{ID: "three", Name: "three", Description: "third"},
	}
}
