#!/usr/bin/env bash
# Works out the version to release, from the release tags of the repository
# and which part of the version to increase: major, minor or patch.
#
# Prints GitHub Actions outputs: previous (the latest released version, or
# 0.1.0 when nothing has been released yet), previous-tag (its tag, empty when
# there is none) and next (the version to release).
set -euo pipefail

increment="${1:?usage: next-version.sh major|minor|patch}"

# Release tags are plain versions such as 1.2.3; a v in front is accepted too.
# The highest one counts, whichever commit it is on.
latest_line=$(git tag --list |
	grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+$' |
	awk '{ version = $0; sub(/^v/, "", version); print version, $0 }' |
	sort -V -k1,1 |
	tail -n 1 || true)

if [[ -z "$latest_line" ]]; then
	previous="0.1.0"
	previous_tag=""
else
	previous="${latest_line%% *}"
	previous_tag="${latest_line##* }"
fi

IFS=. read -r major minor patch <<<"$previous"

case "$increment" in
major)
	major=$((major + 1))
	minor=0
	patch=0
	;;
minor)
	minor=$((minor + 1))
	patch=0
	;;
patch)
	patch=$((patch + 1))
	;;
*)
	echo "unknown increment '$increment': use major, minor or patch" >&2
	exit 1
	;;
esac

next="$major.$minor.$patch"

if git rev-parse --verify --quiet "refs/tags/$next" >/dev/null; then
	echo "the tag $next exists already" >&2
	exit 1
fi

echo "previous=$previous"
echo "previous-tag=$previous_tag"
echo "next=$next"
