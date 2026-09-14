package base84

import "fmt"

const (
	// Alphabet is the standard Base84 alphabet.
	Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%&'()+,-;=@[]^_`{}~"

	alphabetSize = 84
	invalidDigit = 0xff
)

// Encoding is an immutable Base84 encoding defined by an 84-byte alphabet.
type Encoding struct {
	forward [alphabetSize]byte
	inverse [256]byte
}

var (
	// StdEncoding is the standard Base84 encoding.
	StdEncoding = newTrustedEncoding(Alphabet)
	// StandardEncoding initially references StdEncoding.
	StandardEncoding = StdEncoding
)

// NewEncoding returns an Encoding for 84 distinct, non-NUL ASCII bytes.
func NewEncoding(alphabet string) (*Encoding, error) {
	if len(alphabet) != alphabetSize {
		return nil, &AlphabetError{
			problem: fmt.Sprintf("alphabet length is %d bytes, want %d", len(alphabet), alphabetSize),
		}
	}

	var observed [128]bool

	for index := range len(alphabet) {
		character := alphabet[index]
		switch {
		case character == 0:
			return nil, &AlphabetError{problem: fmt.Sprintf("NUL at byte %d", index)}
		case character > 127:
			return nil, &AlphabetError{
				problem: fmt.Sprintf("non-ASCII character %#02x at byte %d", character, index),
			}
		case observed[character]:
			return nil, &AlphabetError{
				problem: fmt.Sprintf("duplicate character %q at byte %d", character, index),
			}
		}

		observed[character] = true
	}

	return newTrustedEncoding(alphabet), nil
}

func newTrustedEncoding(alphabet string) *Encoding {
	encoding := &Encoding{}
	for index := range encoding.inverse {
		encoding.inverse[index] = invalidDigit
	}

	for index := range alphabetSize {
		character := alphabet[index]
		encoding.forward[index] = character
		encoding.inverse[character] = byte(index)
	}

	return encoding
}
