/**
 * Tests for the decision machine. One group per row of the outcome table the
 * live spike against the real harness pinned down: the meaning of the exit
 * codes per seam, the JSON dialect's event-scoped and top-level fields, and
 * the rule that an internal failure never locks anything.
 */

import assert from 'node:assert/strict'
import test from 'node:test'

import { decide, SEAM_EVENTS } from '../src/decision.js'

const NONE = { kind: 'none' }
const ALL_SEAMS = ['sessionStart', 'preToolUse', 'postToolUse', 'stop', 'subagentStart', 'subagentStop']

/** The spawn.js result shape with its clean-run defaults. */
const run = over => ({ exitCode: 0, stdout: '', stderr: '', timedOut: false, ...over })

/** A payload that would deny, block, steer and context everywhere it counted. */
const loud = {
  decision: 'block',
  reason: 'loud reason',
  hookSpecificOutput: {
    hookEventName: 'PreToolUse',
    permissionDecision: 'deny',
    additionalContext: 'loud context',
  },
}

// --- the seam table ---

test('the seam table names the Claude event of every seam', () => {
  assert.deepEqual(SEAM_EVENTS, {
    sessionStart: 'SessionStart',
    preToolUse: 'PreToolUse',
    postToolUse: 'PostToolUse',
    stop: 'Stop',
    subagentStart: 'SubagentStart',
    subagentStop: 'SubagentStop',
  })
})

// --- PreToolUse: exit 2 ---

test('PreToolUse: exit 2 denies with the trimmed stderr as reason', () => {
  const result = run({ exitCode: 2, stderr: '  git push is banned  ' })
  assert.deepEqual(decide('preToolUse', result), { kind: 'deny', reason: 'git push is banned' })
})

test('PreToolUse: exit 2 on a blank stderr denies and leaves the reason to the caller', () => {
  assert.deepEqual(decide('preToolUse', run({ exitCode: 2, stderr: ' \n' })), { kind: 'deny' })
})

// --- PreToolUse: the JSON dialect ---

test('PreToolUse: permissionDecision deny denies', () => {
  const stdout = JSON.stringify({
    hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'deny' },
    continue: true,
  })
  assert.deepEqual(decide('preToolUse', run({ stdout })), { kind: 'deny' })
})

test('PreToolUse: permissionDecision ask asks', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'ask' } })
  assert.deepEqual(decide('preToolUse', run({ stdout })), { kind: 'ask' })
})

test('PreToolUse: permissionDecision allow decides nothing', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'allow' } })
  assert.deepEqual(decide('preToolUse', run({ stdout })), NONE)
})

test('PreToolUse: a permissionDecision of any other value is ignored', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'proceed' } })
  assert.deepEqual(decide('preToolUse', run({ stdout })), NONE)
})

test('PreToolUse: a wrong hookEventName discards the event-scoped permissionDecision', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PostToolUse', permissionDecision: 'deny' } })
  assert.deepEqual(decide('preToolUse', run({ stdout })), NONE)
})

test('PreToolUse: a wrong hookEventName still honors the top-level decision', () => {
  const stdout = JSON.stringify({
    decision: 'block',
    reason: 'forbidden',
    hookSpecificOutput: { hookEventName: 'SessionStart', permissionDecision: 'allow', additionalContext: 'ignored' },
  })
  assert.deepEqual(decide('preToolUse', run({ stdout })), { kind: 'deny', reason: 'forbidden' })
})

test('the event-scoped permissionDecision outranks the top-level decision', () => {
  const stdout = JSON.stringify({
    decision: 'approve',
    hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'deny' },
  })
  assert.deepEqual(decide('preToolUse', run({ stdout })), { kind: 'deny' })
})

// --- PostToolUse ---

test('PostToolUse: exit 2 blocks with the stderr text as feedback', () => {
  const result = run({ exitCode: 2, stderr: 'output looks wrong\n' })
  assert.deepEqual(decide('postToolUse', result), { kind: 'block', reason: 'output looks wrong' })
})

test('PostToolUse: permissionDecision deny takes the Post shape block', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PostToolUse', permissionDecision: 'deny' } })
  assert.deepEqual(decide('postToolUse', run({ stdout })), { kind: 'block' })
})

test('PostToolUse: permissionDecision ask asks', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PostToolUse', permissionDecision: 'ask' } })
  assert.deepEqual(decide('postToolUse', run({ stdout })), { kind: 'ask' })
})

test('PostToolUse: additionalContext becomes context verbatim', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PostToolUse', additionalContext: '  file changed  ' } })
  assert.deepEqual(decide('postToolUse', run({ stdout })), { kind: 'context', context: '  file changed  ' })
})

test('PostToolUse: a wrong hookEventName discards the additionalContext', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'SessionStart', additionalContext: 'x' } })
  assert.deepEqual(decide('postToolUse', run({ stdout })), NONE)
})

test('PostToolUse: a top-level block outranks additionalContext', () => {
  const stdout = JSON.stringify({
    decision: 'block',
    reason: 'redo it',
    hookSpecificOutput: { hookEventName: 'PostToolUse', additionalContext: 'later' },
  })
  assert.deepEqual(decide('postToolUse', run({ stdout })), { kind: 'block', reason: 'redo it' })
})

test('the top-level block takes the deny and block shapes at the Pre and Post seams', () => {
  assert.deepEqual(decide('preToolUse', run({ stdout: JSON.stringify({ decision: 'block', reason: 'nope' }) })), { kind: 'deny', reason: 'nope' })
  assert.deepEqual(decide('postToolUse', run({ stdout: JSON.stringify({ decision: 'block', reason: 'redo' }) })), { kind: 'block', reason: 'redo' })
})

// --- Stop ---

test('Stop: exit 2 steers with the stderr reason', () => {
  assert.deepEqual(decide('stop', run({ exitCode: 2, stderr: ' finish the tests ' })), { kind: 'steer', reason: 'finish the tests' })
})

test('Stop: a top-level decision block steers with its reason', () => {
  const stdout = JSON.stringify({ decision: 'block', reason: 'run the checks' })
  assert.deepEqual(decide('stop', run({ stdout })), { kind: 'steer', reason: 'run the checks' })
})

test('Stop: a top-level decision block without a reason steers', () => {
  assert.deepEqual(decide('stop', run({ stdout: JSON.stringify({ decision: 'block' }) })), { kind: 'steer' })
})

test('Stop: a top-level decision approve releases to none', () => {
  assert.deepEqual(decide('stop', run({ stdout: JSON.stringify({ decision: 'approve' }) })), NONE)
})

test('Stop: permissionDecision does not speak at the stop seam', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'Stop', permissionDecision: 'deny' } })
  assert.deepEqual(decide('stop', run({ stdout })), NONE)
})

test('additionalContext at a seam that does not consume it decides nothing', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'Stop', additionalContext: 'note' } })
  assert.deepEqual(decide('stop', run({ stdout })), NONE)
})

// --- SessionStart and SubagentStart ---

test('SessionStart: additionalContext under its own event name becomes context', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'SessionStart', additionalContext: 'repo rules loaded' } })
  assert.deepEqual(decide('sessionStart', run({ stdout })), { kind: 'context', context: 'repo rules loaded' })
})

test('SubagentStart: additionalContext under its own event name becomes context', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'SubagentStart', additionalContext: 'brief attached' } })
  assert.deepEqual(decide('subagentStart', run({ stdout })), { kind: 'context', context: 'brief attached' })
})

test('SessionStart: a wrong hookEventName lets the additionalContext fall away', () => {
  const stdout = JSON.stringify({ hookSpecificOutput: { hookEventName: 'Session', additionalContext: 'nope' } })
  assert.deepEqual(decide('sessionStart', run({ stdout })), NONE)
})

test('a top-level block is not a decision at the context seams', () => {
  const stdout = JSON.stringify({ decision: 'block', reason: 'r' })
  assert.deepEqual(decide('sessionStart', run({ stdout })), NONE)
  assert.deepEqual(decide('subagentStart', run({ stdout })), NONE)
})

test('additionalContext must be a non-empty string to become context', () => {
  const at = additionalContext => JSON.stringify({ hookSpecificOutput: { hookEventName: 'SessionStart', additionalContext } })
  assert.deepEqual(decide('sessionStart', run({ stdout: at('   ') })), NONE)
  assert.deepEqual(decide('sessionStart', run({ stdout: at(['a', 'b']) })), NONE)
  assert.deepEqual(decide('sessionStart', run({ stdout: at(7) })), NONE)
  assert.deepEqual(decide('sessionStart', run({ stdout: JSON.stringify({ hookSpecificOutput: { hookEventName: 'SessionStart' } }) })), NONE)
})

test('allow and approve do not suppress the additionalContext beside them', () => {
  const stdout1 = JSON.stringify({
    hookSpecificOutput: { hookEventName: 'PostToolUse', permissionDecision: 'allow', additionalContext: 'note' },
  })
  assert.deepEqual(decide('postToolUse', run({ stdout: stdout1 })), { kind: 'context', context: 'note' })
  const stdout2 = JSON.stringify({
    decision: 'approve',
    hookSpecificOutput: { hookEventName: 'PostToolUse', additionalContext: 'note' },
  })
  assert.deepEqual(decide('postToolUse', run({ stdout: stdout2 })), { kind: 'context', context: 'note' })
})

// --- failures that must never lock ---

test('exit 1, a timeout and a spawn failure yield none at every seam', () => {
  const stdout = JSON.stringify(loud)
  for (const seam of ALL_SEAMS) {
    assert.deepEqual(decide(seam, run({ exitCode: 1, stdout, stderr: 'boom' })), NONE, seam)
    assert.deepEqual(decide(seam, run({ exitCode: null, timedOut: true, stdout, stderr: 'boom' })), NONE, seam)
    assert.deepEqual(decide(seam, run({ exitCode: null, stdout, stderr: 'ENOENT' })), NONE, seam)
  }
})

test('an exit code other than 0 and 2 yields none at every seam', () => {
  const stdout = JSON.stringify(loud)
  for (const seam of ALL_SEAMS) {
    assert.deepEqual(decide(seam, run({ exitCode: 3, stdout, stderr: 'boom' })), NONE, seam)
  }
})

test('exit 2 never blocks at the seams that only receive context', () => {
  assert.deepEqual(decide('sessionStart', run({ exitCode: 2, stderr: 'boom' })), NONE)
  assert.deepEqual(decide('subagentStart', run({ exitCode: 2, stderr: 'boom' })), NONE)
})

test('a non-JSON stdout yields none silently at every seam', () => {
  for (const seam of ALL_SEAMS) {
    assert.deepEqual(decide(seam, run({ stdout: 'not json at all' })), NONE, seam)
  }
})

test('stdout that parses as JSON but is not an object yields none', () => {
  for (const stdout of ['', '   ', '5', 'null', 'true', '"text"', '[1,2]', '{"broken"']) {
    assert.deepEqual(decide('preToolUse', run({ stdout })), NONE, stdout)
  }
})

test('a top-level decision other than approve or block is ignored', () => {
  for (const decision of ['ask', 'deny', 'steer', '', 7, null]) {
    const stdout = JSON.stringify({ decision, reason: 'r' })
    assert.deepEqual(decide('stop', run({ stdout })), NONE, String(decision))
    assert.deepEqual(decide('preToolUse', run({ stdout })), NONE, String(decision))
  }
})

// --- seams the machine does not know and the seam that never decides ---

test('an unknown seam always yields none', () => {
  for (const seam of ['PreToolUse', 'pre_tool_use', 'Pre', 'valueOf', 'constructor', undefined, null, 42]) {
    assert.deepEqual(decide(seam, run({ exitCode: 2, stderr: 'boom' })), NONE, String(seam))
    assert.deepEqual(decide(seam, run({ stdout: JSON.stringify(loud) })), NONE, String(seam))
  }
})

test('SubagentStop is always none, whatever it emits', () => {
  assert.deepEqual(decide('subagentStop', run({ exitCode: 2, stderr: 'boom' })), NONE)
  assert.deepEqual(decide('subagentStop', run({ stdout: JSON.stringify({ decision: 'block', reason: 'r' }) })), NONE)
  const stdout = JSON.stringify({
    hookSpecificOutput: { hookEventName: 'SubagentStop', additionalContext: 'note', permissionDecision: 'deny' },
  })
  assert.deepEqual(decide('subagentStop', run({ stdout })), NONE)
})

test('an absent result decides nothing', () => {
  assert.deepEqual(decide('preToolUse', {}), NONE)
  assert.deepEqual(decide('preToolUse'), NONE)
})
