package packet

const (
	MaxIconBytes          = 16
	DefaultFrequencyName  = "main"
	DefaultFrequencyColor = "#FFFFFF"
	MaxFrequencyName      = 32
	MaxUserDataBytes      = 8192
)

const (
	PermNoAccess = 0 + iota
	PermRead
	PermReadWrite
	PermMax
)
