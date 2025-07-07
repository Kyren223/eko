package embeds

import (
	_ "embed"
	"sync/atomic"
)

//go:embed server.crt
var ServerCertificate []byte

//go:embed stub-tos.md
var StubTos string

//go:embed stub-privacy.md
var StubPrivacy string

var (
	TermsOfService atomic.Value
	PrivacyPolicy  atomic.Value
	TosPrivacyHash atomic.Value
)
