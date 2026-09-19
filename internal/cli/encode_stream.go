package cli

import (
	"fmt"
	"io"

	"github.com/swills/base84"
)

type formattingWriter struct {
	output    io.Writer
	column    int
	wrapWidth int
	wrote     bool
}

func encodeStream(input io.Reader, output io.Writer, wrapWidth int) error {
	formatted := &formattingWriter{output: output, column: 0, wrapWidth: wrapWidth, wrote: false}
	encoder := base84.NewEncoder(base84.StdEncoding, formatted)
	buffer := make([]byte, streamBufferSize)

	for {
		count, readErr := input.Read(buffer)
		if count > 0 {
			_, err := encoder.Write(buffer[:count])
			if err != nil {
				return fmt.Errorf("encode input: %w", err)
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				return fmt.Errorf("read input: %w", readErr)
			}

			break
		}
	}

	err := encoder.Close()
	if err != nil {
		return fmt.Errorf("finish encoding: %w", err)
	}

	return formatted.finish()
}

func (writer *formattingWriter) Write(source []byte) (int, error) {
	written := 0

	for len(source) > 0 {
		if writer.wrapWidth > 0 && writer.column == writer.wrapWidth {
			err := writer.writeAll([]byte{'\n'})
			if err != nil {
				return written, err
			}

			writer.column = 0
		}

		count := len(source)
		if writer.wrapWidth > 0 {
			count = min(count, writer.wrapWidth-writer.column)
		}

		err := writer.writeAll(source[:count])
		if err != nil {
			return written, err
		}

		writer.column += count
		writer.wrote = true
		written += count
		source = source[count:]
	}

	return written, nil
}

func (writer *formattingWriter) finish() error {
	if !writer.wrote {
		return nil
	}

	return writer.writeAll([]byte{'\n'})
}

func (writer *formattingWriter) writeAll(value []byte) error {
	written, err := writer.output.Write(value)
	if err == nil && written != len(value) {
		err = io.ErrShortWrite
	}

	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}
