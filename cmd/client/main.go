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

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kyren223/eko/embeds"
	"github.com/kyren223/eko/internal/client"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("version:", embeds.Version)
		fmt.Println("commit:", embeds.Commit)
		fmt.Println("build date:", embeds.BuildDate)
		return
	}

	logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	client.Run()
}
