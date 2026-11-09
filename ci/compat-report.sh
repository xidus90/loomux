#!/bin/sh
# Keeps the issues of the compatibility scout in step with upstream:
#   ci/compat-report.sh <loomux binary> <findings.json> <reports dir>
# findings.json is what `dev compat scout --json` wrote; <reports dir> holds
# one folder per probe run, each with the report.json of the run (absent when
# the run itself failed) and a meta.json naming the platform and the probe.
# Every finding gets one open issue with its title: edited when it exists,
# created otherwise. An open issue of the scout is closed once it is done:
# the list now holds its version, no longer holds a version past end of
# support, or no finding carries its pre-release any more. The scout's issues
# carry the label below. Needs gh (with GH_TOKEN), jq and issues: write. gh's
# output is captured before jq reads it, so that a failing gh stops the
# script (there is no pipefail in sh).
set -eu
bin=${1:?usage: ci/compat-report.sh <loomux binary> <findings.json> <reports dir>}
findings=${2:?findings.json}
reports=${3:?reports dir}
label=compat-scout

gh label create "$label" --force --description "A new upstream version for the compatibility list" >/dev/null

open=$(gh issue list --label "$label" --state open --limit 200 --json number,title)
printf '%s' "$open" | jq -r '.[] | [.number, .title] | @tsv' |
	while IFS="$(printf '\t')" read -r number title; do
		if [ "$("$bin" dev compat scout --resolved "$title" --findings "$findings")" = resolved ]; then
			gh issue close "$number" --comment "Done: the compatibility list now matches this issue, or a newer release took its place."
		fi
	done

body=$(mktemp)
trap 'rm -f "$body"' EXIT
jq -c '.[]' "$findings" | while IFS= read -r finding; do
	title=$(printf '%s' "$finding" | jq -r '.title')
	kind=$(printf '%s' "$finding" | jq -r '.kind')
	line=$(printf '%s' "$finding" | jq -r '.line')
	version=$(printf '%s' "$finding" | jq -r '.version')
	probe=$(printf '%s' "$finding" | jq -c '.probe')
	{
		if [ "$kind" = eol ]; then
			last=$(gh release list --exclude-pre-releases --limit 1 --json tagName --jq '.[0].tagName')
			# shellcheck disable=SC2016 # $0 is awk's
			name=$(printf '%s' "$line" | awk '{ print toupper(substr($0, 1, 1)) substr($0, 2) }')
			printf 'Upstream no longer supports %s **%s** ([endoflife.date](https://endoflife.date/%s)).\n\n' "$name" "$version" "$line"
			# shellcheck disable=SC2016 # the backticks are Markdown
			printf 'Drop its entries from `internal/compat/compat.toml`. The pull request carries at least `release:patch` and names under `Removed` in its `## Changelog`:\n\n'
			printf '> %s %s is no longer supported; %s is the last release that supports it.\n' "$name" "$version" "$last"
		else
			if [ "$kind" = pre ]; then
				# shellcheck disable=SC2016 # the backticks are Markdown
				printf 'Upstream has a pre-release for `%s`: **%s**. It is checked ahead of its release; nothing goes into `internal/compat/compat.toml` until the stable version is out.\n\n' "$line" "$version"
			else
				# shellcheck disable=SC2016 # the backticks are Markdown
				printf 'Upstream has a new version for `%s`: **%s**.\n\n' "$line" "$version"
			fi
			printf 'Result of the probe, per platform:\n\n'
			found=0
			for dir in "$reports"/*/; do
				[ -f "${dir}meta.json" ] || continue
				jq -e --argjson p "$probe" '.probe == $p' "${dir}meta.json" >/dev/null || continue
				found=1
				os=$(jq -r '.os' "${dir}meta.json")
				if [ ! -f "${dir}report.json" ]; then
					printf -- '- %s: the run itself failed, see the workflow log\n' "$os"
				elif jq -e '.green' "${dir}report.json" >/dev/null; then
					printf -- '- %s: green\n' "$os"
				else
					printf -- '- %s: red\n\n~~~\n' "$os"
					jq -r '.lines[]' "${dir}report.json"
					printf '~~~\n\n'
				fi
			done
			[ "$found" = 1 ] || printf -- '- no probe ran\n'
			if [ "$kind" = new ]; then
				# shellcheck disable=SC2016 # the backticks are Markdown
				printf '\nTo support it, add to `internal/compat/compat.toml` once every platform is green:\n\n```toml\n'
				printf '%s' "$finding" | jq -r '.toml'
				printf '```\n'
			fi
		fi
	} >"$body"
	existing=$(gh issue list --label "$label" --state open --limit 200 --json number,title)
	number=$(printf '%s' "$existing" | jq -r --arg t "$title" '[.[] | select(.title == $t)][0].number // empty')
	if [ -n "$number" ]; then
		gh issue edit "$number" --body-file "$body"
	else
		gh issue create --label "$label" --title "$title" --body-file "$body"
	fi
done
