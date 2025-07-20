// Eko: A terminal based social media platform
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

package packet

import "github.com/kyren223/eko/pkg/snowflake"

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
	MaxUsersInGetUsers      = 64
)

const (
	PermNoAccess = 0 + iota
	PermRead
	PermReadWrite
	PermMax
)

const (
	PingEveryone = snowflake.ID(0)
	PingAdmins   = snowflake.ID(1)
)
