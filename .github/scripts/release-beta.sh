#!/usr/bin/env bash
# Beta release of one ref against the GitHub API. It commits nothing and
# touches no changelog: the tag is the only trace a beta leaves in the history.
set -euo pipefail
: "${GH_TOKEN:?}" "${REPO:?}" "${LOOMUX:?}" "${REF:?}" "${BUMP:?}"
work=$RUNNER_TEMP/release
mkdir -p "$work"

git fetch --quiet origin --end-of-options "$REF"
# The commit behind FETCH_HEAD: an annotated tag as REF is a tag object.
commit=$(git rev-parse 'FETCH_HEAD^{commit}')

# The tags come from the remote: a checkout the runner reuses keeps tags that
# were deleted there, and `fetch --tags` never removes them.
version=$(git ls-remote --tags --refs origin 'v*' | sed 's#.*refs/tags/##' | "$LOOMUX" dev release next-beta --bump "$BUMP")

git checkout --quiet --force --detach "$commit"
"$LOOMUX" dev release build --version "$version" --out "$work/dist"
# The Linux binary runs on this runner: what it reports is what users see.
probe=$work/dist/loomux_${version}_linux_amd64
[ -x "$probe" ] || { echo "no built binary $probe to check" >&2; exit 1; }
said=$("$probe" version)
[ "$said" = "loomux $version" ] || { echo "the built binary says '$said', want 'loomux $version'" >&2; exit 1; }
echo "Beta from $REF at $commit." >"$work/notes.md"
gh release create "v$version" --target "$commit" --title "v$version" \
	--notes-file "$work/notes.md" --prerelease "$work"/dist/*
echo "released v$version from $REF at $commit"
