package base84

import "errors"

var (
	// ErrInvalidAlphabet indicates that an alphabet cannot define an Encoding.
	ErrInvalidAlphabet = errors.New("base84: invalid alphabet")
	// ErrInvalidCharacter indicates that encoded data contains a byte outside the encoding alphabet.
	ErrInvalidCharacter = errors.New("base84: invalid character")
	// ErrInvalidPadding indicates that encoded data is not canonical Base84.
	ErrInvalidPadding = errors.New("base84: invalid padding")
	// ErrNoSpaceLeft indicates that a destination cannot hold the complete result.
	ErrNoSpaceLeft = errors.New("base84: no space left")
)

// AlphabetError describes why NewEncoding rejected an alphabet.
type AlphabetError struct {
	problem string
}

func (alphabetError *AlphabetError) Error() string {
	return ErrInvalidAlphabet.Error() + ": " + alphabetError.problem
}

func (*AlphabetError) Is(target error) bool {
	return target == ErrInvalidAlphabet
}
