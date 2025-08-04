#!/usr/bin/env bash

# AGPLv3 header template

# Get current year
CURRENT_YEAR=$(date +%Y)

LICENSE_HEADER_TEMPLATE="// Eko: A terminal-native social media platform
// Copyright (C) {YEARS} Kyren223
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


    head_block=$(head -n 20 "$file")
    license_block=$(echo "$head_block" | sed -n '/Copyright (C) [0-9]\{4\}/,/gnu.org\/licenses\/>/p')
    copyright_line=$(echo "$license_block" | grep -E 'Copyright \(C\) [0-9]{4}')

    START_YEAR=""
    END_YEAR=""
    if [[ -n "$copyright_line" ]]; then
        # Parse years
        if [[ "$copyright_line" =~ Copyright\ \(C\)\ ([0-9]{4})-([0-9]{4}) ]]; then
            START_YEAR="${BASH_REMATCH[1]}"
            END_YEAR="${BASH_REMATCH[2]}"
        elif [[ "$copyright_line" =~ Copyright\ \(C\)\ ([0-9]{4}) ]]; then
            START_YEAR="${BASH_REMATCH[1]}"
        fi
    fi

    # Determine what the correct copyright line should be
    if [[ -z "$START_YEAR" ]]; then
        YEARS="$CURRENT_YEAR"
    elif [[ "$START_YEAR" == "$CURRENT_YEAR" ]]; then
        YEARS="$CURRENT_YEAR"
    else
        YEARS="$START_YEAR-$CURRENT_YEAR"
    fi

    HEADER="${LICENSE_HEADER_TEMPLATE//\{YEARS\}/$YEARS}"

    existing_header=$(sed -n '1,/https:\/\/www\.gnu\.org\/licenses\/>/p' "$file")
    # existing_header_trimmed=$(echo "$existing_header" | sed 's/[[:space:]]*$//')
    # new_header_trimmed=$(echo "$HEADER" | sed 's/[[:space:]]*$//')

    if [[ -z "$license_block" ]]; then
        MODIFIED=true
        if $CHECK_MODE; then
            echo "Missing copyright: $file"
        else
            { echo "$HEADER"; echo; cat "$file"; } > "$file.tmp" && mv "$file.tmp" "$file"
            echo "Added license header to $file"
        fi
      elif [[ "$existing_header" != "$HEADER" ]]; then
        MODIFIED=true
        if $CHECK_MODE; then
            echo "Needs update: $file"
        else
            awk -v header="$HEADER" '
                BEGIN {skipping=1}
                {
                    if (NR == 1) {
                        print header
                        next
                    }

                    if (skipping) {
                        if ($0 ~ /<https:\/\/www\.gnu\.org\/licenses\/>/) {
                            skipping = 0
                            next
                        } else {
                            next
                        }
                    }

                    print
                }
            ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
            echo "Updated license in $file"
        fi
    fi
done < <(find . -type f -name "*.go")

if $MODIFIED; then
    exit 1
fi
