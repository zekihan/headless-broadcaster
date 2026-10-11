#!/bin/sh
set -eu
version=$(tr -d '[:space:]' < VERSION)
if ! printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo 'Invalid VERSION' >&2
    exit 1
fi
tag=$(git describe --tags --exact-match HEAD)
if [ "$tag" != "v$version" ]; then
    echo 'Release tag must match VERSION' >&2
    exit 1
fi
printf '%s\n' "$tag"
