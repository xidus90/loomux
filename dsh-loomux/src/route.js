/**
 * Decide where a grep call runs.
 *
 * There is no fallback and no extension list: the search corpus covers every
 * text file git does not ignore, so `loomux` answers every call that stays
 * inside the repository. A path outside the repository is an error with a
 * hint, never a silently empty list.
 */

import { isAbsolute, relative, resolve, sep } from 'node:path'

/**
 * Report whether `path` lies under `prefix`, both slash-separated and
 * repository-relative. The boundary is a path segment, so `internal/code`
 * does not cover `internal/codex/...`.
 */
export function underPrefix(path, prefix) {
  if (prefix === '' || prefix === '.') return true
  if (path === prefix) return true
  return path.startsWith(`${prefix}/`)
}

/**
 * Route one call and return the repository-relative path when one was given.
 *
 * @param {Record<string, unknown>} input
 * @param {string} repoRoot
 * @returns {{ engine: 'loomux', path?: string }}
 */
export function route(input, repoRoot) {
  if (typeof repoRoot !== 'string' || repoRoot === '') {
    throw new Error('grep: the workspace root is unknown')
  }
  const root = resolve(repoRoot)
  const given = input?.path
  if (given === undefined) {
    return { engine: 'loomux' }
  }
  if (typeof given !== 'string' || given.trim().length === 0) {
    throw new Error('path must be a non-empty string when given')
  }
  const absolute = isAbsolute(given) ? resolve(given) : resolve(root, given)
  const rel = relative(root, absolute)
  if (rel === '') {
    return { engine: 'loomux' }
  }
  if (rel === '..' || rel.startsWith(`..${sep}`) || isAbsolute(rel)) {
    throw new Error(
      `path ${given} is outside the workspace root ${root}; the grep tool only searches the repository. ` +
        'Open the file or use a shell command for paths outside it.',
    )
  }
  return { engine: 'loomux', path: rel.split(sep).join('/') }
}
