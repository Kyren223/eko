package packet

const (
	MaxNetworkNameBytes     = 32
	MaxIconBytes            = 16
	DefaultFrequencyName    = "main"
	DefaultFrequencyColor   = "#FFFFFF"
	MaxFrequencyName        = 32
	MaxUserDataBytes        = 8192
	MaxMessageBytes         = 2000
	MaxUsernameBytes        = 32
	MaxUserDescriptionBytes = 200
	MaxBanReasonBytes       = 64
)

const (
	PermNoAccess = 0 + iota
	PermRead
	PermReadWrite
	PermMax
)
