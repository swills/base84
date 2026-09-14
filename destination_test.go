package base84

import (
	"bytes"
	"errors"
	"strconv"
	"testing"
)

func TestFixedDestinationsPreserveCanonicalPrefixes(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "44 byte phrase", source: []byte("The quick brown fox jumps over the lazy dog.")},
		{name: "all ff", source: bytes.Repeat([]byte{0xff}, 8)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEncodeDestinations(t, test.source)
			testDecodeDestinations(t, test.source)
		})
	}
}

func testEncodeDestinations(t *testing.T, source []byte) {
	t.Helper()

	encoded := []byte(StdEncoding.EncodeToString(source))

	for capacity := 0; capacity <= len(encoded)+1; capacity++ {
		t.Run("encode capacity "+strconv.Itoa(capacity), func(t *testing.T) {
			destination := make([]byte, capacity)
			written, err := StdEncoding.Encode(destination, source)

			if capacity < len(encoded) {
				if !errors.Is(err, ErrNoSpaceLeft) {
					t.Errorf("Encode error = %v, want ErrNoSpaceLeft", err)
				}

				if written != capacity || !bytes.Equal(destination, encoded[:capacity]) {
					t.Errorf("Encode = (%d, %q), want (%d, %q)", written, destination, capacity, encoded[:capacity])
				}

				return
			}

			if err != nil || written != len(encoded) || !bytes.Equal(destination[:written], encoded) {
				t.Errorf("Encode = (%d, %q, %v), want (%d, %q, nil)", written, destination[:written], err, len(encoded), encoded)
			}
		})
	}
}

func testDecodeDestinations(t *testing.T, source []byte) {
	t.Helper()

	encoded := []byte(StdEncoding.EncodeToString(source))

	decoded, err := StdEncoding.DecodeString(string(encoded))
	if err != nil {
		t.Fatalf("DecodeString returned error: %v", err)
	}

	if !bytes.Equal(decoded, source) {
		t.Fatalf("DecodeString = %x, want %x", decoded, source)
	}

	for capacity := 0; capacity <= len(decoded)+1; capacity++ {
		t.Run("decode capacity "+strconv.Itoa(capacity), func(t *testing.T) {
			destination := make([]byte, capacity)
			written, err := StdEncoding.Decode(destination, encoded)

			if capacity < len(decoded) {
				if !errors.Is(err, ErrNoSpaceLeft) {
					t.Errorf("Decode error = %v, want ErrNoSpaceLeft", err)
				}

				if written != capacity || !bytes.Equal(destination, decoded[:capacity]) {
					t.Errorf("Decode = (%d, %x), want (%d, %x)", written, destination, capacity, decoded[:capacity])
				}

				return
			}

			if err != nil || written != len(decoded) || !bytes.Equal(destination[:written], decoded) {
				t.Errorf("Decode = (%d, %x, %v), want (%d, %x, nil)", written, destination[:written], err, len(decoded), decoded)
			}
		})
	}
}
