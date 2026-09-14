package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/swills/base84"
)

// Mode selects whether Run encodes or decodes its input.
type Mode bool

const (
	ModeEncode Mode = false
	ModeDecode Mode = true
)

// ErrNegativeWrapWidth indicates an invalid output wrap width.
var ErrNegativeWrapWidth = errors.New("wrap width must be nonnegative")

// Options configures a CLI transformation.
type Options struct {
	Mode          Mode
	IgnoreGarbage bool
	WrapWidth     int
}

// Run reads all input, applies the configured CLI transformation, and writes the result.
func Run(options Options, input io.Reader, output io.Writer) error {
	if options.WrapWidth < 0 {
		return fmt.Errorf("wrap width %d: %w", options.WrapWidth, ErrNegativeWrapWidth)
	}

	inputBytes, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var outputBytes []byte

	if options.Mode == ModeDecode {
		if options.IgnoreGarbage {
			inputBytes = removeNonAlphabet(inputBytes)
		} else {
			inputBytes = removeASCIIWhitespace(inputBytes)
		}

		outputBytes, err = base84.Decode(string(inputBytes))
		if err != nil {
			return fmt.Errorf("decode input: %w", err)
		}
	} else {
		outputBytes = formatEncoded([]byte(base84.Encode(inputBytes)), options.WrapWidth)
	}

	if len(outputBytes) == 0 {
		return nil
	}

	written, writeErr := output.Write(outputBytes)
	if writeErr != nil {
		return fmt.Errorf("write output: %w", writeErr)
	}

	if written < len(outputBytes) {
		return fmt.Errorf("write output: %w", io.ErrShortWrite)
	}

	return nil
}

func removeNonAlphabet(input []byte) []byte {
	filtered := input[:0]
	for _, character := range input {
		if strings.IndexByte(base84.Alphabet, character) >= 0 {
			filtered = append(filtered, character)
		}
	}

	return filtered
}

func removeASCIIWhitespace(input []byte) []byte {
	filtered := input[:0]
	for _, character := range input {
		switch character {
		case ' ', '\t', '\r', '\n', '\v', '\f':
			continue
		default:
			filtered = append(filtered, character)
		}
	}

	return filtered
}

func formatEncoded(encoded []byte, wrapWidth int) []byte {
	if len(encoded) == 0 {
		return nil
	}

	if wrapWidth == 0 || wrapWidth >= len(encoded) {
		return append(encoded, '\n')
	}

	lineBreaks := (len(encoded) - 1) / wrapWidth

	formatted := make([]byte, 0, len(encoded)+lineBreaks+1)
	for len(encoded) > wrapWidth {
		formatted = append(formatted, encoded[:wrapWidth]...)
		formatted = append(formatted, '\n')
		encoded = encoded[wrapWidth:]
	}

	formatted = append(formatted, encoded...)

	return append(formatted, '\n')
}
