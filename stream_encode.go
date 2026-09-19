package base84

import "io"

const streamBufferSize = 32 * 1024

type encoder struct {
	encoding    *Encoding
	output      io.Writer
	err         error
	buffer      [streamBufferSize]byte
	accumulator uint64
	bitCount    uint
	buffered    int
	closed      bool
}

// NewEncoder returns a new Base84 stream encoder. Writes are encoded with
// encoding and sent to output. The caller must close the encoder to flush any
// partially encoded tail. Closing the encoder does not close output.
func NewEncoder(encoding *Encoding, output io.Writer) io.WriteCloser {
	return &encoder{
		encoding:    encoding,
		output:      output,
		accumulator: 0,
		buffer:      [streamBufferSize]byte{},
		bitCount:    0,
		buffered:    0,
		err:         nil,
		closed:      false,
	}
}

func (stream *encoder) Write(source []byte) (int, error) {
	if stream.closed {
		return 0, io.ErrClosedPipe
	}

	if stream.err != nil {
		return 0, stream.err
	}

	for index, sourceByte := range source {
		if len(stream.buffer)-stream.buffered < groupCharacters {
			stream.flush()

			if stream.err != nil {
				return index, stream.err
			}
		}

		stream.accumulator |= uint64(sourceByte) << stream.bitCount
		stream.bitCount += 8

		if stream.bitCount <= groupBits {
			continue
		}

		low := uint32(stream.accumulator & lowWordMask)
		bits := groupBitCount(uint64(low))
		value := low

		if bits == groupBits {
			value &= uint32(groupMask)
		}

		stream.writeDigits(value, groupCharacters)
		stream.accumulator >>= bits
		stream.bitCount -= bits
	}

	return len(source), nil
}

func (stream *encoder) Close() error {
	if stream.closed {
		return stream.err
	}

	stream.closed = true
	if stream.err != nil {
		return stream.err
	}

	if stream.bitCount > 0 {
		characters := tailCharacterCount(stream.bitCount)
		if fitsOneCharacter(stream.bitCount, stream.accumulator) {
			characters = 1
		}

		if len(stream.buffer)-stream.buffered < characters {
			stream.flush()

			if stream.err != nil {
				return stream.err
			}
		}

		stream.writeDigits(uint32(stream.accumulator&lowWordMask), characters)
	}

	stream.flush()

	return stream.err
}

func (stream *encoder) writeDigits(value uint32, count int) {
	output := stream.buffer[stream.buffered : stream.buffered+count]
	for index := range output {
		output[index] = stream.encoding.forward[value%alphabetSize]
		value /= alphabetSize
	}

	stream.buffered += count
}

func (stream *encoder) flush() {
	if stream.buffered == 0 || stream.err != nil {
		return
	}

	written, err := stream.output.Write(stream.buffer[:stream.buffered])
	if err == nil && written != stream.buffered {
		err = io.ErrShortWrite
	}

	if err != nil {
		stream.err = err

		return
	}

	stream.buffered = 0
}
