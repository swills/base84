package base84

import (
	"errors"
	"fmt"
	"io"
)

type decoder struct {
	encoding       *Encoding
	input          io.Reader
	pendingReadErr error
	terminalErr    error
	returnedErr    error
	inputBuffer    [streamBufferSize]byte
	outputBuffer   [4]byte
	digits         [groupCharacters]byte
	accumulator    uint64
	pendingBits    uint
	finalBits      uint
	inputStart     int
	inputEnd       int
	outputStart    int
	outputEnd      int
	digitCount     int
	encodedLength  int
	noProgress     int
}

// NewDecoder returns a new Base84 stream decoder that reads from input.
// Decoding is strict: bytes outside encoding's alphabet and non-canonical
// padding are reported as errors.
func NewDecoder(encoding *Encoding, input io.Reader) io.Reader {
	return &decoder{
		encoding:       encoding,
		input:          input,
		accumulator:    0,
		inputBuffer:    [streamBufferSize]byte{},
		outputBuffer:   [4]byte{},
		digits:         [groupCharacters]byte{},
		pendingBits:    0,
		finalBits:      0,
		inputStart:     0,
		inputEnd:       0,
		outputStart:    0,
		outputEnd:      0,
		digitCount:     0,
		encodedLength:  0,
		noProgress:     0,
		pendingReadErr: nil,
		terminalErr:    nil,
		returnedErr:    nil,
	}
}

func (stream *decoder) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}

	if stream.returnedErr != nil {
		return 0, stream.returnedErr
	}

	written := 0
	for written < len(destination) {
		if stream.outputStart < stream.outputEnd {
			written += stream.copyOutput(destination[written:])

			continue
		}

		if stream.terminalErr != nil {
			break
		}

		if written > 0 && stream.inputStart == stream.inputEnd && stream.pendingReadErr == nil {
			return written, nil
		}

		stream.advance()
	}

	if written > 0 {
		return written, nil
	}

	stream.returnedErr = stream.terminalErr

	return 0, stream.returnedErr
}

func (stream *decoder) advance() {
	if stream.inputStart < stream.inputEnd {
		stream.consumeCharacter(stream.inputBuffer[stream.inputStart])
		stream.inputStart++

		return
	}

	if stream.pendingReadErr != nil {
		stream.finishInput()

		return
	}

	count, err := stream.input.Read(stream.inputBuffer[:])
	stream.inputStart = 0
	stream.inputEnd = count
	stream.pendingReadErr = err

	if count > 0 {
		stream.noProgress = 0

		return
	}

	if err != nil {
		stream.finishInput()

		return
	}

	stream.noProgress++
	if stream.noProgress >= 100 {
		stream.terminalErr = io.ErrNoProgress
	}
}

func (stream *decoder) consumeCharacter(character byte) {
	stream.digits[stream.digitCount] = character
	stream.digitCount++
	stream.encodedLength++

	if stream.digitCount < groupCharacters {
		return
	}

	value, err := stream.encoding.readChunk(stream.digits[:], stream.encodedLength-groupCharacters)
	if err != nil {
		stream.terminalErr = err

		return
	}

	stream.consumeValue(value, groupBitCount(value))
	stream.digitCount = 0
}

func (stream *decoder) consumeValue(value uint64, bits uint) {
	stream.finalBits = bits
	stream.accumulator |= value << stream.pendingBits
	stream.pendingBits += bits

	for stream.pendingBits >= 8 {
		stream.outputBuffer[stream.outputEnd] = byte(stream.accumulator & 0xff)
		stream.outputEnd++
		stream.accumulator >>= 8
		stream.pendingBits -= 8
	}
}

func (stream *decoder) finishInput() {
	if !errors.Is(stream.pendingReadErr, io.EOF) {
		stream.terminalErr = stream.pendingReadErr

		return
	}

	if stream.digitCount > 0 {
		value, err := stream.encoding.readChunk(
			stream.digits[:stream.digitCount],
			stream.encodedLength-stream.digitCount,
		)
		if err != nil {
			stream.terminalErr = err

			return
		}

		bits, err := tailBitCount(stream.digitCount, stream.pendingBits, value)
		if err != nil {
			stream.terminalErr = fmt.Errorf(
				"tail at byte %d: %w",
				stream.encodedLength-stream.digitCount,
				err,
			)

			return
		}

		stream.consumeValue(value, bits)
	} else if stream.encodedLength > 0 &&
		(stream.accumulator != 0 || stream.finalBits-stream.pendingBits <= 25) {
		stream.terminalErr = fmt.Errorf("final group: %w", ErrInvalidPadding)

		return
	}

	stream.terminalErr = io.EOF
}

func (stream *decoder) copyOutput(destination []byte) int {
	written := copy(destination, stream.outputBuffer[stream.outputStart:stream.outputEnd])
	stream.outputStart += written

	if stream.outputStart == stream.outputEnd {
		stream.outputStart = 0
		stream.outputEnd = 0
	}

	return written
}
