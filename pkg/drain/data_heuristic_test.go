package drain

import (
	"testing"
)

func TestDataHeuristic(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{token: "1234567890", want: true},
		{token: "1234567890abcdef", want: true},
		{token: "1234567890abcdefg", want: true},
		{token: "abcdefg1", want: false},
		{token: "apple", want: false},
		{token: "application_1234567890", want: true},
		{token: "application_1234567890abcdef", want: true},
	}
	for _, test := range tests {
		t.Run(test.token, func(t *testing.T) {
			if got := defaultIsDataHeuristic(test.token); got != test.want {
				t.Errorf("DefaultIsDataHeuristic(%q) = %v, want %v", test.token, got, test.want)
			}
		})
	}
}
