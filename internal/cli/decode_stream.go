package cli

import (
	"fmt"
	"io"

	"github.com/swills/base84"
)

func decodeStream(input io.Reader, output io.Writer, ignoreGarbage bool) error {
	filtered := &filteringReader{input: input, ignoreGarbage: ignoreGarbage}
	decoder := base84.NewDecoder(base84.StdEncoding, filtered)
	buffer := make([]byte, streamBufferSize)

	for {
		count, readErr := decoder.Read(buffer)
		if count > 0 {
			err := writeDecoded(output, buffer[:count])
			if err != nil {
				return err
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}

			sourceErr := unwrapStreamReadError(readErr)
			if sourceErr != nil {
				return fmt.Errorf("read input: %w", sourceErr)
			}

			return fmt.Errorf("decode input: %w", readErr)
		}
	}
}

func writeDecoded(output io.Writer, value []byte) error {
	written, err := output.Write(value)
	if err == nil && written != len(value) {
		err = io.ErrShortWrite
	}

	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}
