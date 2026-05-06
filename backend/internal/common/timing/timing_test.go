package timing

import "testing"

func TestRoundUpSecToMinute(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, 0},
		{-1, 0},
		{1, 60},
		{59, 60},
		{60, 60},
		{61, 120},
		{119, 120},
		{120, 120},
		{1930, 1980}, // 32:10 → 33:00
		{99_999, 100_020},
	}
	for _, c := range cases {
		if got := RoundUpSecToMinute(c.in); got != c.want {
			t.Errorf("RoundUpSecToMinute(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRoundUpUnixToMinute(t *testing.T) {
	cases := []struct{ in, want int64 }{
		{0, 0},
		{-5, -5},
		{60, 60},
		{61, 120},
		{1_700_000_010, 1_700_000_040},
	}
	for _, c := range cases {
		if got := RoundUpUnixToMinute(c.in); got != c.want {
			t.Errorf("RoundUpUnixToMinute(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
