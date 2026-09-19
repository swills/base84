package base84

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

var errDecoderStreamTest = errors.New("stream decoder test error")

type limitedChunkReader struct {
	input []byte
	size  int
}

func (reader *limitedChunkReader) Read(destination []byte) (int, error) {
	if len(reader.input) == 0 {
		return 0, io.EOF
	}

	count := min(len(destination), reader.size, len(reader.input))
	copy(destination, reader.input[:count])
	reader.input = reader.input[count:]

	return count, nil
}

type dataErrorReader struct {
	data []byte
	done bool
}

func (reader *dataErrorReader) Read(destination []byte) (int, error) {
	if reader.done {
		return 0, errDecoderStreamTest
	}

	reader.done = true

	return copy(destination, reader.data), errDecoderStreamTest
}

type noProgressReader struct{}

func (noProgressReader) Read([]byte) (int, error) {
	return 0, nil
}

func TestNewDecoderMatchesFixedCodecAcrossReadBoundaries(t *testing.T) {
	for length := range 97 {
		payload := make([]byte, length)
		for index := range payload {
			payload[index] = byte(index*29 + length)
		}

		encoded := []byte(StdEncoding.EncodeToString(payload))
		for chunkSize := 1; chunkSize <= 17; chunkSize++ {
			input := &limitedChunkReader{input: append([]byte(nil), encoded...), size: chunkSize}

			decoded, err := io.ReadAll(NewDecoder(StdEncoding, input))
			if err != nil {
				t.Fatalf("length %d, chunk size %d: ReadAll: %v", length, chunkSize, err)
			}

			if !bytes.Equal(decoded, payload) {
				t.Fatalf("length %d, chunk size %d: decoded = %x, want %x", length, chunkSize, decoded, payload)
			}
		}
	}
}

func TestNewDecoderStrictInputAndPartialOutput(t *testing.T) {
	for _, whitespace := range []byte{' ', '\t', '\r', '\n', '\v', '\f'} {
		_, err := io.ReadAll(NewDecoder(StdEncoding, bytes.NewReader([]byte{'A', whitespace})))
		if !errors.Is(err, ErrInvalidCharacter) {
			t.Errorf("whitespace %q error = %v, want %v", whitespace, err, ErrInvalidCharacter)
		}
	}

	decoded, err := io.ReadAll(NewDecoder(StdEncoding, bytes.NewBufferString("AAAAA/")))
	if !errors.Is(err, ErrInvalidCharacter) {
		t.Fatalf("invalid suffix error = %v, want %v", err, ErrInvalidCharacter)
	}

	if want := []byte{0, 0, 0, 0}; !bytes.Equal(decoded, want) {
		t.Errorf("invalid suffix decoded = %x, want %x", decoded, want)
	}

	tests := []struct {
		wantErr error
		input   string
	}{
		{input: "AAAA/", wantErr: ErrInvalidCharacter},
		{input: "A/", wantErr: ErrInvalidCharacter},
		{input: "A", wantErr: ErrInvalidPadding},
		{input: "AE", wantErr: ErrInvalidPadding},
		{input: "rxQLr", wantErr: ErrInvalidPadding},
		{input: "AAAAArxQLr", wantErr: ErrInvalidPadding},
	}

	for _, test := range tests {
		_, err = io.ReadAll(NewDecoder(StdEncoding, bytes.NewBufferString(test.input)))
		if !errors.Is(err, test.wantErr) {
			t.Errorf("NewDecoder(%q) error = %v, want %v", test.input, err, test.wantErr)
		}
	}
}

func TestNewDecoderUsesCustomAlphabetVerbatim(t *testing.T) {
	encoding, err := NewEncoding("\r\n" + Alphabet[2:])
	if err != nil {
		t.Fatalf("NewEncoding: %v", err)
	}

	want := []byte{84}
	encoded := encoding.EncodeToString(want)

	if !strings.ContainsAny(encoded, "\r\n") {
		t.Fatalf("custom encoding %q does not exercise CR or LF", encoded)
	}

	decoded, err := io.ReadAll(NewDecoder(encoding, bytes.NewBufferString(encoded)))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(decoded, want) {
		t.Errorf("decoded = %x, want %x", decoded, want)
	}
}

func TestNewDecoderSticksSourceErrorAfterOutput(t *testing.T) {
	encoded := []byte(StdEncoding.EncodeToString([]byte{1, 2, 3, 4}))
	stream := NewDecoder(StdEncoding, &dataErrorReader{data: encoded})
	decoded := make([]byte, 4)

	written, err := io.ReadFull(stream, decoded)
	if err != nil || written != len(decoded) {
		t.Fatalf("ReadFull = %d, %v", written, err)
	}

	for range 2 {
		written, err = stream.Read(decoded)
		if written != 0 || !errors.Is(err, errDecoderStreamTest) {
			t.Errorf("Read after source error = %d, %v, want 0, %v", written, err, errDecoderStreamTest)
		}
	}
}

func TestNewDecoderReadBoundaries(t *testing.T) {
	payload := []byte("small destination")
	stream := NewDecoder(StdEncoding, bytes.NewBufferString(StdEncoding.EncodeToString(payload)))

	written, err := stream.Read(nil)
	if written != 0 || err != nil {
		t.Fatalf("Read(nil) = %d, %v, want 0, nil", written, err)
	}

	var decoded bytes.Buffer

	buffer := make([]byte, 1)

	for {
		written, err = stream.Read(buffer)
		decoded.Write(buffer[:written])

		if err != nil {
			break
		}
	}

	if err != io.EOF {
		t.Fatalf("terminal error = %v, want %v", err, io.EOF)
	}

	if !bytes.Equal(decoded.Bytes(), payload) {
		t.Errorf("decoded = %q, want %q", decoded.Bytes(), payload)
	}
}

func TestNewDecoderReportsNoProgress(t *testing.T) {
	stream := NewDecoder(StdEncoding, noProgressReader{})

	buffer := make([]byte, 1)

	written, err := stream.Read(buffer)
	if written != 0 || !errors.Is(err, io.ErrNoProgress) {
		t.Errorf("Read = %d, %v, want 0, %v", written, err, io.ErrNoProgress)
	}
}
