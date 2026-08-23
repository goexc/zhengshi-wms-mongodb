package ui

import "testing"

func TestTotalPages(t *testing.T) {
	tests := []struct {
		total int64
		size  int
		want  int
	}{{0, 20, 1}, {1, 20, 1}, {20, 20, 1}, {21, 20, 2}, {101, 50, 3}, {10, 0, 1}}
	for _, test := range tests {
		if got := totalPages(test.total, test.size); got != test.want {
			t.Fatalf("totalPages(%d, %d) = %d, want %d", test.total, test.size, got, test.want)
		}
	}
}
