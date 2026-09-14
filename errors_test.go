package base84

import "testing"

func TestAlphabetErrorError(t *testing.T) {
	alphabetError := &AlphabetError{problem: "duplicate character"}

	if got, want := alphabetError.Error(), "base84: invalid alphabet: duplicate character"; got != want {
		t.Errorf("AlphabetError.Error() = %q, want %q", got, want)
	}
}
