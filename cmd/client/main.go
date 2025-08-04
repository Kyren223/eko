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

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kyren223/eko/embeds"
	"github.com/kyren223/eko/internal/client"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("version:", embeds.Version)
		fmt.Println("commit:", embeds.Commit)
		buildDate := embeds.BuildDate
		if buildDate != "unknown" {
			t, err := strconv.ParseInt(buildDate, 10, 64)
			if err == nil {
				buildDate = time.Unix(t, 0).Format("2006-01-02")
			}
		}
		fmt.Println("build date:", buildDate)
		return
	}

	client.Run()
}
