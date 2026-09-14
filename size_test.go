package base84

import "testing"

func TestGroupBitCountThreshold(t *testing.T) {
	tests := []struct {
		value uint64
		want  uint
	}{
		{value: extraBitLimit - 1, want: 32},
		{value: extraBitLimit, want: 31},
	}

	for _, test := range tests {
		if got := groupBitCount(test.value); got != test.want {
			t.Errorf("groupBitCount(%d) = %d, want %d", test.value, got, test.want)
		}
	}
}
