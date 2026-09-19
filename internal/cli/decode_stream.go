package cli

import (
	"bufio"
	"fmt"
	"io"

	"github.com/swills/base84"
)

type streamDecoder struct {
	output        *bufio.Writer
	accumulator   uint64
	pendingBits   uint
	finalBits     uint
	encodedLength int
	digitCount    int
	digits        [streamGroupCharacters]byte
	ignoreGarbage bool
}

func decodeStream(input io.Reader, output io.Writer, ignoreGarbage bool) error {
	decoder := streamDecoder{
		output:        bufio.NewWriterSize(output, streamBufferSize),
		accumulator:   0,
		pendingBits:   0,
		finalBits:     0,
		digits:        [streamGroupCharacters]byte{},
		digitCount:    0,
		encodedLength: 0,
		ignoreGarbage: ignoreGarbage,
	}
	buffer := make([]byte, streamBufferSize)

	for {
		count, readErr := input.Read(buffer)
		if count > 0 {
			err := decoder.write(buffer[:count])
			if err != nil {
				return decoder.flushBefore(err)
			}

			err = decoder.flush()
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

	err := decoder.finish()
	if err != nil {
		return decoder.flushBefore(err)
	}

	return decoder.flush()
}

func (decoder *streamDecoder) write(source []byte) error {
	for _, character := range source {
		digit := standardDigits[character]
		if digit == streamInvalidDigit {
			if decoder.ignoreGarbage || isASCIIWhitespace(character) {
				continue
			}

			return fmt.Errorf(
				"decode input: character %q at byte %d: %w",
				character,
				decoder.encodedLength,
				base84.ErrInvalidCharacter,
			)
		}

		decoder.digits[decoder.digitCount] = digit
		decoder.digitCount++
		decoder.encodedLength++

		if decoder.digitCount == streamGroupCharacters {
			err := decoder.consume(streamGroupBitCount(decoder.value()))
			if err != nil {
				return err
			}

			decoder.digitCount = 0
		}
	}

	return nil
}

func (decoder *streamDecoder) finish() error {
	if decoder.digitCount > 0 {
		bits, err := streamTailBitCount(decoder.digitCount, decoder.pendingBits, decoder.value())
		if err != nil {
			return fmt.Errorf(
				"decode input: tail at byte %d: %w",
				decoder.encodedLength-decoder.digitCount,
				err,
			)
		}

		return decoder.consume(bits)
	}

	if decoder.encodedLength > 0 &&
		(decoder.accumulator != 0 || decoder.finalBits-decoder.pendingBits <= 25) {
		return fmt.Errorf("decode input: final group: %w", base84.ErrInvalidPadding)
	}

	return nil
}

func (decoder *streamDecoder) consume(bits uint) error {
	decoder.finalBits = bits
	decoder.accumulator |= decoder.value() << decoder.pendingBits
	decoder.pendingBits += bits

	for decoder.pendingBits >= 8 {
		value := byte(decoder.accumulator & 0xff)

		err := decoder.output.WriteByte(value)
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		decoder.accumulator >>= 8
		decoder.pendingBits -= 8
	}

	return nil
}

func (decoder *streamDecoder) value() uint64 {
	var (
		value uint64
		place = uint64(1)
	)

	for _, digit := range decoder.digits[:decoder.digitCount] {
		value += uint64(digit) * place
		place *= uint64(streamAlphabetSize)
	}

	return value
}

func (decoder *streamDecoder) flush() error {
	err := decoder.output.Flush()
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func (decoder *streamDecoder) flushBefore(transformErr error) error {
	err := decoder.flush()
	if err != nil {
		return err
	}

	return transformErr
}

func isASCIIWhitespace(character byte) bool {
	switch character {
	case ' ', '\t', '\r', '\n', '\v', '\f':
		return true
	default:
		return false
	}
}
