package base84

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncodingAppend(t *testing.T) {
	encodedPrefix := []byte("encoded:")

	encoded := StdEncoding.AppendEncode(bytes.Clone(encodedPrefix), []byte("Man"))
	if want := append(bytes.Clone(encodedPrefix), "pRRM"...); !bytes.Equal(encoded, want) {
		t.Errorf("AppendEncode = %q, want %q", encoded, want)
	}

	decodedPrefix := []byte{0xde, 0xad}

	decoded, err := StdEncoding.AppendDecode(bytes.Clone(decodedPrefix), []byte("pRRM"))
	if err != nil {
		t.Fatalf("AppendDecode returned error: %v", err)
	}

	if want := append(bytes.Clone(decodedPrefix), "Man"...); !bytes.Equal(decoded, want) {
		t.Errorf("AppendDecode = %x, want %x", decoded, want)
	}
}

func TestEncodingFixedDestinations(t *testing.T) {
	source := []byte("Man")
	encoded := make([]byte, StdEncoding.EncodedLen(len(source)))

	encodedCount, err := StdEncoding.Encode(encoded, source)
	if err != nil {
		t.Fatalf("Encode with exact destination returned error: %v", err)
	}

	if encodedCount != len(encoded) || string(encoded[:encodedCount]) != "pRRM" {
		t.Errorf("Encode = (%d, %q), want (%d, %q)", encodedCount, encoded[:encodedCount], len(encoded), "pRRM")
	}

	decoded := make([]byte, StdEncoding.DecodedLen(encodedCount))

	decodedCount, err := StdEncoding.Decode(decoded, encoded[:encodedCount])
	if err != nil {
		t.Fatalf("Decode with exact destination returned error: %v", err)
	}

	if decodedCount != len(source) || !bytes.Equal(decoded[:decodedCount], source) {
		t.Errorf("Decode = (%d, %x), want (%d, %x)", decodedCount, decoded[:decodedCount], len(source), source)
	}
}

func TestEncodingFixedDestinationsReturnPartialOutput(t *testing.T) {
	encodeTests := []struct {
		name        string
		source      []byte
		destination []byte
		want        []byte
	}{
		{name: "short", source: []byte("Man "), destination: make([]byte, 4), want: []byte(";dA^")},
		{name: "zero", source: []byte("M"), destination: nil, want: nil},
	}

	for _, test := range encodeTests {
		t.Run("encode "+test.name, func(t *testing.T) {
			written, err := StdEncoding.Encode(test.destination, test.source)
			if !errors.Is(err, ErrNoSpaceLeft) {
				t.Errorf("Encode error = %v, want errors.Is(_, ErrNoSpaceLeft)", err)
			}

			if written != len(test.want) || !bytes.Equal(test.destination, test.want) {
				t.Errorf("Encode = (%d, %q), want (%d, %q)", written, test.destination, len(test.want), test.want)
			}
		})
	}

	decodeTests := []struct {
		name        string
		encoded     []byte
		destination []byte
		want        []byte
	}{
		{name: "short", encoded: []byte(";dA^K"), destination: make([]byte, 3), want: []byte("Man")},
		{name: "zero", encoded: []byte("AA"), destination: nil, want: nil},
	}

	for _, test := range decodeTests {
		t.Run("decode "+test.name, func(t *testing.T) {
			written, err := StdEncoding.Decode(test.destination, test.encoded)
			if !errors.Is(err, ErrNoSpaceLeft) {
				t.Errorf("Decode error = %v, want errors.Is(_, ErrNoSpaceLeft)", err)
			}

			if written != len(test.want) || !bytes.Equal(test.destination, test.want) {
				t.Errorf("Decode = (%d, %x), want (%d, %x)", written, test.destination, len(test.want), test.want)
			}
		})
	}
}

func TestEncodingDecodePreservesPartialOutput(t *testing.T) {
	prefix := []byte{0xde, 0xad}
	wantPartial := []byte{0, 0, 0, 0}

	appended, err := StdEncoding.AppendDecode(bytes.Clone(prefix), []byte("AAAAA/"))
	if !errors.Is(err, ErrInvalidCharacter) {
		t.Errorf("AppendDecode error = %v, want errors.Is(_, ErrInvalidCharacter)", err)
	}

	if want := append(bytes.Clone(prefix), wantPartial...); !bytes.Equal(appended, want) {
		t.Errorf("AppendDecode = %x, want %x", appended, want)
	}

	decoded, err := StdEncoding.DecodeString("AAAAA/")
	if !errors.Is(err, ErrInvalidCharacter) {
		t.Errorf("DecodeString error = %v, want errors.Is(_, ErrInvalidCharacter)", err)
	}

	if !bytes.Equal(decoded, wantPartial) {
		t.Errorf("DecodeString = %x, want %x", decoded, wantPartial)
	}

	decoded, err = Decode("AAAAA/")
	if !errors.Is(err, ErrInvalidCharacter) {
		t.Errorf("Decode error = %v, want errors.Is(_, ErrInvalidCharacter)", err)
	}

	if decoded != nil {
		t.Errorf("Decode = %x, want nil", decoded)
	}
}

func TestEncodingDecodeErrorPrecedence(t *testing.T) {
	tests := []struct {
		wantErr error
		name    string
		encoded string
	}{
		{name: "padding before space", encoded: "A", wantErr: ErrInvalidPadding},
		{name: "character before padding", encoded: "A/", wantErr: ErrInvalidCharacter},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			written, err := StdEncoding.Decode(nil, []byte(test.encoded))
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Decode(%q) error = %v, want errors.Is(_, %v)", test.encoded, err, test.wantErr)
			}

			if written != 0 {
				t.Errorf("Decode(%q) wrote %d bytes, want 0", test.encoded, written)
			}
		})
	}
}
