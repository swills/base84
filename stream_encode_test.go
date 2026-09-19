package base84

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

var errEncoderStreamTest = errors.New("stream encoder test error")

type streamErrorWriter struct {
	short bool
}

func (writer streamErrorWriter) Write(source []byte) (int, error) {
	if writer.short {
		return len(source) - 1, nil
	}

	return 0, errEncoderStreamTest
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
		{writer: streamErrorWriter{}, wantErr: errEncoderStreamTest},
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
	if written == len(payload) || !errors.Is(err, errEncoderStreamTest) {
		t.Fatalf("Write = %d, %v, want a partial count and %v", written, err, errEncoderStreamTest)
	}

	written, err = stream.Write(payload[:1])
	if written != 0 || !errors.Is(err, errEncoderStreamTest) {
		t.Errorf("Write after error = %d, %v, want 0, %v", written, err, errEncoderStreamTest)
	}

	err = stream.Close()
	if !errors.Is(err, errEncoderStreamTest) {
		t.Errorf("Close after error = %v, want %v", err, errEncoderStreamTest)
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
