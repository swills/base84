package cli

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/swills/base84"
)

var (
	errRead  = errors.New("test read failure")
	errWrite = errors.New("test write failure")
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errRead
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWrite
}

type shortWriter struct{}

func (shortWriter) Write(value []byte) (int, error) {
	return len(value) / 2, nil
}

type countingWriter struct {
	calls int
}

func (writer *countingWriter) Write(value []byte) (int, error) {
	writer.calls++

	return len(value), nil
}

func TestRunEncodeFormatsOutput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wrapWidth int
	}{
		{name: "empty input", input: "", want: "", wrapWidth: 0},
		{
			name:      "unwrapped output",
			input:     "Hello, World!",
			want:      "s@Etk'#Qedrxz+hhA\n",
			wrapWidth: 0,
		},
		{
			name:      "width smaller than output",
			input:     "Hello, World!",
			want:      "s@Et\nk'#Q\nedrx\nz+hh\nA\n",
			wrapWidth: 4,
		},
		{
			name:      "width larger than output",
			input:     "Hello, World!",
			want:      "s@Etk'#Qedrxz+hhA\n",
			wrapWidth: 18,
		},
		{
			name:      "width equal to output",
			input:     "Hello, World!",
			want:      "s@Etk'#Qedrxz+hhA\n",
			wrapWidth: 17,
		},
		{
			name:      "width divides output",
			input:     "Hello, World!",
			want:      "s\n@\nE\nt\nk\n'\n#\nQ\ne\nd\nr\nx\nz\n+\nh\nh\nA\n",
			wrapWidth: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer

			options := Options{Mode: ModeEncode, WrapWidth: test.wrapWidth}

			err := Run(options, bytes.NewBufferString(test.input), &output)
			if err != nil {
				t.Fatalf("Run(%+v, ...) returned error: %v", options, err)
			}

			if got := output.String(); got != test.want {
				t.Errorf("Run(%+v, ...) wrote %q, want %q", options, got, test.want)
			}
		})
	}
}

func TestRunEncodeEmptyDoesNotWrite(t *testing.T) {
	output := &countingWriter{}

	err := Run(Options{Mode: ModeEncode}, bytes.NewReader(nil), output)
	if err != nil {
		t.Fatalf("Run(empty encode) returned error: %v", err)
	}

	if output.calls != 0 {
		t.Errorf("Run(empty encode) called Write %d times, want 0", output.calls)
	}
}

func TestRunDecodeIgnoresASCIIWhitespace(t *testing.T) {
	modes := []struct {
		name          string
		ignoreGarbage bool
	}{
		{name: "default"},
		{name: "ignore garbage", ignoreGarbage: true},
	}

	tests := []struct {
		name       string
		whitespace byte
	}{
		{name: "space", whitespace: ' '},
		{name: "tab", whitespace: '\t'},
		{name: "carriage return", whitespace: '\r'},
		{name: "line feed", whitespace: '\n'},
		{name: "vertical tab", whitespace: '\v'},
		{name: "form feed", whitespace: '\f'},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					var output bytes.Buffer

					input := []byte{'A', 'Y', test.whitespace, 'y', '4', 'A'}
					options := Options{Mode: ModeDecode, IgnoreGarbage: mode.ignoreGarbage}

					err := Run(options, bytes.NewReader(input), &output)
					if err != nil {
						t.Fatalf("Run(%+v, decode %s) returned error: %v", options, test.name, err)
					}

					if got, want := output.Bytes(), []byte{0x00, 0xe0, 0xff, 0x01}; !bytes.Equal(got, want) {
						t.Errorf("Run(%+v, decode %s) wrote %x, want %x", options, test.name, got, want)
					}
				})
			}
		})
	}
}

func TestRunDecodeGarbageSucceedsOnlyWhenIgnored(t *testing.T) {
	tests := []struct {
		wantErr       error
		name          string
		ignoreGarbage bool
	}{
		{name: "default rejects garbage", wantErr: base84.ErrInvalidCharacter},
		{name: "option ignores garbage", ignoreGarbage: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer

			options := Options{Mode: ModeDecode, IgnoreGarbage: test.ignoreGarbage}

			err := Run(options, bytes.NewBufferString("AY/y4A"), &output)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Run(%+v, decode garbage) error = %v, want %v", options, err, test.wantErr)
			}

			if test.wantErr == nil {
				if got, want := output.Bytes(), []byte{0x00, 0xe0, 0xff, 0x01}; !bytes.Equal(got, want) {
					t.Errorf("Run(%+v, decode garbage) wrote %x, want %x", options, got, want)
				}
			}
		})
	}
}

func TestRunDecodeIgnoreGarbagePreservesAlphabetPunctuation(t *testing.T) {
	for index := range len(base84.Alphabet) {
		character := base84.Alphabet[index]
		if character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' {
			continue
		}

		err := Run(Options{Mode: ModeDecode, IgnoreGarbage: true}, bytes.NewReader([]byte{character}), io.Discard)
		if !errors.Is(err, base84.ErrInvalidPadding) {
			t.Errorf(
				"Run(ignore garbage, alphabet punctuation %q) error = %v, want %v",
				character, err, base84.ErrInvalidPadding,
			)
		}
	}
}

func TestRunDecodeIgnoreGarbageKeepsStrictPaddingValidation(t *testing.T) {
	err := Run(Options{Mode: ModeDecode, IgnoreGarbage: true}, bytes.NewBufferString("/A?"), io.Discard)

	if !errors.Is(err, base84.ErrInvalidPadding) {
		t.Errorf("Run(ignore garbage, invalid padding) error = %v, want %v", err, base84.ErrInvalidPadding)
	}
}

func TestRunEncodeIgnoresIgnoreGarbage(t *testing.T) {
	for _, ignoreGarbage := range []bool{false, true} {
		var output bytes.Buffer

		options := Options{Mode: ModeEncode, IgnoreGarbage: ignoreGarbage}

		err := Run(options, bytes.NewBufferString("Hello, World!"), &output)
		if err != nil {
			t.Fatalf("Run(%+v, encode) returned error: %v", options, err)
		}

		if got, want := output.String(), "s@Etk'#Qedrxz+hhA\n"; got != want {
			t.Errorf("Run(%+v, encode) wrote %q, want %q", options, got, want)
		}
	}
}

func TestRunDecodeWritesRawBytesWithoutSuffix(t *testing.T) {
	var output bytes.Buffer

	err := Run(Options{Mode: ModeDecode, WrapWidth: 1}, bytes.NewBufferString("AYy4A\n"), &output)
	if err != nil {
		t.Fatalf("Run(decode) returned error: %v", err)
	}

	if got, want := output.Bytes(), []byte{0x00, 0xe0, 0xff, 0x01}; !bytes.Equal(got, want) {
		t.Errorf("Run(decode) wrote %x, want raw bytes %x", got, want)
	}
}

func TestRunDecodeErrorsStayWrapped(t *testing.T) {
	tests := []struct {
		wantErr error
		name    string
		input   string
	}{
		{name: "invalid character", input: "AA/", wantErr: base84.ErrInvalidCharacter},
		{name: "invalid padding", input: "A", wantErr: base84.ErrInvalidPadding},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Run(Options{Mode: ModeDecode}, bytes.NewBufferString(test.input), io.Discard)

			if !errors.Is(err, test.wantErr) || errors.Unwrap(err) == nil {
				t.Errorf("Run(decode %q) error = %v, want wrapped %v", test.input, err, test.wantErr)
			}
		})
	}
}

func TestRunRejectsNegativeWrapWidth(t *testing.T) {
	err := Run(Options{Mode: ModeEncode, WrapWidth: -1}, bytes.NewBufferString("unread"), io.Discard)

	if !errors.Is(err, ErrNegativeWrapWidth) {
		t.Errorf("Run(negative wrap) error = %v, want errors.Is(_, %v)", err, ErrNegativeWrapWidth)
	}
}

func TestRunReadErrorStaysWrapped(t *testing.T) {
	for _, mode := range []Mode{ModeEncode, ModeDecode} {
		err := Run(Options{Mode: mode}, failingReader{}, io.Discard)

		if !errors.Is(err, errRead) || errors.Unwrap(err) == nil {
			t.Errorf("Run(mode %v, read failure) error = %v, want a wrapped read error", mode, err)
		}
	}
}

func TestRunWriteFailuresStayWrapped(t *testing.T) {
	tests := []struct {
		writer  io.Writer
		wantErr error
		name    string
	}{
		{name: "write error", writer: failingWriter{}, wantErr: errWrite},
		{name: "short write", writer: shortWriter{}, wantErr: io.ErrShortWrite},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Run(Options{Mode: ModeEncode}, bytes.NewBufferString("M"), test.writer)

			if !errors.Is(err, test.wantErr) || errors.Unwrap(err) == nil {
				t.Errorf("Run(%s) error = %v, want wrapped %v", test.name, err, test.wantErr)
			}
		})
	}
}

func TestRunDecodeWriteFailuresStayWrapped(t *testing.T) {
	encoded := bytes.NewBufferString(base84.Encode([]byte("decoded output")))

	err := Run(Options{Mode: ModeDecode}, encoded, shortWriter{})
	if !errors.Is(err, io.ErrShortWrite) || errors.Unwrap(err) == nil {
		t.Errorf("Run(decode short write) error = %v, want wrapped %v", err, io.ErrShortWrite)
	}
}
