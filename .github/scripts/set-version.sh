#!/usr/bin/env bash
# Writes a version into the application, where the about dialog shows it.
# The release workflow runs it before building and before committing, so the
# released binaries and the tagged commit carry the same version.
set -euo pipefail

version="${1:?usage: set-version.sh 1.2.3}"
file="internal/app/version.go"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "'$version' is not a version such as 1.2.3" >&2
	exit 1
fi

# No anchor at the end of the line: a checkout on Windows can end it with \r.
sed -i.bak -E "s/^(const applicationVersion = )\"[^\"]*\"/\1\"$version\"/" "$file"
rm -f "$file.bak"

if ! grep -q "^const applicationVersion = \"$version\"" "$file"; then
	echo "the version line in $file was not found" >&2
	exit 1
fi

echo "set the version to $version"
