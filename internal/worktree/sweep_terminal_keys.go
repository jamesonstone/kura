package worktree

import "io"

type sweepKeyKind int

const (
	sweepKeyIgnore sweepKeyKind = iota
	sweepKeyNext
	sweepKeyPrevious
	sweepKeyToggle
	sweepKeyChoose
	sweepKeyCancel
	sweepKeyAll
	sweepKeyClear
	sweepKeySort
	sweepKeyExplain
	sweepKeyFilter
	sweepKeyBackspace
	sweepKeyRune
)

type sweepKey struct {
	kind sweepKeyKind
	char byte
}

func readSweepKey(input io.Reader) (sweepKey, error) {
	var first [1]byte
	if _, err := io.ReadFull(input, first[:]); err != nil {
		return sweepKey{}, err
	}
	switch first[0] {
	case '\r', '\n':
		return sweepKey{kind: sweepKeyChoose}, nil
	case ' ':
		return sweepKey{kind: sweepKeyToggle}, nil
	case '\t', 'j', 'J':
		return sweepKey{kind: sweepKeyNext}, nil
	case 'k', 'K':
		return sweepKey{kind: sweepKeyPrevious}, nil
	case 'q', 'Q', 3:
		return sweepKey{kind: sweepKeyCancel}, nil
	case 'a', 'A':
		return sweepKey{kind: sweepKeyAll}, nil
	case 'u', 'U':
		return sweepKey{kind: sweepKeyClear}, nil
	case 's', 'S':
		return sweepKey{kind: sweepKeySort}, nil
	case 'e', 'E':
		return sweepKey{kind: sweepKeyExplain}, nil
	case '/':
		return sweepKey{kind: sweepKeyFilter}, nil
	case 0x7f, 0x08:
		return sweepKey{kind: sweepKeyBackspace}, nil
	case 0x1b:
		return readSweepEscape(input)
	default:
		if first[0] >= 0x20 && first[0] < 0x7f {
			return sweepKey{kind: sweepKeyRune, char: first[0]}, nil
		}
		return sweepKey{kind: sweepKeyIgnore}, nil
	}
}

func readSweepEscape(input io.Reader) (sweepKey, error) {
	var sequence [2]byte
	if _, err := io.ReadFull(input, sequence[:]); err != nil {
		return sweepKey{kind: sweepKeyCancel}, nil
	}
	if sequence[0] != '[' && sequence[0] != 'O' {
		return sweepKey{kind: sweepKeyCancel}, nil
	}
	switch sequence[1] {
	case 'A', 'D', 'Z':
		return sweepKey{kind: sweepKeyPrevious}, nil
	case 'B', 'C':
		return sweepKey{kind: sweepKeyNext}, nil
	default:
		return sweepKey{kind: sweepKeyIgnore}, nil
	}
}
