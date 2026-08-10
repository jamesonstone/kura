package selector

import (
	"fmt"
	"strings"
)

func Render(model *Model) string {
	var output strings.Builder
	output.WriteString("Kura command storehouse\n\n")
	output.WriteString("Use ↑/↓ (or j/k) to move, Space to select, Enter to install, q to cancel.\n\n")
	for index, item := range model.items {
		cursor := "  "
		if index == model.focused {
			cursor = "› "
		}
		mark := " "
		if model.selected[item.ID] {
			mark = "x"
		}
		fmt.Fprintf(&output, "%s[%s] %-14s %s\n", cursor, mark, item.Name, item.Description)
	}
	if len(model.items) == 0 {
		output.WriteString("  No tools are available.\n")
	}
	return output.String()
}
