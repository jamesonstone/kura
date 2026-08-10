package selector

import (
	"bufio"
	"fmt"
	"io"
)

type key int

const (
	keyUnknown key = iota
	keyUp
	keyDown
	keyToggle
	keyConfirm
	keyCancel
)

func readKey(reader *bufio.Reader) (key, error) {
	value, err := reader.ReadByte()
	if err != nil {
		return keyUnknown, err
	}
	switch value {
	case 3, 'q':
		return keyCancel, nil
	case ' ', 'x':
		return keyToggle, nil
	case '\r', '\n':
		return keyConfirm, nil
	case '\t', 'j':
		return keyDown, nil
	case 'k':
		return keyUp, nil
	case 27:
		return readEscapeKey(reader)
	default:
		return keyUnknown, nil
	}
}

func readEscapeKey(reader *bufio.Reader) (key, error) {
	first, err := reader.ReadByte()
	if err == io.EOF {
		return keyCancel, nil
	}
	if err != nil {
		return keyUnknown, err
	}
	if first != '[' {
		return keyCancel, nil
	}
	second, err := reader.ReadByte()
	if err != nil {
		return keyUnknown, fmt.Errorf("read escape sequence: %w", err)
	}
	switch second {
	case 'A':
		return keyUp, nil
	case 'B':
		return keyDown, nil
	default:
		return keyUnknown, nil
	}
}
