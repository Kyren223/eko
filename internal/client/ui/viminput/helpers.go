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

package viminput

import "unicode"

func IsKeyword(r rune) bool {
	alphaLower := 'a' <= r && r <= 'z'
	alphaUpper := 'A' <= r && r <= 'Z'
	numeric := '0' <= r && r <= '9'
	return alphaLower || alphaUpper || numeric || r == '_'
}

func IsGrouped(r1, r2 rune) bool {
	if IsKeyword(r1) {
		return IsKeyword(r2)
	}
	if unicode.IsSpace(r1) {
		return unicode.IsSpace(r2)
	}
	return !IsKeyword(r2) && !unicode.IsSpace(r2)
}

func SearchChar(line []rune, i, dir int, c rune) (index int, ok bool) {
	found := false
	for i >= 0 && i < len(line) {
		if line[i] == c {
			found = true
			break
		}
		i += dir
	}

	return i, found
}

func SearchCharFunc(line []rune, i, dir int, f func(c rune) bool) (index int, ok bool) {
	found := false
	for i >= 0 && i < len(line) {
		if f(line[i]) {
			found = true
			break
		}
		i += dir
	}

	return i, found
}
