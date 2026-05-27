package timing

func RoundUpSecToMinute(sec int) int {
	if sec <= 0 {
		return 0
	}
	r := sec % 60
	if r == 0 {
		return sec
	}
	return sec + (60 - r)
}

func RoundUpUnixToMinute(unix int64) int64 {
	if unix <= 0 {
		return unix
	}
	r := unix % 60
	if r == 0 {
		return unix
	}
	return unix + (60 - r)
}
