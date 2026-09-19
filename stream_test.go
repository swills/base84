package base84

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

var errStreamTest = errors.New("stream test error")

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

type streamErrorWriter struct {
	short bool
}

func (writer streamErrorWriter) Write(source []byte) (int, error) {
	if writer.short {
		return len(source) - 1, nil
	}

	return 0, errStreamTest
}

type dataErrorReader struct {
	data []byte
	done bool
}

type noProgressReader struct{}

func (noProgressReader) Read([]byte) (int, error) {
	return 0, nil
}

func (reader *dataErrorReader) Read(destination []byte) (int, error) {
	if reader.done {
		return 0, errStreamTest
	}

	reader.done = true

	return copy(destination, reader.data), errStreamTest
}

func TestNewEncoderMatchesFixedCodecAcrossWriteBoundaries(t *testing.T) {
	for length := range 97 {
		payload := make([]byte, length)
		for index := range payload {
			payload[index] = byte(index*37 + length)
		}

		for writeSize := 1; writeSize <= 17; writeSize++ {
			var output bytes.Buffer

			stream := NewEncoder(StdEncoding, &output)

			for offset := 0; offset < len(payload); offset += writeSize {
				end := min(offset+writeSize, len(payload))
				written, err := stream.Write(payload[offset:end])

				if err != nil || written != end-offset {
					t.Fatalf("length %d, write size %d: Write = %d, %v", length, writeSize, written, err)
				}
			}

			err := stream.Close()
			if err != nil {
				t.Fatalf("length %d, write size %d: Close: %v", length, writeSize, err)
			}

			if got, want := output.String(), StdEncoding.EncodeToString(payload); got != want {
				t.Fatalf("length %d, write size %d: output = %q, want %q", length, writeSize, got, want)
			}
		}
	}
}

func TestNewEncoderCloseAndErrors(t *testing.T) {
	var output bytes.Buffer

	stream := NewEncoder(StdEncoding, &output)

	_, err := stream.Write([]byte{1})
	if err != nil || output.Len() != 0 {
		t.Fatalf("Write(one byte) error = %v, output length = %d", err, output.Len())
	}

	err = stream.Close()
	if err != nil || output.String() != StdEncoding.EncodeToString([]byte{1}) {
		t.Fatalf("Close error = %v, output = %q", err, output.String())
	}

	closeErr := stream.Close()
	if closeErr != nil {
		t.Errorf("second Close error = %v", closeErr)
	}

	written, err := stream.Write([]byte{2})
	if written != 0 || !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("Write after Close = %d, %v, want 0, %v", written, err, io.ErrClosedPipe)
	}

	for _, test := range []struct {
		writer  io.Writer
		wantErr error
	}{
		{writer: streamErrorWriter{}, wantErr: errStreamTest},
		{writer: streamErrorWriter{short: true}, wantErr: io.ErrShortWrite},
	} {
		failed := NewEncoder(StdEncoding, test.writer)

		_, err = failed.Write([]byte{1})
		if err != nil {
			t.Fatalf("buffered Write error = %v", err)
		}

		err = failed.Close()
		if !errors.Is(err, test.wantErr) {
			t.Errorf("Close error = %v, want %v", err, test.wantErr)
		}

		repeated := failed.Close()
		if !errors.Is(repeated, test.wantErr) {
			t.Errorf("second Close error = %v, want %v", repeated, test.wantErr)
		}
	}
}

func TestNewEncoderSticksErrorFromWrite(t *testing.T) {
	stream := NewEncoder(StdEncoding, streamErrorWriter{})
	payload := make([]byte, streamBufferSize)

	written, err := stream.Write(payload)
	if written == len(payload) || !errors.Is(err, errStreamTest) {
		t.Fatalf("Write = %d, %v, want a partial count and %v", written, err, errStreamTest)
	}

	written, err = stream.Write(payload[:1])
	if written != 0 || !errors.Is(err, errStreamTest) {
		t.Errorf("Write after error = %d, %v, want 0, %v", written, err, errStreamTest)
	}

	err = stream.Close()
	if !errors.Is(err, errStreamTest) {
		t.Errorf("Close after error = %v, want %v", err, errStreamTest)
	}
}

func TestNewEncoderUsesCustomAlphabet(t *testing.T) {
	alphabet := []byte(Alphabet)
	for left, right := 0, len(alphabet)-1; left < right; left, right = left+1, right-1 {
		alphabet[left], alphabet[right] = alphabet[right], alphabet[left]
	}

	encoding, err := NewEncoding(string(alphabet))
	if err != nil {
		t.Fatalf("NewEncoding: %v", err)
	}

	var output bytes.Buffer

	stream := NewEncoder(encoding, &output)

	_, err = stream.Write([]byte("custom alphabet"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = stream.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if got, want := output.String(), encoding.EncodeToString([]byte("custom alphabet")); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
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
		if written != 0 || !errors.Is(err, errStreamTest) {
			t.Errorf("Read after source error = %d, %v, want 0, %v", written, err, errStreamTest)
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
