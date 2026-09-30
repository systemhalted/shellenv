#!/usr/bin/env bash
# Prints the license text of every third-party Go module linked into the
# shellenv binary for the current GOOS/GOARCH. Release tarballs ship the
# output as THIRD_PARTY_LICENSES, which Apache-2.0 (Cobra) and BSD-3-Clause
# (pflag) require alongside a binary distribution.
set -euo pipefail

cd "$(dirname "$0")/.."

main_module="$(go list -m)"
modules="$(go list -deps -f '{{with .Module}}{{.Path}} {{.Version}} {{.Dir}}{{end}}' ./cmd/shellenv \
  | awk -v main="$main_module" 'NF == 3 && $1 != main' | sort -u)"

echo "The shellenv binary includes the third-party Go modules below."
echo "Each module's license text follows its header."

while read -r path version dir; do
  [ -n "$path" ] || continue
  found=0
  for name in LICENSE LICENSE.txt LICENSE.md COPYING NOTICE NOTICE.txt NOTICE.md; do
    [ -f "$dir/$name" ] || continue
    found=1
    printf '\n%s\n%s %s (%s)\n%s\n\n' \
      "================================================================" \
      "$path" "$version" "$name" \
      "================================================================"
    cat "$dir/$name"
  done
  if [ "$found" -eq 0 ]; then
    echo "error: no license file found for $path $version in $dir" >&2
    exit 1
  fi
done <<< "$modules"
