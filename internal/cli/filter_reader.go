package cli

import (
	"errors"
	"io"
	"strings"

	"github.com/swills/base84"
)

const streamBufferSize = 32 * 1024

type filteringReader struct {
	input         io.Reader
	ignoreGarbage bool
}

type streamReadError struct {
	err error
}

func (streamErr *streamReadError) Error() string {
	return streamErr.err.Error()
}

func (streamErr *streamReadError) Unwrap() error {
	return streamErr.err
}

func (reader *filteringReader) Read(destination []byte) (int, error) {
	for {
		count, err := reader.input.Read(destination)
		kept := 0

		for _, character := range destination[:count] {
			if reader.keep(character) {
				destination[kept] = character
				kept++
			}
		}

		if kept > 0 {
			if err != nil && err != io.EOF {
				err = &streamReadError{err: err}
			}

			return kept, err
		}

		if err != nil {
			if err == io.EOF {
				return 0, io.EOF
			}

			return 0, &streamReadError{err: err}
		}
	}
}

func (reader *filteringReader) keep(character byte) bool {
	if strings.IndexByte(base84.Alphabet, character) >= 0 {
		return true
	}

	return !reader.ignoreGarbage && !isASCIIWhitespace(character)
}

func isASCIIWhitespace(character byte) bool {
	switch character {
	case ' ', '\t', '\r', '\n', '\v', '\f':
		return true
	default:
		return false
	}
}

func unwrapStreamReadError(err error) error {
	readErr, ok := errors.AsType[*streamReadError](err)
	if ok {
		return readErr.err
	}

	return nil
}
