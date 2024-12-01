package packet

const (
	MaxIconBytes          = 16
	DefaultFrequencyName = "main"
	MaxFrequencyName = 32
)

const (
	PermNoAccess = 0 + iota
	PermRead
	PermReadWrite
	PermMax
)
