package base84

import (
	"bytes"
	"errors"
	"strconv"
	"testing"
)

var authoritativeVectors = []struct {
	name    string
	encoded string
	decoded []byte
}{
	{name: "empty", decoded: []byte{}, encoded: ""},
	{name: "M", decoded: []byte("M"), encoded: "]A"},
	{name: "Ma", decoded: []byte("Ma"), encoded: "tsD"},
	{name: "Man", decoded: []byte("Man"), encoded: "pRRM"},
	{name: "Man space", decoded: []byte("Man "), encoded: ";dA^K"},
	{name: "Hello World", decoded: []byte("Hello, World!"), encoded: "s@Etk'#Qedrxz+hhA"},
	{name: "one zero", decoded: []byte{0x00}, encoded: "AA"},
	{name: "two zeroes", decoded: []byte{0x00, 0x00}, encoded: "AAA"},
	{name: "three zeroes", decoded: []byte{0x00, 0x00, 0x00}, encoded: "AAAA"},
	{name: "four zeroes", decoded: []byte{0x00, 0x00, 0x00, 0x00}, encoded: "AAAAA"},
	{name: "five zeroes", decoded: []byte{0x00, 0x00, 0x00, 0x00, 0x00}, encoded: "AAAAAAA"},
	{name: "one ff", decoded: []byte{0xff}, encoded: "DD"},
	{name: "two ff", decoded: []byte{0xff, 0xff}, encoded: "PYJ"},
	{name: "three ff", decoded: []byte{0xff, 0xff, 0xff}, encoded: "#8Zc"},
	{name: "four ff", decoded: []byte{0xff, 0xff, 0xff, 0xff}, encoded: "rxQLrB"},
	{name: "five ff", decoded: []byte{0xff, 0xff, 0xff, 0xff, 0xff}, encoded: "rxQLrHG"},
	{name: "mixed bytes", decoded: []byte{0x00, 0xe0, 0xff, 0x01}, encoded: "AYy4A"},
	{
		name:    "sequential bytes",
		decoded: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
		encoded: "8%LBB${'eC(No8D-dMGF",
	},
	{
		name:    "pangram",
		decoded: []byte("The quick brown fox jumps over the lazy dog."),
		encoded: "wBB]KdJ}phb-#tmbNB^K!!J_K^3h=l_pj[nJXJLnkB3kkhkT_KM}r1P",
	},
}

func TestEncode(t *testing.T) {
	for _, test := range authoritativeVectors {
		t.Run(test.name, func(t *testing.T) {
			input := bytes.Clone(test.decoded)

			encoded := Encode(input)

			if encoded != test.encoded {
				t.Errorf("Encode(%x) = %q, want %q", test.decoded, encoded, test.encoded)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	for _, test := range authoritativeVectors {
		t.Run(test.name, func(t *testing.T) {
			encoded := test.encoded

			decoded, err := Decode(encoded)
			if err != nil {
				t.Fatalf("Decode(%q) returned error: %v", encoded, err)
			}

			if !bytes.Equal(decoded, test.decoded) {
				t.Errorf("Decode(%q) = %x, want %x", encoded, decoded, test.decoded)
			}
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	tests := []struct {
		wantErr error
		name    string
		encoded string
	}{
		{name: "one digit tail", encoded: "A", wantErr: ErrInvalidPadding},
		{name: "noncanonical two digit tail", encoded: "AE", wantErr: ErrInvalidPadding},
		{name: "noncanonical three digit tail", encoded: "AAK", wantErr: ErrInvalidPadding},
		{name: "noncanonical four digit tail", encoded: "AAAd", wantErr: ErrInvalidPadding},
		{name: "truncated full group", encoded: "rxQLr", wantErr: ErrInvalidPadding},
		{name: "noncanonical punctuation tail", encoded: "oi'-o", wantErr: ErrInvalidPadding},
		{name: "six digit tail", encoded: "AAAAAA", wantErr: ErrInvalidPadding},
		{
			name:    "invalid padding after carried bits",
			encoded: "rxQLrrxQLrrxQLrrxQLrrxQLrrxQLrrxQLrAA",
			wantErr: ErrInvalidPadding,
		},
		{name: "slash", encoded: "AA/", wantErr: ErrInvalidCharacter},
		{name: "space", encoded: "A A", wantErr: ErrInvalidCharacter},
		{name: "nul", encoded: string([]byte{'A', 0x00, 'A'}), wantErr: ErrInvalidCharacter},
		{name: "high byte 80", encoded: string([]byte{'A', 0x80, 'A'}), wantErr: ErrInvalidCharacter},
		{name: "non ASCII byte", encoded: string([]byte{'A', 0xff, 'A'}), wantErr: ErrInvalidCharacter},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded := test.encoded

			_, err := Decode(encoded)

			if !errors.Is(err, test.wantErr) {
				t.Errorf("Decode(%q) error = %v, want errors.Is(_, %v)", encoded, err, test.wantErr)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	lengths := []int{0, 1, 2, 3, 4, 5, 7, 8, 15, 16, 31, 32, 33, 63, 64, 65, 127, 128, 129}

	for _, length := range lengths {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			input := make([]byte, length)
			for index := range input {
				input[index] = byte((index*73 + length*29 + 11) & 0xff)
			}

			decoded, err := Decode(Encode(input))
			if err != nil {
				t.Fatalf("round trip for length %d returned error: %v", length, err)
			}

			if !bytes.Equal(decoded, input) {
				t.Errorf("round trip for length %d = %x, want %x", length, decoded, input)
			}
		})
	}
}
