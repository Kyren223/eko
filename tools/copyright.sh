#!/usr/bin/env bash

# AGPLv3 header template

# Get current year
CURRENT_YEAR=$(date +%Y)

HEADER="// Eko: A terminal based social media platform
// Copyright (C) $CURRENT_YEAR Kyren223
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

# List of paths to exclude (files or directories)
EXCLUDE_PATHS=("internal/data" "tools/test_rate_limit.go")

MODIFIED=false
CHECK_MODE=false
if [[ "$1" == "--check" ]]; then
    CHECK_MODE=true
fi

# Check if .go files exist recursively
if ! find . -type f -name "*.go" | grep -q .; then
    echo "No .go files found in the current directory or subdirectories."
    exit 1
fi

# Apply header to all .go files recursively, excluding specified paths
while IFS= read -r file; do
    # Check if file or its parent directories are in EXCLUDE_PATHS
    skip=false
    for exclude in "${EXCLUDE_PATHS[@]}"; do
        if [[ "$file" == "./$exclude" || "$file" == ./"$exclude"/* ]]; then
            skip=true
            break
        fi
    done

    if [ "$skip" = true ]; then
        if [ $CHECK_MODE = false ]; then
          echo "Skipped $file (in excluded path)"
        fi
        continue
    fi

    head_block=$(head -n 10 "$file")
    copyright_line=$(echo "$head_block" | grep -E 'Copyright \(C\) [0-9]{4}')

    if [[ -z "$copyright_line" ]]; then
        MODIFIED=true
        if $CHECK_MODE; then
            echo "Missing copyright: $file"
        else
            { echo "$HEADER"; echo; cat "$file"; } > "$file.tmp" && mv "$file.tmp" "$file"
            echo "Added license header to $file"
        fi
        continue
    fi

    # Extract the starting year
    start_year=$(echo "$copyright_line" | sed -E 's/.*Copyright \(C\) ([0-9]{4})(-[0-9]{4})?.*/\1/')

    if [[ "$start_year" != "$CURRENT_YEAR" ]]; then
        MODIFIED=true
        if $CHECK_MODE; then
            echo "Needs update: $file"
        else
            sed -i "1,10s/Copyright (C) $start_year/Copyright (C) $start_year-$CURRENT_YEAR/" "$file"
            echo "Updated year in $file"
        fi
    fi
done < <(find . -type f -name "*.go")

if $MODIFIED; then
    exit 1
fi
