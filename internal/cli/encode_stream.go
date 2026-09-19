package cli

import (
	"bufio"
	"fmt"
	"io"

	"github.com/swills/base84"
)

type streamEncoder struct {
	output      *bufio.Writer
	accumulator uint64
	bitCount    uint
	column      int
	wrapWidth   int
	wrote       bool
}

func encodeStream(input io.Reader, output io.Writer, wrapWidth int) error {
	encoder := streamEncoder{
		output:      bufio.NewWriterSize(output, streamBufferSize),
		accumulator: 0,
		bitCount:    0,
		column:      0,
		wrapWidth:   wrapWidth,
		wrote:       false,
	}
	buffer := make([]byte, streamBufferSize)

	for {
		count, readErr := input.Read(buffer)
		if count > 0 {
			err := encoder.write(buffer[:count])
			if err != nil {
				return err
			}

			err = encoder.flush()
			if err != nil {
				return err
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				return fmt.Errorf("read input: %w", readErr)
			}

			break
		}
	}

	err := encoder.finish()
	if err != nil {
		return err
	}

	return encoder.flush()
}

func (encoder *streamEncoder) write(source []byte) error {
	for _, sourceByte := range source {
		encoder.accumulator |= uint64(sourceByte) << encoder.bitCount
		encoder.bitCount += 8

		if encoder.bitCount <= streamGroupBits {
			continue
		}

		low := uint32(encoder.accumulator & streamLowWordMask)
		bits := streamGroupBitCount(uint64(low))

		value := low
		if bits == streamGroupBits {
			value &= uint32(streamGroupMask)
		}

		err := encoder.writeDigits(value, streamGroupCharacters)
		if err != nil {
			return err
		}

		encoder.accumulator >>= bits
		encoder.bitCount -= bits
	}

	return nil
}

func (encoder *streamEncoder) finish() error {
	if encoder.bitCount > 0 {
		characters := streamTailCharacterCount(encoder.bitCount)
		if streamFitsOneCharacter(encoder.bitCount, encoder.accumulator) {
			characters = 1
		}

		value := uint32(encoder.accumulator & streamLowWordMask)

		err := encoder.writeDigits(value, characters)
		if err != nil {
			return err
		}
	}

	if !encoder.wrote {
		return nil
	}

	err := encoder.output.WriteByte('\n')
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func (encoder *streamEncoder) writeDigits(value uint32, count int) error {
	for index := range [streamGroupCharacters]struct{}{} {
		if index == count {
			break
		}

		if encoder.wrapWidth > 0 && encoder.column == encoder.wrapWidth {
			err := encoder.output.WriteByte('\n')
			if err != nil {
				return fmt.Errorf("write output: %w", err)
			}

			encoder.column = 0
		}

		err := encoder.output.WriteByte(base84.Alphabet[value%uint32(streamAlphabetSize)])
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		value /= uint32(streamAlphabetSize)
		encoder.column++
		encoder.wrote = true
	}

	return nil
}

func (encoder *streamEncoder) flush() error {
	err := encoder.output.Flush()
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}
