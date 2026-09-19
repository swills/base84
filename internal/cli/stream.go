package cli

import "github.com/swills/base84"

const (
	streamBufferSize      = 32 * 1024
	streamGroupCharacters = 5
	streamGroupBits       = 31
	streamGroupMask       = uint64(1<<streamGroupBits - 1)
	streamLowWordMask     = uint64(1<<32 - 1)
	streamExtraBitLimit   = uint64(2034635776)
	streamInvalidDigit    = 0xff
	streamAlphabetSize    = 84
)

var standardDigits = makeStandardDigits()

func makeStandardDigits() [256]byte {
	digits := [256]byte{}
	for index := range digits {
		digits[index] = streamInvalidDigit
	}

	digit := byte(0)
	for index := range base84.Alphabet {
		digits[base84.Alphabet[index]] = digit
		digit++
	}

	return digits
}

func streamGroupBitCount(value uint64) uint {
	if value&streamGroupMask < streamExtraBitLimit {
		return streamGroupBits + 1
	}

	return streamGroupBits
}

func streamTailCharacterCount(bits uint) int {
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

func streamFitsOneCharacter(bits uint, value uint64) bool {
	return bits <= 7 && value < uint64(streamAlphabetSize)
}

func streamTailBitCount(characters int, pendingBits uint, value uint64) (uint, error) {
	var bits uint
	if characters == 1 && pendingBits == 1 {
		bits = 7
	} else {
		for candidate := uint(1); candidate <= 25; candidate++ {
			paddingBits := (8 - candidate%8) % 8
			if streamTailCharacterCount(candidate) == characters && paddingBits == pendingBits {
				bits = candidate

				break
			}
		}
	}

	if bits == 0 || value>>bits != 0 {
		return 0, base84.ErrInvalidPadding
	}

	if characters > 1 && streamFitsOneCharacter(bits, value) {
		return 0, base84.ErrInvalidPadding
	}

	return bits, nil
}
