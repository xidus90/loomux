#!/usr/bin/env bash
# Release of one merged pull request against the GitHub API. Every decision
# comes from `loomux dev release`; this script only fetches and publishes.
set -euo pipefail
: "${GH_TOKEN:?}" "${REPO:?}" "${LOOMUX:?}"
work=$RUNNER_TEMP/release
mkdir -p "$work"

if [ -n "${DISPATCH_PR:-}" ]; then
	pr=$DISPATCH_PR
else
	# A dispatch without a pull request number has no SHA either.
	if [ -z "${SHA:-}" ]; then
		echo "neither a pull request number nor a commit to look one up from; pass pr when dispatching with mode pr" >&2
		exit 1
	fi
	pr=$(gh api "repos/$REPO/commits/$SHA/pulls" \
		--jq '[.[] | select(.merged_at != null and .base.ref == "master")][0].number // empty')
	if [ -z "$pr" ]; then
		echo "no merged pull request for $SHA; nothing to release"
		exit 0
	fi
fi
gh api "repos/$REPO/pulls/$pr" >"$work/pr.json"
if [ "$(jq -r '.merged_at != null and .base.ref == "master"' "$work/pr.json")" != true ]; then
	echo "pull request #$pr is not merged into master; refusing to release" >&2
	exit 1
fi
jq -r '.body // ""' "$work/pr.json" >"$work/body.md"
labels=$(jq -r '[.labels[].name] | join(",")' "$work/pr.json")
link=$(jq -r .html_url "$work/pr.json")
"$LOOMUX" dev release parse-body --labels "$labels" --body "$work/body.md" >"$work/parsed.json"
bump=$(jq -r .bump "$work/parsed.json")
if [ "$bump" = none ]; then
	echo "pull request #$pr is release:none"
	exit 0
fi
jq -r .changelog "$work/parsed.json" >"$work/notes.md"

# The contents API commits on top of the current master and refuses only when
# CHANGELOG.md itself changed since we read it: then read again, once.
commit=
for attempt in 1 2; do
	git fetch --quiet --force --tags origin master
	git checkout --quiet --force --detach origin/master
	# The tags come from the remote: a checkout the runner reuses keeps tags
	# that were deleted there, and `fetch --tags` never removes them.
	version=$(git ls-remote --tags --refs origin 'v*' | sed 's#.*refs/tags/##' | "$LOOMUX" dev release next-version --bump "$bump")
	# Exit 1 means this version is already in CHANGELOG.md: an earlier run
	# landed the release commit and failed afterwards.
	rc=0
	"$LOOMUX" dev release changelog-insert --version "$version" --date "$(date -u +%F)" \
		--link "$link" --notes "$work/notes.md" || rc=$?
	if [ "$rc" -eq 1 ]; then
		# Dots in the version must match literally in the grep pattern.
		dot='\.'
		commit=$(git log origin/master -1 --format=%H --grep="^chore(release): v${version//./$dot}\$")
		if [ -z "$commit" ]; then
			echo "v$version is in CHANGELOG.md but no commit 'chore(release): v$version' is on master" >&2
			exit 1
		fi
		# The same version number can sit in master's history from an older
		# count: only a release commit that follows this pull request's merge is
		# the one an earlier run of this release left behind.
		merge=$(jq -r '.merge_commit_sha // empty' "$work/pr.json")
		if [ -z "$merge" ] || ! git merge-base --is-ancestor "$merge" "$commit"; then
			echo "release commit $commit for v$version does not follow the merge commit '${merge:-none}' of #$pr; it is from an earlier release" >&2
			exit 1
		fi
		echo "release commit for v$version already on master: $commit"
		break
	elif [ "$rc" -ne 0 ]; then
		exit "$rc"
	fi
	old=$(git rev-parse -q --verify origin/master:CHANGELOG.md || true)
	base64 -w0 CHANGELOG.md >"$work/content.b64"
	jq -n --arg message "chore(release): v$version" --arg old "$old" \
		--rawfile content "$work/content.b64" \
		'{message: $message, branch: "master", content: $content}
		+ (if $old == "" then {} else {sha: $old} end)' >"$work/put.json"
	if commit=$(gh api -X PUT "repos/$REPO/contents/CHANGELOG.md" \
		--input "$work/put.json" --jq .commit.sha); then
		break
	fi
	commit=
	echo "attempt $attempt: updating CHANGELOG.md on master failed (it may have moved)" >&2
done
if [ -z "$commit" ]; then
	echo "release of #$pr failed; rerun release.yml by hand with pr=$pr" >&2
	exit 1
fi

git fetch --quiet origin "$commit"
git checkout --quiet --force --detach "$commit"
"$LOOMUX" dev release build --version "$version" --out "$work/dist"
# The Linux binary runs on this runner: what it reports is what users see.
probe=$work/dist/loomux_${version}_linux_amd64
[ -x "$probe" ] || { echo "no built binary $probe to check" >&2; exit 1; }
said=$("$probe" version)
[ "$said" = "loomux $version" ] || { echo "the built binary says '$said', want 'loomux $version'" >&2; exit 1; }
# A version with a suffix is a pre-release; the suffix alone decides.
pre=()
case $version in *-*) pre=(--prerelease) ;; esac
gh release create "v$version" --target "$commit" --title "v$version" \
	--notes-file "$work/notes.md" "${pre[@]}" "$work"/dist/*
echo "released v$version for #$pr"
