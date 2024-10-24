package utils

func Clamp(value, minimum, maximum int64) int64 {
	return min(max(value, minimum), maximum)
}
