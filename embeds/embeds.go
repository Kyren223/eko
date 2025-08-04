// Eko: A terminal-native social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package embeds

import (
	_ "embed"
	"sync/atomic"
)

//go:embed VERSION
var Version string

var (
	Commit    string = "unknown"
	BuildDate string = "unknown"
)

//go:embed install.sh
var Installer string

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

func init() {
	// Strip \n from file import
	Version = Version[:len(Version)-1]
}
