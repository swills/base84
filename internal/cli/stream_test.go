package cli

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/swills/base84"
)

var errOutputNotStarted = errors.New("output did not start before the next read")

type chunkReader struct {
	data      []byte
	chunkSize int
}

func (reader *chunkReader) Read(destination []byte) (int, error) {
	if len(reader.data) == 0 {
		return 0, io.EOF
	}

	count := min(len(destination), reader.chunkSize, len(reader.data))
	copy(destination, reader.data[:count])
	reader.data = reader.data[count:]

	return count, nil
}

type observedReader struct {
	output *bytes.Buffer
	data   []byte
	read   bool
}

func (reader *observedReader) Read(destination []byte) (int, error) {
	if reader.read && reader.output.Len() == 0 {
		return 0, errOutputNotStarted
	}

	reader.read = true
	if len(reader.data) == 0 {
		return 0, io.EOF
	}

	count := min(len(destination), len(reader.data))
	copy(destination, reader.data[:count])
	reader.data = reader.data[count:]

	return count, nil
}

func TestRunStreamsAcrossOneByteReads(t *testing.T) {
	payload := make([]byte, 257)
	for index := range payload {
		payload[index] = byte((index*131 + 17) & 0xff)
	}

	var encoded bytes.Buffer

	err := Run(Options{Mode: ModeEncode}, &chunkReader{data: payload, chunkSize: 1}, &encoded)
	if err != nil {
		t.Fatalf("Run(encode one-byte reads) returned error: %v", err)
	}

	wantEncoded := base84.Encode(payload) + "\n"
	if got := encoded.String(); got != wantEncoded {
		t.Fatalf("Run(encode one-byte reads) = %q, want %q", got, wantEncoded)
	}

	var decoded bytes.Buffer

	err = Run(
		Options{Mode: ModeDecode},
		&chunkReader{data: encoded.Bytes(), chunkSize: 1},
		&decoded,
	)
	if err != nil {
		t.Fatalf("Run(decode one-byte reads) returned error: %v", err)
	}

	if got := decoded.Bytes(); !bytes.Equal(got, payload) {
		t.Fatalf("Run(decode one-byte reads) = %x, want %x", got, payload)
	}
}

func TestRunStreamingMatchesCodecAcrossBoundaries(t *testing.T) {
	payload := make([]byte, 96)
	for index := range payload {
		payload[index] = byte((index*131 + 17) & 0xff)
	}

	for length := range [97]struct{}{} {
		wantEncoded := base84.Encode(payload[:length])
		if wantEncoded != "" {
			wantEncoded += "\n"
		}

		for chunkSize := 1; chunkSize <= 17; chunkSize++ {
			var encoded bytes.Buffer

			err := Run(
				Options{Mode: ModeEncode},
				&chunkReader{data: payload[:length], chunkSize: chunkSize},
				&encoded,
			)
			if err != nil {
				t.Fatalf("Run(encode length %d, chunk %d) returned error: %v", length, chunkSize, err)
			}

			if got := encoded.String(); got != wantEncoded {
				t.Fatalf("Run(encode length %d, chunk %d) = %q, want %q", length, chunkSize, got, wantEncoded)
			}

			var decoded bytes.Buffer

			err = Run(
				Options{Mode: ModeDecode},
				&chunkReader{data: encoded.Bytes(), chunkSize: chunkSize},
				&decoded,
			)
			if err != nil {
				t.Fatalf("Run(decode length %d, chunk %d) returned error: %v", length, chunkSize, err)
			}

			if got := decoded.Bytes(); !bytes.Equal(got, payload[:length]) {
				t.Fatalf("Run(decode length %d, chunk %d) = %x, want %x", length, chunkSize, got, payload[:length])
			}
		}
	}
}

func TestRunWritesBeforeReadingToEOF(t *testing.T) {
	payload := make([]byte, 128*1024)
	for index := range payload {
		payload[index] = byte(index)
	}

	tests := []struct {
		name    string
		input   []byte
		options Options
	}{
		{name: "encode", options: Options{Mode: ModeEncode}, input: payload},
		{
			name:    "decode",
			options: Options{Mode: ModeDecode},
			input:   []byte(base84.Encode(payload)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer

			input := &observedReader{data: test.input, output: &output, read: false}

			err := Run(test.options, input, &output)
			if err != nil {
				t.Fatalf("Run(%s) returned error: %v", test.name, err)
			}
		})
	}
}

func TestRunDecodeWritesValidPrefixBeforeError(t *testing.T) {
	var output bytes.Buffer

	err := Run(Options{Mode: ModeDecode}, bytes.NewBufferString("AAAAA/"), &output)
	if !errors.Is(err, base84.ErrInvalidCharacter) {
		t.Fatalf("Run(decode invalid suffix) error = %v, want %v", err, base84.ErrInvalidCharacter)
	}

	if got, want := output.Bytes(), []byte{0, 0, 0, 0}; !bytes.Equal(got, want) {
		t.Errorf("Run(decode invalid suffix) output = %x, want %x", got, want)
	}
}
