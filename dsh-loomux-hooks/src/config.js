/**
 * The hook definition resolver: where hook definitions live and how the
 * Claude dialect of a hooks file becomes the normalized entry list a plugin
 * registers.
 *
 * The chain, in order: the explicit config path, then
 * `<workdir>/.loomux/dsh/hooks.json`, then `<workdir>/.claude/settings.json`,
 * then `<workdir>/hooks/hooks.json` — the codex position `internal/hosts/codex.go`
 * documents as "hooks/hooks.json" relative to the project. The first existing
 * file wins: a file that exists but fails to parse or validate resolves to
 * null and its reason is reported through the onWarn callback; it does not
 * fall through to the next position, because a project that placed a hooks
 * file there meant that file to speak. No file anywhere resolves to null.
 *
 * The accepted dialect is the Claude settings shape, where `hooks.<Event>`
 * maps either to a list of matcher groups `{ matcher, hooks: [handler...] }`
 * or — the flat variant — directly to a list of handler objects. Both
 * normalize to `{ event, matcher, command, timeoutMs }`: a missing matcher
 * becomes `.*` for every event including SessionStart and Stop, `timeout` is
 * SECONDS in the file and lands as `timeoutMs` (a missing timeout stays
 * undefined), and a handler of any type other than "command" is dropped with
 * a warning. Event names are kept as written: the parser stores what the
 * file says, and the plugin registers the events it knows.
 */

import fs from 'node:fs'
import path from 'node:path'

/** Matcher used when a group and a handler both define none. */
const DEFAULT_MATCHER = '.*'

/** Sink for warnings when the caller wires no logger of its own. */
const noop = () => { /* the caller did not ask to hear about broken files */ }

/**
 * Sentinel separating "reject the whole file" from a merely dropped handler
 * (null): one broken matcher or timeout invalidates every entry the file claims.
 */
const REJECT = Symbol('reject')

/**
 * Resolve the hook definitions for one project along the dsh chain.
 *
 * @param {string|null|undefined} explicitPath the configured config path; first chain position when it exists
 * @param {string} workdir the project directory the default positions hang under
 * @param {(warning: string) => void} [onWarn] sink for every reason a file is skipped or rejected
 * @returns {{ entries: Array<{ event: string, matcher: string, command: string, timeoutMs: number|undefined }>, sourcePath: string } | null}
 */
export function resolveConfig(explicitPath, workdir, onWarn = noop) {
  const chain = [
    explicitPath,
    path.join(workdir, '.loomux', 'dsh', 'hooks.json'),
    path.join(workdir, '.claude', 'settings.json'),
    path.join(workdir, 'hooks', 'hooks.json'),
  ].filter(Boolean)
  for (const file of chain) {
    if (!fs.existsSync(file)) continue
    // The first existing file speaks: a rejection ends the chain here and now,
    // it never falls through to the next position.
    const entries = parseFile(file, onWarn)
    if (entries === REJECT) return null
    return { entries, sourcePath: file }
  }
  return null
}

/**
 * Read one hooks file and validate its dialect.
 *
 * @param {string} file the existing file to read
 * @param {(warning: string) => void} warn
 * @returns {Array<object>|typeof REJECT} the normalized entries, or REJECT after reporting the reason
 */
function parseFile(file, warn) {
  let doc
  try {
    doc = JSON.parse(fs.readFileSync(file, 'utf8'))
  } catch (error) {
    warn(`dsh-loomux-hooks: ${file} ignored: invalid JSON: ${error.message}`)
    return REJECT
  }
  const hooks = doc?.hooks
  if (typeof hooks !== 'object' || hooks === null || Array.isArray(hooks)) {
    warn(`dsh-loomux-hooks: ${file} ignored: it has no "hooks" object`)
    return REJECT
  }
  const entries = []
  for (const [event, items] of Object.entries(hooks)) {
    if (!Array.isArray(items)) {
      warn(`dsh-loomux-hooks: ${file} rejected: hooks.${event} is not a list`)
      return REJECT
    }
    const parsed = parseEvent(file, event, items, warn)
    if (parsed === REJECT) return REJECT
    entries.push(...parsed)
  }
  return entries
}

/**
 * Normalize one event's list, accepting both the matcher-group variant and
 * the flat handler variant: an item with a "hooks" property is a group whose
 * matcher governs its handlers, any other object is a handler of its event.
 *
 * @param {string} file
 * @param {string} event
 * @param {unknown[]} items
 * @param {(warning: string) => void} warn
 * @returns {Array<object>|typeof REJECT}
 */
function parseEvent(file, event, items, warn) {
  const entries = []
  for (const item of items) {
    if (item === null || typeof item !== 'object' || Array.isArray(item)) {
      warn(`dsh-loomux-hooks: ${file} rejected: hooks.${event} item is not an object`)
      return REJECT
    }
    if (Object.hasOwn(item, 'hooks')) {
      if (!Array.isArray(item.hooks)) {
        warn(`dsh-loomux-hooks: ${file} rejected: hooks.${event} matcher group has no "hooks" list`)
        return REJECT
      }
      for (const handler of item.hooks) {
        const entry = parseHandler(file, event, handler, item.matcher, warn)
        if (entry === REJECT) return REJECT
        if (entry !== null) entries.push(entry)
      }
    } else {
      const entry = parseHandler(file, event, item, undefined, warn)
      if (entry === REJECT) return REJECT
      if (entry !== null) entries.push(entry)
    }
  }
  return entries
}

/**
 * Normalize one handler object. Returns null for a dropped handler: only
 * type "command" runs, every other type leaves a warning behind. A group
 * matcher governs its handlers; a handler matcher applies only without one.
 *
 * @param {string} file
 * @param {string} event
 * @param {unknown} handler
 * @param {unknown} groupMatcher the matcher of the group this handler sits in, if any
 * @param {(warning: string) => void} warn
 * @returns {object|null|typeof REJECT} the entry, null when dropped, REJECT for the whole file
 */
function parseHandler(file, event, handler, groupMatcher, warn) {
  if (handler === null || typeof handler !== 'object' || Array.isArray(handler)) {
    warn(`dsh-loomux-hooks: ${file} rejected: hooks.${event} handler is not an object`)
    return REJECT
  }
  if (handler.type !== 'command') {
    warn(`dsh-loomux-hooks: ${file}: hooks.${event} dropped handler of type ${JSON.stringify(handler.type)}; only "command" runs`)
    return null
  }
  const matcher = groupMatcher !== undefined ? groupMatcher : handler.matcher
  return toEntry(file, event, matcher, handler.command, handler.timeout, warn)
}

/**
 * Build the normalized entry, validating the fields one hook run needs: the
 * matcher must compile as a regular expression — a broken one rejects the
 * whole file by name — the command must be a string, and a timeout must be a
 * non-negative number of SECONDS, stored as `timeoutMs`.
 *
 * @param {string} file
 * @param {string} event
 * @param {unknown} matcher
 * @param {unknown} command
 * @param {unknown} timeout
 * @param {(warning: string) => void} warn
 * @returns {object|typeof REJECT}
 */
function toEntry(file, event, matcher, command, timeout, warn) {
  if (matcher !== undefined && typeof matcher !== 'string') {
    warn(`dsh-loomux-hooks: ${file} rejected: hooks.${event} matcher must be a string: ${JSON.stringify(matcher)}`)
    return REJECT
  }
  const pattern = matcher === undefined ? DEFAULT_MATCHER : matcher
  try {
    new RegExp(pattern)
  } catch (error) {
    warn(`dsh-loomux-hooks: ${file} rejected: hooks.${event} has an invalid matcher ${JSON.stringify(pattern)}: ${error.message}`)
    return REJECT
  }
  if (typeof command !== 'string' || command === '') {
    warn(`dsh-loomux-hooks: ${file} rejected: hooks.${event} handler command must be a non-empty string`)
    return REJECT
  }
  let timeoutMs
  if (timeout !== undefined) {
    if (typeof timeout !== 'number' || !Number.isFinite(timeout) || timeout < 0) {
      warn(`dsh-loomux-hooks: ${file} rejected: hooks.${event} handler timeout must be a non-negative number of seconds: ${JSON.stringify(timeout)}`)
      return REJECT
    }
    timeoutMs = timeout * 1000
  }
  return { event, matcher: pattern, command, timeoutMs }
}
