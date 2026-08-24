package worktree

import (
	"fmt"
	"io"
	"strings"
)

func writeSweepConfigDiff(writer io.Writer, before, after []byte) error {
	if _, err := fmt.Fprintln(writer, "\nConfiguration changes:"); err != nil {
		return err
	}
	for _, line := range sweepLineDiff(string(before), string(after)) {
		if _, err := fmt.Fprintln(writer, sanitizeTerminalField(line)); err != nil {
			return err
		}
	}
	return nil
}

func sweepLineDiff(before, after string) []string {
	left, right := splitSweepLines(before), splitSweepLines(after)
	lengths := make([][]int, len(left)+1)
	for index := range lengths {
		lengths[index] = make([]int, len(right)+1)
	}
	for leftIndex := len(left) - 1; leftIndex >= 0; leftIndex-- {
		for rightIndex := len(right) - 1; rightIndex >= 0; rightIndex-- {
			if left[leftIndex] == right[rightIndex] {
				lengths[leftIndex][rightIndex] = lengths[leftIndex+1][rightIndex+1] + 1
			} else {
				lengths[leftIndex][rightIndex] = max(lengths[leftIndex+1][rightIndex], lengths[leftIndex][rightIndex+1])
			}
		}
	}
	result := make([]string, 0, len(left)+len(right))
	leftIndex, rightIndex := 0, 0
	for leftIndex < len(left) && rightIndex < len(right) {
		switch {
		case left[leftIndex] == right[rightIndex]:
			result = append(result, "  "+left[leftIndex])
			leftIndex++
			rightIndex++
		case lengths[leftIndex+1][rightIndex] >= lengths[leftIndex][rightIndex+1]:
			result = append(result, "- "+left[leftIndex])
			leftIndex++
		default:
			result = append(result, "+ "+right[rightIndex])
			rightIndex++
		}
	}
	for ; leftIndex < len(left); leftIndex++ {
		result = append(result, "- "+left[leftIndex])
	}
	for ; rightIndex < len(right); rightIndex++ {
		result = append(result, "+ "+right[rightIndex])
	}
	return result
}

func splitSweepLines(value string) []string {
	value = strings.TrimSuffix(value, "\n")
	if value == "" {
		return []string{}
	}
	return strings.Split(value, "\n")
}
