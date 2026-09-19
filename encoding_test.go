package base84

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

var _ *Encoding = StdEncoding
var _ *Encoding = StandardEncoding
var _ func(string) (*Encoding, error) = NewEncoding
var _ func(*Encoding, []byte) string = (*Encoding).EncodeToString
var _ func(*Encoding, string) ([]byte, error) = (*Encoding).DecodeString
var _ func(*Encoding, int) int = (*Encoding).EncodedLen
var _ func(*Encoding, int) int = (*Encoding).DecodedLen
var _ func(*Encoding, []byte, []byte) []byte = (*Encoding).AppendEncode
var _ func(*Encoding, []byte, []byte) ([]byte, error) = (*Encoding).AppendDecode
var _ func(*Encoding, []byte, []byte) (int, error) = (*Encoding).Encode
var _ func(*Encoding, []byte, []byte) (int, error) = (*Encoding).Decode
var _ func(*Encoding, io.Writer) io.WriteCloser = NewEncoder
var _ func(*Encoding, io.Reader) io.Reader = NewDecoder

func TestAlphabet(t *testing.T) {
	const want = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%&'()+,-;=@[]^_`{}~"

	if Alphabet != want {
		t.Fatalf("Alphabet = %q, want %q", Alphabet, want)
	}

	seen := make(map[byte]struct{}, len(Alphabet))

	for index := range len(Alphabet) {
		character := Alphabet[index]
		if character < 0x21 || character > 0x7e {
			t.Fatalf("Alphabet[%d] = %q, want a printable ASCII character", index, character)
		}

		if _, exists := seen[character]; exists {
			t.Fatalf("Alphabet contains duplicate character %q", character)
		}

		seen[character] = struct{}{}
	}

	if len(Alphabet) != 84 {
		t.Fatalf("len(Alphabet) = %d, want 84", len(Alphabet))
	}
}

func TestStandardEncodingAliases(t *testing.T) {
	if StdEncoding == nil {
		t.Fatal("StdEncoding is nil")
	}

	if StandardEncoding != StdEncoding {
		t.Error("StandardEncoding and StdEncoding refer to different encodings")
	}

	if got := StdEncoding.EncodeToString([]byte("M")); got != "]A" {
		t.Errorf("StdEncoding.EncodeToString(M) = %q, want %q", got, "]A")
	}

	decoded, err := StandardEncoding.DecodeString("]A")
	if err != nil {
		t.Fatalf("StandardEncoding.DecodeString returned error: %v", err)
	}

	if !bytes.Equal(decoded, []byte("M")) {
		t.Errorf("StandardEncoding.DecodeString(]A) = %x, want %x", decoded, []byte("M"))
	}
}

func TestNewEncodingCreatesIndependentEncoding(t *testing.T) {
	reversed := make([]byte, len(Alphabet))
	for index := range len(Alphabet) {
		reversed[index] = Alphabet[len(Alphabet)-1-index]
	}

	encoding, err := NewEncoding(string(reversed))
	if err != nil {
		t.Fatalf("NewEncoding(reversed alphabet) returned error: %v", err)
	}

	if got := encoding.EncodeToString([]byte("M")); got != "G~" {
		t.Errorf("reversed EncodeToString(M) = %q, want %q", got, "G~")
	}

	decoded, err := encoding.DecodeString("G~")
	if err != nil {
		t.Fatalf("reversed DecodeString(G~) returned error: %v", err)
	}

	if !bytes.Equal(decoded, []byte("M")) {
		t.Errorf("reversed DecodeString(G~) = %x, want %x", decoded, []byte("M"))
	}

	if got := StdEncoding.EncodeToString([]byte("M")); got != "]A" {
		t.Errorf("custom encoding changed StdEncoding: got %q, want %q", got, "]A")
	}
}

func TestNewEncodingRejectsInvalidAlphabet(t *testing.T) {
	tests := []struct {
		name     string
		alphabet string
	}{
		{name: "length 83", alphabet: Alphabet[:83]},
		{name: "length 85", alphabet: Alphabet + "?"},
		{name: "duplicate", alphabet: Alphabet[:83] + string(Alphabet[0])},
		{name: "nul", alphabet: "\x00" + Alphabet[1:]},
		{name: "non ASCII", alphabet: "\x80" + Alphabet[1:]},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewEncoding(test.alphabet)

			if !errors.Is(err, ErrInvalidAlphabet) {
				t.Errorf("NewEncoding error = %v, want errors.Is(_, ErrInvalidAlphabet)", err)
			}

			if _, ok := errors.AsType[*AlphabetError](err); !ok {
				t.Errorf("NewEncoding error = %T, want *AlphabetError", err)
			}
		})
	}
}

func TestNewEncodingAcceptsDistinctNonNULASCII(t *testing.T) {
	encoding, err := NewEncoding("\r\n" + Alphabet[2:])
	if err != nil {
		t.Fatalf("NewEncoding with CR and LF returned error: %v", err)
	}

	if got := encoding.EncodeToString([]byte{1}); got != "\n\r" {
		t.Errorf("EncodeToString(01) = %q, want %q", got, "\n\r")
	}

	decoded, err := encoding.DecodeString("\n\r")
	if err != nil {
		t.Fatalf("DecodeString with LF and CR returned error: %v", err)
	}

	if !bytes.Equal(decoded, []byte{1}) {
		t.Errorf("DecodeString with LF and CR = %x, want 01", decoded)
	}
}

func TestEncodingSizes(t *testing.T) {
	encodedLengths := []int{0, 2, 3, 4, 6, 7}
	for sourceLength, want := range encodedLengths {
		if got := StdEncoding.EncodedLen(sourceLength); got != want {
			t.Errorf("EncodedLen(%d) = %d, want %d", sourceLength, got, want)
		}
	}

	decodedLengths := []int{0, 0, 1, 2, 3, 4, 4, 5}
	for encodedLength, want := range decodedLengths {
		if got := StdEncoding.DecodedLen(encodedLength); got != want {
			t.Errorf("DecodedLen(%d) = %d, want %d", encodedLength, got, want)
		}
	}

	for sourceLength := 0; sourceLength <= 64; sourceLength++ {
		source := bytes.Repeat([]byte{0xff}, sourceLength)
		if got, want := len(StdEncoding.EncodeToString(source)), StdEncoding.EncodedLen(sourceLength); got != want {
			t.Errorf("len(EncodeToString(ff x %d)) = %d, want EncodedLen = %d", sourceLength, got, want)
		}
	}

	maximumInt := int(^uint(0) >> 1)
	if got := StdEncoding.EncodedLen(maximumInt); got != maximumInt {
		t.Errorf("EncodedLen(maxInt) = %d, want %d", got, maximumInt)
	}
}
