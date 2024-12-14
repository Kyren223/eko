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
	for i > 0 && i < len(line) { // TODO: fix this, make sure it's i>=0
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
