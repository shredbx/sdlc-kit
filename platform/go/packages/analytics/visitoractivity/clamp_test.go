package visitoractivity

import "testing"

// TC-A6-1c: clampLimit defaults non-positive limits to 10 and caps any
// client-supplied limit at 500 so TopProperties can't request an unbounded
// grouped set (Part B passes the ?limit= query value straight in).
func TestClampLimit(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 10},
		{-5, 10},
		{10, 10},
		{10000, 500},
		{500, 500},
	}
	for _, c := range cases {
		if got := clampLimit(c.in); got != c.want {
			t.Errorf("clampLimit(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
