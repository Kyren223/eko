#!/usr/bin/env bash

# AGPLv3 header template
START_YEAR=2025
HEADER="// Eko: A terminal based social media platform
// Copyright (C) $START_YEAR Kyren223
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
// along with this program.  If not, see <https://www.gnu.org/licenses/>."

# Check if .go files exist recursively
if ! find . -type f -name "*.go" | grep -q .; then
    echo "No .go files found in the current directory or subdirectories."
    exit 1
fi

# Get current year
CURRENT_YEAR=$(date +%Y)

# Apply header to all .go files recursively
find . -type f -name "*.go" | while IFS= read -r file; do
    # Skip if file already has Copyright in the first 10 lines
    if ! head -n 10 "$file" | grep -q "Copyright"; then
        # Prepend header with a single newline, preserving original content
        { echo "$HEADER"; echo; cat "$file"; } > "$file.tmp" && mv "$file.tmp" "$file"
        echo "Added license header to $file"
    else
        # Update year in the header if needed and current year differs
        if [ "$CURRENT_YEAR" != "$START_YEAR" ] && head -n 10 "$file" | grep -q "Copyright (C) $START_YEAR"; then
            sed -i "1,10s/Copyright (C) $START_YEAR/Copyright (C) $START_YEAR-$CURRENT_YEAR/" "$file"
            echo "Updated year in $file"
        fi
    fi
done
