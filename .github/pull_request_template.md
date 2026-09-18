<!--
The pr-label check (.github/workflows/pr-label.yml, rules in
internal/release) fails this pull request unless:

- every commit header is a Conventional Commit,
  <type>[(<scope>)][!]: <description>, type one of feat, fix, build, chore,
  ci, docs, style, refactor, perf, test, revert (headers git writes itself,
  such as "Merge ..." and "fixup! ...", are exempt);
- it carries exactly one label, release:major, release:minor, release:patch
  or release:none, never lower than its commits ask for: "!" or a
  BREAKING CHANGE footer needs major, feat minor, fix patch;
- unless the label is release:none, the "## Changelog" section below holds
  only "### <heading>" and "- <entry>" lines, at least one entry, and every
  heading is one of Added, Changed, Deprecated, Removed, Fixed, Security.
  Delete the headings you do not use; with release:none delete the section.

The same check runs locally before you push:
  go run ./cmd/loomux dev release parse-body --labels release:<level> \
    --body <file> --commits <json array of the commit messages>
The full procedure is README.md, "Opening a pull request".

Re-running an old check run re-reads the body and labels it started with.
After fixing them, let the edit trigger a fresh run instead.
-->
Release: <level> — <one-sentence reason>

## Summary
- <what changes, from the user's point of view>

## Changelog
### Added
- <entry>
### Changed
- <entry>
### Fixed
- <entry>
