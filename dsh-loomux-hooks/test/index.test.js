/**
 * Tests for the plugin entry: identity, the wiring factory and the seam
 * registration. The host is faked — a ctx that records registrations, an
 * agent that records inject/steer/append, and a scripted runHook.
 */

import assert from 'node:assert/strict'
import test from 'node:test'
import path from 'node:path'

import * as plugin from '../index.js'
import { apply } from '../index.js'
import { runHook } from '../src/spawn.js'

test('index exposes the plugin identity and re-exports the runner', () => {
  assert.equal(plugin.name, 'dsh-loomux-hooks')
  assert.deepEqual(plugin.inject, [])
  assert.equal(plugin.runHook, runHook)
})

/** One hook entry as config.js would emit it: the timeoutMs key always exists. */
function entry(event, over = {}) {
  return { event, matcher: '.*', command: `loomux hook ${event} --host dsh --root "C:/w"`, timeoutMs: undefined, ...over }
}

/** A runHook fake: records calls and replays scripted results in order. */
function fakeRunHook(results = []) {
  const calls = []
  const run = async (command, payload, opts) => {
    calls.push({ command, payload, opts })
    return results.shift() ?? { exitCode: 0, stdout: '', stderr: '', timedOut: false }
  }
  return { run, calls }
}

/** An agent fake with the shape the host exposes: session, append, inject, steer. */
function fakeAgent(over = {}) {
  const agent = {
    id: 'a1',
    session: {
      id: 's1',
      header: { cwd: 'C:/w' },
      appends: [],
      append(type, data) {
        this.appends.push({ type, data })
      },
    },
    injected: [],
    inject(message) {
      this.injected.push(message)
    },
    steered: [],
    steer(message) {
      this.steered.push(message)
    },
    ...over,
  }
  if (over.session) agent.session = Object.assign({ appends: [], append(type, data) { this.appends.push({ type, data }) } }, over.session)
  return agent
}

/**
 * Wire apply() against a fake ctx. The resolver is injected so the test
 * controls the hook entries without touching the filesystem.
 */
function wire({ entries, config = {}, results, agents = new Map(), resolve, get } = {}) {
  const registrations = []
  const cleanups = []
  const disposed = []
  const infoLog = []
  const warnLog = []
  const fake = fakeRunHook(results)
  const ctx = {
    registrations,
    cleanups,
    disposed,
    infoLog,
    warnLog,
    agents,
    logger: {
      info: (type, data) => infoLog.push({ type, data }),
      warn: (message) => warnLog.push(message),
      debug: () => {},
    },
    on(event, handler, opts) {
      registrations.push({ event, handler, opts })
      return () => disposed.push(event)
    },
    effect(fn) {
      cleanups.push(fn())
    },
  }
  if (get) ctx.get = get
  apply(ctx, config, {
    runHook: fake.run,
    resolveConfig: resolve ?? (entries === undefined ? undefined : () => (entries === null ? null : { entries, sourcePath: 'C:/w/.loomux/dsh/hooks.json' })),
  })
  const handlerFor = (event) => registrations.find(r => r.event === event)?.handler
  return { ctx, registrations, handlerFor, calls: fake.calls, infoLog, warnLog }
}

const SEVEN_EVENTS = [
  'agent/created',
  'tools/pre-execute',
  'tools/post-execute',
  'agent/turn-stopping',
  'subagent/start',
  'subagent/end',
  'agent/disposed',
]

test('without a hook file the seams register but never run a hook', async () => {
  const { registrations, handlerFor, calls } = wire({ entries: null })
  assert.deepEqual(registrations.map(r => r.event), SEVEN_EVENTS)
  await handlerFor('agent/created')({ agent: fakeAgent() })
  const outcome = await handlerFor('tools/pre-execute')({ agent: fakeAgent(), name: 'write', arguments: {} }, async () => 'PASSED')
  assert.equal(outcome, 'PASSED')
  assert.equal(calls.length, 0)
})

test('an empty entry list stays silent the same way', async () => {
  const { handlerFor, calls } = wire({ entries: [] })
  await handlerFor('agent/created')({ agent: fakeAgent() })
  const outcome = await handlerFor('tools/pre-execute')({ agent: fakeAgent(), name: 'write', arguments: {} }, async () => 'PASSED')
  assert.equal(outcome, 'PASSED')
  assert.equal(calls.length, 0)
})

test('the chain resolves per session directory, never per process directory', async () => {
  const seen = []
  const { handlerFor } = wire({
    resolve: (_explicit, cwd) => {
      seen.push(cwd)
      return { entries: [entry('PreToolUse')], sourcePath: 'p' }
    },
  })
  await handlerFor('agent/created')({ agent: fakeAgent() })
  await handlerFor('tools/pre-execute')({ agent: fakeAgent(), name: 'write', arguments: {} }, async () => undefined)
  assert.deepEqual(seen, ['C:/w'])
  assert.ok(!seen.includes(process.cwd()))
})

test('a host whose agents service throws does not break the subagent seam', async () => {
  const { handlerFor, calls } = wire({
    entries: [entry('SubagentStart')],
    agents: null,
    get: () => { throw new Error('agents service not ready') },
  })
  await handlerFor('subagent/start')({ id: 'x', provider: 'p' })
  assert.equal(calls.length, 1)
})

test('a hook file registers all six seams plus disposal', () => {
  const { registrations } = wire({ entries: [entry('PreToolUse')] })
  assert.deepEqual(registrations.map(r => r.event), [
    'agent/created',
    'tools/pre-execute',
    'tools/post-execute',
    'agent/turn-stopping',
    'subagent/start',
    'subagent/end',
    'agent/disposed',
  ])
  assert.deepEqual(registrations[0].opts, { global: true })
})

test('resolveConfig warnings reach the logger prefixed when a seam resolves', async () => {
  const registrations = []
  const warnLog = []
  const ctx = {
    registrations,
    logger: { info: () => {}, warn: (m) => warnLog.push(m), debug: () => {} },
    on(event, handler) {
      registrations.push({ event, handler })
      return () => {}
    },
    effect(fn) {
      fn()
    },
  }
  apply(ctx, {}, {
    runHook: fakeRunHook().run,
    resolveConfig: (_explicit, _cwd, warn) => {
      warn('broken file')
      return { entries: [entry('PreToolUse')], sourcePath: 'p' }
    },
  })
  assert.deepEqual(warnLog, [])
  await registrations.find(r => r.event === 'tools/pre-execute').handler(
    { agent: fakeAgent(), name: 'write', arguments: {} }, async () => undefined,
  )
  assert.deepEqual(warnLog, ['[dsh-loomux-hooks] broken file'])
})

test('SessionStart runs awaited and injects each context verbatim', async () => {
  const context = JSON.stringify({ hookSpecificOutput: { hookEventName: 'SessionStart', additionalContext: 'context text' } })
  const { handlerFor, calls } = wire({ entries: [entry('SessionStart')], results: [{ exitCode: 0, stdout: context, stderr: '', timedOut: false }] })
  const agent = fakeAgent()
  await handlerFor('agent/created')({ agent })
  assert.equal(calls.length, 1)
  assert.deepEqual(agent.injected, [{ content: [{ type: 'text', text: 'context text' }], source: { kind: 'user' } }])
})

test('the SessionStart payload carries the ruled base fields', async () => {
  const { handlerFor, calls } = wire({ entries: [entry('SessionStart')] })
  const agent = fakeAgent()
  await handlerFor('agent/created')({ agent })
  assert.deepEqual(calls[0].payload, {
    session_id: 's1',
    transcript_path: '',
    hook_event_name: 'SessionStart',
    cwd: 'C:/w',
  })
})

test('PreToolUse denies with the stderr reason and falls back when blank', async () => {
  const exec = { agent: fakeAgent(), name: 'write', arguments: { file_path: 'a' }, callId: 'c1', turn: 3 }
  const denyWire = wire({ entries: [entry('PreToolUse')], results: [{ exitCode: 2, stdout: '', stderr: 'nope', timedOut: false }] })
  const denied = await denyWire.handlerFor('tools/pre-execute')(exec, async () => ({ kind: 'allow' }))
  assert.deepEqual(denied, { kind: 'deny', reason: 'nope' })
  assert.deepEqual(denyWire.calls[0].payload, {
    session_id: 's1',
    transcript_path: '',
    hook_event_name: 'PreToolUse',
    cwd: 'C:/w',
    tool_name: 'write',
    tool_input: { file_path: 'a' },
    tool_use_id: 'c1',
  })
  const blankWire = wire({ entries: [entry('PreToolUse')], results: [{ exitCode: 2, stdout: '', stderr: '', timedOut: false }] })
  const blank = await blankWire.handlerFor('tools/pre-execute')(exec, async () => ({ kind: 'allow' }))
  assert.deepEqual(blank, { kind: 'deny', reason: 'blocked by loomux hook' })
  // The missing callId degrades to the empty string.
  const noCall = wire({ entries: [entry('PreToolUse')] })
  await noCall.handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => undefined)
  assert.equal(noCall.calls[0].payload.tool_use_id, '')
})

test('PreToolUse passes through unless the hook owns the decision', async () => {
  const { handlerFor } = wire({ entries: [entry('PreToolUse')] })
  const nextResult = { kind: 'allow' }
  const out = await handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => nextResult)
  assert.equal(out, nextResult)
})

test('PreToolUse ask returns the ask decision', async () => {
  const ask = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'ask' } })
  const { handlerFor } = wire({ entries: [entry('PreToolUse')], results: [{ exitCode: 0, stdout: ask, stderr: '', timedOut: false }] })
  const out = await handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => ({ kind: 'allow' }))
  assert.deepEqual(out, { kind: 'ask' })
})

test('PostToolUse blocks with the ruled feedback shape', async () => {
  const { handlerFor, calls } = wire({ entries: [entry('PostToolUse')], results: [{ exitCode: 2, stdout: '', stderr: 'bad output', timedOut: false }] })
  const exec = { name: 'write', arguments: {}, agent: fakeAgent() }
  const out = await handlerFor('tools/post-execute')(exec, { content: [{ type: 'text', text: 'done' }] }, async () => ({ kind: 'allow' }))
  assert.deepEqual(out, { kind: 'block', feedback: [{ type: 'text', text: 'bad output' }] })
  assert.equal(calls[0].payload.tool_response, 'done')
})

test('PostToolUse tool_response joins text blocks and survives exotic results', async () => {
  const wireOne = wire({ entries: [entry('PostToolUse')] })
  await wireOne.handlerFor('tools/post-execute')({ name: 'write', arguments: {} }, { content: [{ type: 'text', text: 'one' }, { type: 'image' }, { type: 'text', text: 'two' }] }, async () => undefined)
  assert.equal(wireOne.calls[0].payload.tool_response, 'one\ntwo')
  const wireTwo = wire({ entries: [entry('PostToolUse')] })
  await wireTwo.handlerFor('tools/post-execute')({ name: 'write', arguments: {} }, { plain: true }, async () => undefined)
  assert.equal(wireTwo.calls[0].payload.tool_response, JSON.stringify({ plain: true }))
  const circular = {}
  circular.self = circular
  const wireThree = wire({ entries: [entry('PostToolUse')] })
  await wireThree.handlerFor('tools/post-execute')({ name: 'write', arguments: {} }, circular, async () => undefined)
  assert.equal(typeof wireThree.calls[0].payload.tool_response, 'string')
})

test('PostToolUse context injects after the call result and returns it', async () => {
  const context = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PostToolUse', additionalContext: 'extra' } })
  const { handlerFor } = wire({ entries: [entry('PostToolUse')], results: [{ exitCode: 0, stdout: context, stderr: '', timedOut: false }] })
  const agent = fakeAgent()
  const order = []
  agent.inject = (m) => {
    order.push('inject')
    agent.injected.push(m)
  }
  const out = await handlerFor('tools/post-execute')({ name: 'write', arguments: {}, agent }, { content: [] }, async () => {
    order.push('next')
    return { kind: 'allow' }
  })
  assert.deepEqual(order, ['next', 'inject'])
  assert.deepEqual(out, { kind: 'allow' })
  assert.equal(agent.injected[0].content[0].text, 'extra')
})

test('PostToolUse ask lets the call pass', async () => {
  const ask = JSON.stringify({ hookSpecificOutput: { hookEventName: 'PostToolUse', permissionDecision: 'ask' } })
  const { handlerFor } = wire({ entries: [entry('PostToolUse')], results: [{ exitCode: 0, stdout: ask, stderr: '', timedOut: false }] })
  const out = await handlerFor('tools/post-execute')({ name: 'write', arguments: {} }, { content: [] }, async () => ({ kind: 'allow' }))
  assert.deepEqual(out, { kind: 'allow' })
})

test('Stop steers with the reason, flips the flag and stops after eight', async () => {
  const entries = [entry('Stop')]
  const steerResults = Array.from({ length: 12 }, () => ({ exitCode: 2, stdout: '', stderr: 'keep going', timedOut: false }))
  const { handlerFor, calls } = wire({ entries, results: steerResults })
  const stop = handlerFor('agent/turn-stopping')
  const agent = fakeAgent()
  for (let i = 0; i < 9; i++) await stop({ agent, turn: i })
  assert.equal(calls.length, 9)
  assert.equal(calls[0].payload.stop_hook_active, false)
  assert.equal(calls[1].payload.stop_hook_active, true)
  assert.equal(agent.steered.length, 8)
  assert.deepEqual(agent.steered[0], { content: [{ type: 'text', text: 'keep going' }], source: { kind: 'user' } })
})

test('Stop falls back to a default reason when stderr is blank', async () => {
  const { handlerFor } = wire({ entries: [entry('Stop')], results: [{ exitCode: 2, stdout: '', stderr: '', timedOut: false }] })
  const agent = fakeAgent()
  await handlerFor('agent/turn-stopping')({ agent, turn: 1 })
  assert.equal(agent.steered[0].content[0].text, 'loomux stop hook')
})

test('agent disposal resets the steer counter', async () => {
  const steerResult = { exitCode: 2, stdout: '', stderr: 'go', timedOut: false }
  const { handlerFor, calls } = wire({ entries: [entry('Stop')], results: Array.from({ length: 10 }, () => steerResult) })
  const agent = fakeAgent()
  const stop = handlerFor('agent/turn-stopping')
  for (let i = 0; i < 8; i++) await stop({ agent, turn: i })
  await handlerFor('agent/disposed')({ agent })
  await stop({ agent, turn: 9 })
  assert.equal(calls.length, 9)
  assert.equal(calls[8].payload.stop_hook_active, false)
  assert.equal(agent.steered.length, 9)
})

test('subagent/start injects into the child and defaults the agent type', async () => {
  const context = JSON.stringify({ hookSpecificOutput: { hookEventName: 'SubagentStart', additionalContext: 'child context' } })
  const child = fakeAgent({ id: 'c1' })
  const agents = new Map([['c1', child]])
  const { handlerFor, calls } = wire({
    entries: [entry('SubagentStart')],
    results: [{ exitCode: 0, stdout: context, stderr: '', timedOut: false }],
    agents,
  })
  await handlerFor('subagent/start')({ id: 'c1', runId: 'r1' })
  assert.equal(calls[0].payload.agent_id, 'c1')
  assert.equal(calls[0].payload.agent_type, 'general-purpose')
  assert.equal(child.injected[0].content[0].text, 'child context')
})

test('subagent/start without a provider and without a resolvable child never throws', async () => {
  const { handlerFor, calls } = wire({ entries: [entry('SubagentStart')] })
  await handlerFor('subagent/start')({ id: 'gone', provider: 'xa' })
  assert.equal(calls[0].payload.agent_type, 'xa')
})

test('subagent/end runs SubagentStop hooks and decides nothing', async () => {
  const context = JSON.stringify({ hookSpecificOutput: { hookEventName: 'SubagentStop', additionalContext: 'ignored' } })
  const child = fakeAgent({ id: 'c1' })
  const agents = new Map([['c1', child]])
  const { handlerFor, calls } = wire({
    entries: [entry('SubagentStop')],
    results: [{ exitCode: 2, stdout: context, stderr: 'x', timedOut: false }],
    agents,
  })
  await handlerFor('subagent/end')({ id: 'c1', provider: 'local', stopReason: 'end' })
  assert.equal(calls[0].payload.hook_event_name, 'SubagentStop')
  assert.equal(calls[0].payload.stop_hook_active, false)
  assert.deepEqual(child.injected, [])
})

test('entries for other events never run at this seam', async () => {
  const { handlerFor, calls } = wire({ entries: [entry('SessionStart'), entry('PreToolUse')] })
  await handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => undefined)
  assert.equal(calls.length, 1)
  assert.match(calls[0].command, /PreToolUse/)
})

test('the matcher filters by subject and empty subjects match only dot-star', async () => {
  const { handlerFor, calls } = wire({ entries: [entry('PreToolUse', { matcher: 'write' }), entry('SessionStart', { matcher: 'write' })] })
  await handlerFor('tools/pre-execute')({ name: 'pwsh', arguments: {} }, async () => 'skipped')
  assert.equal(calls.length, 0)
  const agent = fakeAgent()
  await handlerFor('agent/created')({ agent })
  assert.equal(calls.length, 0, 'subject "" does not match the write matcher')
})

test('the timeout prefers the entry, then the config, then sixty seconds', async () => {
  const entries = [entry('PreToolUse', { timeoutMs: 5 }), entry('PreToolUse', { command: 'second' })]
  const { handlerFor, calls } = wire({ entries, config: { defaultTimeoutMs: 9 } })
  await handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => undefined)
  assert.equal(calls[0].opts.timeoutMs, 5)
  assert.equal(calls[1].opts.timeoutMs, 9)
  const dflt = wire({ entries: [entry('PreToolUse')] })
  await dflt.handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => undefined)
  assert.equal(dflt.calls[0].opts.timeoutMs, 60_000)
})

test('the cwd falls back to the process directory when the header is broken', async () => {
  const { handlerFor, calls } = wire({ entries: [entry('SessionStart')] })
  // Built by hand: the getter must survive until the plugin reads it.
  const agent = {
    id: 'a9',
    session: { id: 's2', get header() { throw new Error('boom') }, appends: [], append() {} },
    injected: [],
    inject(m) { this.injected.push(m) },
    steered: [],
    steer(m) { this.steered.push(m) },
  }
  await handlerFor('agent/created')({ agent })
  assert.equal(calls[0].payload.cwd, process.cwd())
})

test('every run appends hook/invoked and hook/result to the session log', async () => {
  const { handlerFor } = wire({ entries: [entry('PreToolUse', { matcher: 'w.*' })], results: [{ exitCode: 2, stdout: '', stderr: 'x'.repeat(600), timedOut: false }] })
  const agent = fakeAgent()
  await handlerFor('tools/pre-execute')({ agent, name: 'write', arguments: {}, turn: 7 }, async () => undefined)
  const [invoked, result] = agent.session.appends
  assert.equal(invoked.type, 'hook/invoked')
  assert.deepEqual(invoked.data, { turn: 7, point: 'tools/pre-execute', dialect: 'claude', handlerId: 'loomux:PreToolUse:0', matcher: 'w.*' })
  assert.equal(result.type, 'hook/result')
  assert.equal(result.data.turn, 7)
  assert.equal(result.data.point, 'tools/pre-execute')
  assert.equal(result.data.handlerId, 'loomux:PreToolUse:0')
  assert.equal(result.data.decision, 'deny')
  assert.equal(typeof result.data.durationMs, 'number')
  assert.equal(result.data.exitCode, 2)
  assert.equal(result.data.stderrSummary.length, 500)
})

test('a session append that throws loses the record but never breaks the hook', async () => {
  const { handlerFor, infoLog } = wire({ entries: [entry('PreToolUse')] })
  const agent = fakeAgent({ session: { id: 's1', header: { cwd: 'C:/w' }, appends: [], append() { throw new Error('closed') } } })
  const out = await handlerFor('tools/pre-execute')({ agent, name: 'write', arguments: {} }, async () => 'survived')
  assert.equal(out, 'survived')
  assert.deepEqual(infoLog, [])
})

test('the audit routes to the logger when no agent holds the session', async () => {
  const { handlerFor, infoLog } = wire({ entries: [entry('PreToolUse')] })
  await handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => undefined)
  assert.deepEqual(infoLog.map(l => l.type), ['hook/invoked', 'hook/result'])
})

test('a missing effect or missing logger never breaks apply', () => {
  const registrations = []
  apply({ on: (event, handler) => { registrations.push(event); return () => {} } }, {}, {
    runHook: fakeRunHook().run,
    resolveConfig: () => ({ entries: [entry('PreToolUse')], sourcePath: 'p' }),
  })
  assert.equal(registrations.length, 7)
})

test('disposal errors are swallowed and every disposer runs', () => {
  const disposed = []
  const ctx = {
    logger: { info: () => {}, warn: () => {}, debug: () => {} },
    on(event) {
      return () => {
        disposed.push(event)
        if (event === 'agent/created') throw new Error('gone')
      }
    },
    effect(fn) {
      fn()()
    },
  }
  apply(ctx, {}, {
    runHook: fakeRunHook().run,
    resolveConfig: () => ({ entries: [entry('PreToolUse')], sourcePath: 'p' }),
  })
  assert.equal(disposed.length, 7)
})

test('all entries of a seam run and the first decisive answer wins', async () => {
  const { handlerFor, calls } = wire({
    entries: [entry('PreToolUse', { command: 'one' }), entry('PreToolUse', { command: 'two' })],
    results: [
      { exitCode: 2, stdout: '', stderr: 'first', timedOut: false },
      { exitCode: 2, stdout: '', stderr: 'second', timedOut: false },
    ],
  })
  const out = await handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => 'nexted')
  assert.equal(calls.length, 2)
  assert.deepEqual(out, { kind: 'deny', reason: 'first' })
})

test('contexts accumulate across entries', async () => {
  const ctxRun = (t) => ({
    exitCode: 0,
    stdout: JSON.stringify({ hookSpecificOutput: { hookEventName: 'SessionStart', additionalContext: t } }),
    stderr: '',
    timedOut: false,
  })
  const { handlerFor } = wire({
    entries: [entry('SessionStart', { command: 'one' }), entry('SessionStart', { command: 'two' })],
    results: [ctxRun('one ctx'), ctxRun('two ctx')],
  })
  const agent = fakeAgent()
  await handlerFor('agent/created')({ agent })
  assert.deepEqual(agent.injected.map(m => m.content[0].text), ['one ctx', 'two ctx'])
})

const CLAUDE_FORM = '"${LOCALAPPDATA}/loomux/dev/$(basename "${CLAUDE_PROJECT_DIR}")/loomux.exe" hook pre-tool-use --host dsh --root "${CLAUDE_PROJECT_DIR}"'

test('the bash placeholders of the hook file reach the runner resolved', async () => {
  const saved = { la: process.env.LOCALAPPDATA, cpd: process.env.CLAUDE_PROJECT_DIR }
  process.env.LOCALAPPDATA = 'C:\\LA'
  try {
    const { handlerFor, calls } = wire({ entries: [entry('PreToolUse', { command: CLAUDE_FORM })] })
    await handlerFor('tools/pre-execute')({ agent: fakeAgent(), name: 'write', arguments: {} }, async () => undefined)
    assert.equal(
      calls[0].command,
      `"C:\\LA/loomux/dev/${path.basename('C:/w')}/loomux.exe" hook pre-tool-use --host dsh --root "C:/w"`,
    )
  } finally {
    if (saved.la === undefined) delete process.env.LOCALAPPDATA
    else process.env.LOCALAPPDATA = saved.la
    if (saved.cpd !== undefined) process.env.CLAUDE_PROJECT_DIR = saved.cpd
  }
})

test('a command without placeholders and without LOCALAPPDATA stays verbatim', async () => {
  const saved = process.env.LOCALAPPDATA
  delete process.env.LOCALAPPDATA
  try {
    const { handlerFor, calls } = wire({ entries: [entry('PreToolUse', { command: '${LOCALAPPDATA}/plain.exe --root "${CLAUDE_PROJECT_DIR}"' })] })
    await handlerFor('tools/pre-execute')({ agent: fakeAgent(), name: 'write', arguments: {} }, async () => undefined)
    assert.equal(calls[0].command, '${LOCALAPPDATA}/plain.exe --root "C:/w"')
    const bare = wire({ entries: [entry('PreToolUse', { command: 'loomux hook stop' })] })
    await bare.handlerFor('tools/pre-execute')({ name: 'write', arguments: {} }, async () => undefined)
    assert.equal(bare.calls[0].command, 'loomux hook stop')
  } finally {
    if (saved !== undefined) process.env.LOCALAPPDATA = saved
  }
})
