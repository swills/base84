package base84

const (
	groupCharacters = 5
	groupBits       = 31
	groupMask       = uint64(1<<groupBits - 1)
	lowWordMask     = uint64(1<<32 - 1)
	extraBitLimit   = uint64(2034635776)
)

// EncodedLen returns the maximum encoded length for a source length.
// It returns the largest int when the mathematical bound does not fit in int.
func (*Encoding) EncodedLen(sourceLength int) int {
	return encodedSizeUpperBound(sourceLength)
}

// DecodedLen returns the maximum decoded length for an encoded length.
func (*Encoding) DecodedLen(encodedLength int) int {
	return decodedSizeUpperBound(encodedLength)
}

func encodedSizeUpperBound(sourceLength int) int {
	if sourceLength <= 0 {
		return 0
	}

	fullPeriods := sourceLength / groupBits
	remainder := sourceLength % groupBits
	remainderCharacters := 0

	if remainder > 0 {
		groups := (8*remainder - 1) / groupBits
		tailBits := uint(8*remainder - groupBits*groups)
		remainderCharacters = groups*groupCharacters + tailCharacterCount(tailBits)
	}

	maximumInt := int(^uint(0) >> 1)
	if fullPeriods > (maximumInt-remainderCharacters)/40 {
		return maximumInt
	}

	return fullPeriods*40 + remainderCharacters
}

func decodedSizeUpperBound(encodedLength int) int {
	if encodedLength <= 0 {
		return 0
	}

	chunkBits := [...]int{0, 6, 12, 19, 25}

	return encodedLength/groupCharacters*4 + chunkBits[encodedLength%groupCharacters]/8
}

func groupBitCount(value uint64) uint {
	if value&groupMask < extraBitLimit {
		return groupBits + 1
	}

	return groupBits
}

func tailBitCount(characters int, pendingBits uint, value uint64) (uint, error) {
	var bits uint
	if characters == 1 && pendingBits == 1 {
		bits = 7
	} else {
		for candidate := uint(1); candidate <= 25; candidate++ {
			paddingBits := (8 - candidate%8) % 8
			if tailCharacterCount(candidate) == characters && paddingBits == pendingBits {
				bits = candidate

				break
			}
		}
	}

	if bits == 0 || value>>bits != 0 {
		return 0, ErrInvalidPadding
	}

	if characters > 1 && fitsOneCharacter(bits, value) {
		return 0, ErrInvalidPadding
	}

	return bits, nil
}

func tailCharacterCount(bits uint) int {
	switch {
	case bits <= 6:
		return 1
	case bits <= 12:
		return 2
	case bits <= 19:
		return 3
	case bits <= 25:
		return 4
	default:
		return 5
	}
}

func fitsOneCharacter(bits uint, value uint64) bool {
	return bits <= 7 && value < alphabetSize
}
