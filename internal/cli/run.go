package cli

import (
	"errors"
	"fmt"
	"io"
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

// Run streams input through the configured CLI transformation.
func Run(options Options, input io.Reader, output io.Writer) error {
	if options.WrapWidth < 0 {
		return fmt.Errorf("wrap width %d: %w", options.WrapWidth, ErrNegativeWrapWidth)
	}

	if options.Mode == ModeDecode {
		return decodeStream(input, output, options.IgnoreGarbage)
	}

	return encodeStream(input, output, options.WrapWidth)
}
