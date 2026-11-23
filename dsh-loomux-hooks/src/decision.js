/**
 * The decision machine: one hook run's outcome — the seam it ran at, its exit
 * code, and the stdout and stderr the runner reported — becomes the normalized
 * decision the plugin acts on. Pure by contract: the decision hangs on
 * exactly those four fields and the seam name; no file, no environment, no
 * clock.
 *
 * The pinned semantics (a live spike against the real harness):
 *
 * - Exit 2 is the blocking failure. At PreToolUse it denies the tool call, at
 *   PostToolUse it blocks with the stderr text as feedback, at Stop it steers
 *   the agent onward. The trimmed stderr is the reason; a blank stderr leaves
 *   the reason undefined for the caller's fallback text. The seams that only
 *   receive context never block: an internal failure locks nothing.
 * - Exit 0 with a JSON object on stdout is the JSON dialect; any other stdout
 *   decides nothing, silently. Fields under hookSpecificOutput are event-scoped
 *   and count only while hookSpecificOutput.hookEventName names the seam's
 *   Claude event — a wrong name discards exactly those fields, never the
 *   top-level decision.
 * - hookSpecificOutput.permissionDecision: deny and ask decide the tool call
 *   at the two tool-gating seams (deny at PostToolUse takes the Post shape
 *   block); allow decides nothing and lets the fields beside it speak. Every
 *   other value is ignored.
 * - The top-level decision: block decides per seam (deny, block, steer) and
 *   its reason field carries the feedback text; approve releases and, like
 *   allow, does not suppress the context beside it. Every other value is
 *   ignored.
 * - hookSpecificOutput.additionalContext becomes context verbatim at
 *   SessionStart, PostToolUse and SubagentStart; the caller prepends it. It
 *   must be a non-empty string.
 * - SubagentStop decides nothing at all, and so does every seam at exit 1, on
 *   a timeout and on a spawn failure: an internal error never locks.
 */

/** The seam names the plugin registers, each mapped to the Claude hook event whose dialect it answers. */
export const SEAM_EVENTS = Object.freeze({
  sessionStart: 'SessionStart',
  preToolUse: 'PreToolUse',
  postToolUse: 'PostToolUse',
  stop: 'Stop',
  subagentStart: 'SubagentStart',
  subagentStop: 'SubagentStop',
})

/** The kind a blocking failure (exit 2) or a top-level block takes per seam; the context seams have none. */
const NEGATIVE = { preToolUse: 'deny', postToolUse: 'block', stop: 'steer' }

/** The kind a permissionDecision deny takes: only the seams that gate a tool call. */
const PERMISSION_DENIED = { preToolUse: 'deny', postToolUse: 'block' }

/** The seams a permissionDecision ask may prompt the user at. */
const ASK_SEAMS = new Set(['preToolUse', 'postToolUse'])

/** The seams that consume hookSpecificOutput.additionalContext as context. */
const CONTEXT_SEAMS = new Set(['sessionStart', 'postToolUse', 'subagentStart'])

/**
 * Trim one feedback field; a missing or blank one leaves no reason behind.
 *
 * @param {unknown} value the raw stderr or reason field
 * @returns {string|undefined} the trimmed text, undefined when there is none
 */
function text(value) {
  return typeof value === 'string' && value.trim() !== '' ? value.trim() : undefined
}

/**
 * Parse the stdout of a clean run; everything that is not a JSON object —
 * empty output, plain text, a number, a list — decides nothing.
 *
 * @param {unknown} stdout the raw stdout
 * @returns {object|undefined} the parsed document, undefined when there is none
 */
function parse(stdout) {
  try {
    const doc = JSON.parse(stdout)
    return typeof doc === 'object' && doc !== null ? doc : undefined
  } catch {
    return undefined
  }
}

/**
 * Map one hook run's outcome onto a normalized decision.
 *
 * @param {string} seam one of the SEAM_EVENTS keys; anything else decides nothing
 * @param {{ exitCode: number|null, stdout: string, stderr: string, timedOut: boolean }} [result] the runner's result for the seam
 * @returns {{ kind: 'none'|'deny'|'ask'|'block'|'steer'|'context', reason?: string, context?: string }}
 */
export function decide(seam, result) {
  // A seam the machine does not know decides nothing; the own-property test
  // keeps Object.prototype members (valueOf, constructor) out of the lookup.
  // SubagentStop is known and its decisions are ignored by the protocol: it
  // never blocks.
  if (!Object.hasOwn(SEAM_EVENTS, seam) || seam === 'subagentStop') return { kind: 'none' }
  const event = SEAM_EVENTS[seam]
  const { exitCode, stdout, stderr, timedOut } = result ?? {}
  // A timeout, a spawn failure and every exit code but 0 and 2: an internal
  // failure never locks anything, at any seam.
  if (timedOut || exitCode === null || (exitCode !== 0 && exitCode !== 2)) return { kind: 'none' }
  if (exitCode === 2) {
    const kind = NEGATIVE[seam]
    // A context seam has no negative kind: exit 2 blocks nothing there.
    if (kind === undefined) return { kind: 'none' }
    const reason = text(stderr)
    return reason === undefined ? { kind } : { kind, reason }
  }
  const doc = parse(stdout)
  if (doc === undefined) return { kind: 'none' }
  // hookSpecificOutput speaks for its event only: a wrong hookEventName
  // discards the event-scoped fields while the top-level decision still counts.
  const scoped = doc.hookSpecificOutput
  const output = typeof scoped === 'object' && scoped !== null && scoped.hookEventName === event ? scoped : undefined
  const permission = output?.permissionDecision
  if (permission === 'deny') {
    const kind = PERMISSION_DENIED[seam]
    if (kind !== undefined) return { kind }
  }
  if (permission === 'ask' && ASK_SEAMS.has(seam)) return { kind: 'ask' }
  // allow and every other permissionDecision value decide nothing; the
  // top-level fields beside it are still read, and so is approve.
  if (doc.decision === 'block') {
    const kind = NEGATIVE[seam]
    if (kind !== undefined) {
      const reason = text(doc.reason)
      return reason === undefined ? { kind } : { kind, reason }
    }
  }
  if (CONTEXT_SEAMS.has(seam)) {
    const context = output?.additionalContext
    if (typeof context === 'string' && context.trim() !== '') return { kind: 'context', context }
  }
  return { kind: 'none' }
}
