/**
 * Classify a finished `loomux graph grep` call.
 *
 * Exit 1 is a failure in loomux (missing graph, bad pattern, uncovered
 * prefix), never an empty result, unlike ripgrep. The wording comes from the
 * Go side; the tests pin it, and the code reads it once here instead of at
 * every call site.
 */

const CLASSES = [
  'no-binary',
  'no-graph',
  'invalid-regex',
  'unsupported',
  'prefix-not-indexed',
  'usage',
  'timeout',
  'stdout-cap',
  'unknown',
]

const HINTS = {
  'no-binary': 'loomux is not on PATH. Install it or point the plugin at the binary.',
  'no-graph': 'Build the code graph first: `loomux graph build`.',
  'invalid-regex': 'The pattern is not a valid RE2 regular expression.',
  unsupported: 'The pattern uses a construct RE2 does not support (lookaround or backreference).',
  'prefix-not-indexed': 'The requested prefix is not part of the search corpus.',
  usage: 'The plugin built an invalid argument list; this is a bug in dsh-loomux.',
  timeout: 'The search was cancelled or timed out before it finished.',
  'stdout-cap': 'The answer exceeded the output cap; the result would have been incomplete.',
  unknown: 'loomux graph grep failed; see the captured stderr for the reason.',
}

/**
 * @param {number|null} exit
 * @param {string} stderr
 * @param {{ aborted?: boolean, stdoutCap?: boolean, spawnError?: { code?: string }|null }} [info]
 * @returns {string} one of CLASSES, or 'ok'
 */
export function classify(exit, stderr, info = {}) {
  if (info.spawnError) {
    return info.spawnError.code === 'ENOENT' ? 'no-binary' : 'unknown'
  }
  if (info.stdoutCap) return 'stdout-cap'
  if (info.aborted) return 'timeout'
  if (exit === 0) return 'ok'

  const text = typeof stderr === 'string' ? stderr : ''
  if (text.includes('no graph. Run')) return 'no-graph'
  if (text.includes('invalid regex:')) return 'invalid-regex'
  if (text.includes('not supported in RE2 regex:')) return 'unsupported'
  if (text.includes('prefix not indexed:')) return 'prefix-not-indexed'
  if (exit === 2) return 'usage'
  return 'unknown'
}

/** Build the Error a classified failure throws, with its hint. */
export function errorFor(code, detail) {
  const hint = HINTS[code] ?? HINTS.unknown
  const suffix = detail ? `\n${detail.trim()}` : ''
  return new Error(`loomux graph grep failed (${code}): ${hint}${suffix}`)
}

/** Keep an exit-0 stderr note; loomux announces a rebuild there. */
export function noteFromStderr(stderr) {
  if (typeof stderr !== 'string') return ''
  return stderr.trim()
}

export { CLASSES, HINTS }
