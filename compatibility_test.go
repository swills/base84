package base84

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/rand"
	"testing"
)

func TestRoundTripAllInputsUpToTwoBytes(t *testing.T) {
	assertRoundTrip := func(input []byte) {
		encoded := Encode(input)

		decoded, err := Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(Encode(%x)) returned error: %v", input, err)
		}

		if !bytes.Equal(decoded, input) {
			t.Fatalf("Decode(Encode(%x)) = %x", input, decoded)
		}
	}

	assertRoundTrip(nil)

	input := [2]byte{}
	for first := range 256 {
		input[0] = byte(first)
		assertRoundTrip(input[:1])
	}

	for value := range 1 << 16 {
		input[0] = byte(value)
		input[1] = byte(value >> 8)
		assertRoundTrip(input[:])
	}
}

func TestCanonicalityAllEncodedStringsUpToThreeCharacters(t *testing.T) {
	assertCanonical := func(encoded string) {
		decoded, err := Decode(encoded)
		if err != nil {
			if errors.Is(err, ErrInvalidCharacter) {
				t.Fatalf("Decode(%q) returned ErrInvalidCharacter for alphabet-only input", encoded)
			}

			if !errors.Is(err, ErrInvalidPadding) {
				t.Fatalf("Decode(%q) returned unexpected error: %v", encoded, err)
			}

			return
		}

		if reencoded := Encode(decoded); reencoded != encoded {
			t.Fatalf("Encode(Decode(%q)) = %q", encoded, reencoded)
		}
	}

	count := 1

	assertCanonical("")

	encoded := [3]byte{}
	for first := range len(Alphabet) {
		encoded[0] = Alphabet[first]
		assertCanonical(string(encoded[:1]))

		count++

		for second := range len(Alphabet) {
			encoded[1] = Alphabet[second]
			assertCanonical(string(encoded[:2]))

			count++

			for third := range len(Alphabet) {
				encoded[2] = Alphabet[third]
				assertCanonical(string(encoded[:]))

				count++
			}
		}
	}

	if count != 1+84+7056+592704 {
		t.Fatalf("enumerated %d encoded strings", count)
	}
}

func TestRandomRoundTrips(t *testing.T) {
	const seed int64 = 0x4b1d84

	random := rand.New(rand.NewSource(seed))

	for iteration := range 50000 {
		input := make([]byte, random.Intn(256))

		_, err := random.Read(input)
		if err != nil {
			t.Fatalf("seed=%d iteration=%d random read: %v", seed, iteration, err)
		}

		decoded, err := Decode(Encode(input))
		if err != nil {
			t.Fatalf("seed=%d iteration=%d length=%d decode error: %v", seed, iteration, len(input), err)
		}

		if !bytes.Equal(decoded, input) {
			t.Fatalf("seed=%d iteration=%d round trip = %x, want %x", seed, iteration, decoded, input)
		}
	}
}

func TestRandomAlterationsRemainStrict(t *testing.T) {
	const seed int64 = 0x57a1c784

	random := rand.New(rand.NewSource(seed))

	for iteration := range 50000 {
		input := make([]byte, random.Intn(33))

		_, err := random.Read(input)
		if err != nil {
			t.Fatalf("seed=%d iteration=%d random read: %v", seed, iteration, err)
		}

		altered := []byte(Encode(input))
		character := Alphabet[random.Intn(len(Alphabet))]
		altered = randomlyAlterEncoding(t, random, altered, character, seed, iteration)

		decoded, err := Decode(string(altered))
		if err != nil {
			if !errors.Is(err, ErrInvalidPadding) {
				t.Fatalf("seed=%d iteration=%d Decode(%q) error = %v", seed, iteration, altered, err)
			}

			continue
		}

		if reencoded := Encode(decoded); reencoded != string(altered) {
			t.Fatalf("seed=%d iteration=%d Encode(Decode(%q)) = %q", seed, iteration, altered, reencoded)
		}
	}
}

func randomlyAlterEncoding(
	t *testing.T,
	random *rand.Rand,
	encoded []byte,
	character byte,
	seed int64,
	iteration int,
) []byte {
	t.Helper()

	alteration := random.Intn(3)
	switch alteration {
	case 0:
		if len(encoded) > 0 {
			encoded[random.Intn(len(encoded))] = character
		}
	case 1:
		encoded = append(encoded, character)
	case 2:
		if len(encoded) > 0 {
			encoded = encoded[:len(encoded)-1]
		}
	default:
		t.Fatalf("seed=%d iteration=%d unexpected alteration %d", seed, iteration, alteration)
	}

	return encoded
}

func TestThresholdAdjacentFourByteGroups(t *testing.T) {
	limit := uint32(extraBitLimit)
	tests := []struct {
		encoded string
		value   uint32
	}{
		{value: limit - 1, encoded: "ni'-o"},
		{value: limit, encoded: "oi'-oA"},
		{value: limit + 1, encoded: "pi'-oA"},
		{value: 1 << 31, encoded: "sxQLr"},
		{value: limit - 1 | 1<<31, encoded: "~~~~~"},
		{value: limit | 1<<31, encoded: "oi'-oB"},
		{value: limit + 1 | 1<<31, encoded: "pi'-oB"},
	}

	for _, test := range tests {
		input := make([]byte, 4)
		binary.LittleEndian.PutUint32(input, test.value)

		if encoded := Encode(input); encoded != test.encoded {
			t.Errorf("Encode(little-endian %#08x) = %q, want %q", test.value, encoded, test.encoded)
		}

		decoded, err := Decode(test.encoded)
		if err != nil {
			t.Errorf("Decode(%q) returned error: %v", test.encoded, err)

			continue
		}

		if !bytes.Equal(decoded, input) {
			t.Errorf("Decode(%q) = %x, want %x", test.encoded, decoded, input)
		}
	}
}

func TestSizeBoundInvariants(t *testing.T) {
	t.Run("small bounds", testSmallSizeBounds)
	t.Run("round trips", testSizeBoundRoundTrips)
	t.Run("large bounds", testLargeSizeBounds)
}

func testSmallSizeBounds(t *testing.T) {
	wantEncoded := [...]int{0, 2, 3, 4, 6, 7}
	for sourceLength, want := range wantEncoded {
		if got := encodedSizeUpperBound(sourceLength); got != want {
			t.Errorf("encodedSizeUpperBound(%d) = %d, want %d", sourceLength, got, want)
		}
	}

	wantDecoded := [...]int{0, 0, 1, 2, 3, 4, 4, 5}
	for encodedLength, want := range wantDecoded {
		if got := decodedSizeUpperBound(encodedLength); got != want {
			t.Errorf("decodedSizeUpperBound(%d) = %d, want %d", encodedLength, got, want)
		}
	}
}

func testSizeBoundRoundTrips(t *testing.T) {
	input := bytes.Repeat([]byte{0xff}, 255)
	for length := 0; length <= len(input); length++ {
		encoded := Encode(input[:length])
		if len(encoded) != encodedSizeUpperBound(length) {
			t.Fatalf("len(Encode(ff x %d)) = %d, bound = %d", length, len(encoded), encodedSizeUpperBound(length))
		}

		decoded, err := Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(Encode(ff x %d)) returned error: %v", length, err)
		}

		if len(decoded) > decodedSizeUpperBound(len(encoded)) {
			t.Fatalf(
				"decoded length %d exceeds bound %d for encoded length %d",
				len(decoded),
				decodedSizeUpperBound(len(encoded)),
				len(encoded),
			)
		}
	}
}

func testLargeSizeBounds(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	if got, want := encodedSizeUpperBound(maxInt/40*31), maxInt/40*40; got != want {
		t.Errorf("large encoded size bound = %d, want %d", got, want)
	}

	if got, minimum := decodedSizeUpperBound(maxInt), maxInt/5*4; got < minimum {
		t.Errorf("large decoded size bound = %d, want at least %d", got, minimum)
	}
}
