/**
 * Map the grep tool arguments to `loomux graph grep` arguments.
 *
 * The list is built here, never by a shell: an `include` with spaces stays one
 * argument. `--channel cloud` is appended unconditionally and no input field
 * reaches it, so a caller cannot ask for the local channel.
 */

export const DEFAULT_MAX_RESULTS = 300
export const MIN_MAX_RESULTS = 20
export const MAX_MAX_RESULTS = 5000

/**
 * Clamp max_results the way the tool schema promises: 20 to 5000, 300 when the
 * value is absent, zero or not a number.
 */
export function clampMaxResults(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return DEFAULT_MAX_RESULTS
  if (number === 0) return DEFAULT_MAX_RESULTS
  return Math.min(MAX_MAX_RESULTS, Math.max(MIN_MAX_RESULTS, Math.trunc(number)))
}

/**
 * Reject an `include` that is not one positive glob filter.
 */
export function validateInclude(include) {
  if (typeof include !== 'string' || include.trim().length === 0) {
    throw new Error('include must be a non-empty string when given')
  }
  if (include.startsWith('!')) {
    throw new Error('include must be a positive glob filter; negated patterns ("!…") are not supported')
  }
  let braceDepth = 0
  for (const char of include) {
    if (char === '{') braceDepth++
    else if (char === '}') braceDepth = Math.max(0, braceDepth - 1)
    else if (char === ',' && braceDepth === 0) {
      throw new Error('include must be one glob, not a comma-separated list (use {a,b} alternation instead)')
    }
  }
}

/**
 * Build the argument list for one search.
 *
 * @param {Record<string, unknown>} input
 * @returns {string[]}
 */
export function buildArgs(input) {
  const args = ['graph', 'grep']
  if (!input || typeof input !== 'object') {
    throw new Error('grep: arguments must be an object')
  }
  if (typeof input.pattern !== 'string' || input.pattern.length === 0) {
    throw new Error('pattern must be a non-empty string')
  }
  args.push(input.pattern)

  args.push('--json')
  args.push('--channel', 'cloud')

  if (input.case_insensitive !== undefined && input.case_insensitive !== false) {
    args.push('-i')
  }
  if (input.include !== undefined) {
    validateInclude(input.include)
    args.push('--glob', String(input.include))
  }
  if (input.path !== undefined) {
    if (typeof input.path !== 'string' || input.path.trim().length === 0) {
      throw new Error('path must be a non-empty string when given')
    }
    args.push('--in', input.path)
  }
  args.push('--max-hits', String(clampMaxResults(input.max_results)))
  return args
}
