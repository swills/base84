package base84

import "errors"

var (
	ErrInvalidAlphabet  = errors.New("base84: invalid alphabet")
	ErrInvalidCharacter = errors.New("base84: invalid character")
	ErrInvalidPadding   = errors.New("base84: invalid padding")
	ErrNoSpaceLeft      = errors.New("base84: no space left")
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
