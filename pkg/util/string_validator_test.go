package util

import "testing"

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name string
		s    String
		want bool
	}{
		{
			name: "valid",
			s:    String("123"),
			want: true,
		},
		{
			name: "valid",
			s:    String("1.23"),
			want: true,
		},
		{
			name: "invalid",
			s:    String("abc"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.IsNumeric(); got != tt.want {
				t.Errorf("String.IsNumeric() = %v, want %v", got, tt.want)
			}
		})
	}
}
