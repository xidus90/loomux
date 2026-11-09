import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  apply,
  createGrepTool,
  name as pluginName,
  narrowSearchFiles,
  parseAnswer,
  resolveCwd,
} from '../index.js'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const symbolGroup = JSON.parse(readFileSync(new URL('./fixtures/grep-symbol-group.json', import.meta.url), 'utf8'))

function toolWith(run, config = {}) {
  return createGrepTool({ binary: 'loomux', run, ...config })
}

test('the plugin exports its name and injects tools', () => {
  assert.equal(pluginName, 'dsh-loomux')
})

test('config.maxResults is the --max-hits of a call without max_results', async () => {
  let seen = null
  const tool = toolWith(async args => {
    seen = args
    return { exit: 0, stdout: JSON.stringify(symbolGroup), stderr: '', spawnError: null, stdoutCap: false, aborted: false }
  }, { maxResults: 100 })
  await tool.execute({ pattern: 'x' }, { cwd: root })
  assert.deepEqual(seen.slice(-2), ['--max-hits', '100'])
  await tool.execute({ pattern: 'x', max_results: 50 }, { cwd: root })
  assert.deepEqual(seen.slice(-2), ['--max-hits', '50'])
})

test('the tool carries the documented schema and timeout', () => {
  const tool = toolWith(async () => ({ exit: 0, stdout: '{}', stderr: '' }))
  assert.equal(tool.name, 'grep')
  assert.equal(tool.timeoutMs, 30_000)
  assert.deepEqual(tool.parameters.required, ['pattern'])
  assert.deepEqual(Object.keys(tool.parameters.properties), [
    'pattern',
    'path',
    'include',
    'case_insensitive',
    'max_results',
  ])
  assert.match(tool.description, /loomux graph grep/)
})

test('execute maps the call and returns matches with a note', async () => {
  let seen = null
  const tool = toolWith(async (args, options) => {
    seen = { args, options }
    return {
      exit: 0,
      stdout: JSON.stringify(symbolGroup),
      stderr: 'loomux graph grep: 9 files moved, rebuilding the graph\n',
      spawnError: null,
      stdoutCap: false,
      aborted: false,
    }
  })
  const exec = { agent: { session: { header: { cwd: root } } } }
  const value = await tool.execute({ pattern: 'TruncatedHits > 0', include: '*.go' }, exec)
  assert.deepEqual(seen.args, [
    'graph',
    'grep',
    '--',
    'TruncatedHits > 0',
    '--json',
    '--channel',
    'cloud',
    '--glob',
    '*.go',
    '--max-hits',
    '300',
  ])
  assert.equal(seen.options.cwd, root)
  assert.equal(value.matches.length, 1)
  assert.equal(value.matches[0].symbol, 'GrepReport')
  assert.match(value.notes[0], /rebuilding/)
  assert.equal(seen.options.signal, undefined)
})

test('execute passes the abort signal through', async () => {
  let signal = null
  const tool = toolWith(async (_args, options) => {
    signal = options.signal
    return { exit: 0, stdout: '{"groups":null}', stderr: '', spawnError: null, stdoutCap: false, aborted: false }
  })
  const controller = new AbortController()
  await tool.execute({ pattern: 'x' }, { cwd: root, signal: controller.signal })
  assert.equal(signal, controller.signal)
})

test('execute rejects a path outside the workspace', async () => {
  const tool = toolWith(async () => {
    throw new Error('must not run')
  })
  await assert.rejects(tool.execute({ pattern: 'x', path: '..' }, { cwd: root }), /outside the workspace root/)
})

// A model often names the workspace root itself, absolute or as ".": loomux
// knows no such prefix, so the call must search the whole corpus instead.
test('execute searches everything when the path is the workspace root', async () => {
  for (const path of [root, '.', `${root}/`]) {
    let seen = null
    const tool = toolWith(async args => {
      seen = args
      return { exit: 0, stdout: JSON.stringify(symbolGroup), stderr: '', spawnError: null, stdoutCap: false, aborted: false }
    })
    await tool.execute({ pattern: 'x', path }, { cwd: root })
    assert.equal(seen.includes('--in'), false, `path ${path}: ${seen.join(' ')}`)
  }
})

test('execute passes a path inside the workspace relative to its root', async () => {
  let seen = null
  const tool = toolWith(async args => {
    seen = args
    return { exit: 0, stdout: JSON.stringify(symbolGroup), stderr: '', spawnError: null, stdoutCap: false, aborted: false }
  })
  await tool.execute({ pattern: 'x', path: resolve(root, 'src') }, { cwd: root })
  assert.deepEqual(seen.slice(seen.indexOf('--in'), seen.indexOf('--in') + 2), ['--in', 'src'])
})

test('execute turns every failing exit into an error, never an empty list', async () => {
  for (const [stderr, pattern] of [
    ['loomux graph grep: no graph. Run `loomux graph build` first.', /no-graph/],
    ['loomux graph grep: invalid regex: bad', /invalid-regex/],
    ['loomux graph grep: lookaround not supported in RE2 regex: (?=x)', /unsupported/],
    ['loomux graph grep: prefix not indexed: vendor', /prefix-not-indexed/],
  ]) {
    const tool = toolWith(async () => ({ exit: 1, stdout: '', stderr, spawnError: null, stdoutCap: false, aborted: false }))
    await assert.rejects(tool.execute({ pattern: 'x' }, { cwd: root }), pattern)
  }
})

test('execute fails on a missing binary and on the stdout cap', async () => {
  const missing = toolWith(async () => ({
    exit: null,
    stdout: '',
    stderr: '',
    spawnError: { code: 'ENOENT' },
    stdoutCap: false,
    aborted: false,
  }))
  await assert.rejects(missing.execute({ pattern: 'x' }, { cwd: root }), /no-binary/)
  const capped = toolWith(async () => ({ exit: null, stdout: '', stderr: '', spawnError: null, stdoutCap: true, aborted: false }))
  await assert.rejects(capped.execute({ pattern: 'x' }, { cwd: root }), /stdout-cap/)
})

test('execute fails when the answer is not the documented JSON', async () => {
  for (const stdout of ['', '[]', '{"groups":{}}', 'not json']) {
    const tool = toolWith(async () => ({ exit: 0, stdout, stderr: '', spawnError: null, stdoutCap: false, aborted: false }))
    await assert.rejects(tool.execute({ pattern: 'x' }, { cwd: root }))
  }
})

test('parseAnswer accepts groups null and a groups array', () => {
  assert.deepEqual(parseAnswer('{"groups":null}'), { groups: null })
  assert.equal(parseAnswer(JSON.stringify(symbolGroup)).total_hits, 1)
})

test('narrowSearchFiles rejects malformed replays and keeps valid ones', () => {
  assert.equal(narrowSearchFiles(undefined), undefined)
  assert.equal(narrowSearchFiles([{ path: 'a' }]), undefined)
  assert.equal(narrowSearchFiles([{ path: 'a', matches: [{ lineNumber: 0, line: 'x' }] }]), undefined)
  assert.deepEqual(narrowSearchFiles([{ path: 'a', matches: [{ lineNumber: 1, line: 'x' }] }]), [
    { path: 'a', matches: [{ lineNumber: 1, line: 'x' }] },
  ])
})

test('presentCall and presentResult shape the card', async () => {
  const tool = toolWith(async () => ({
    exit: 0,
    stdout: JSON.stringify(symbolGroup),
    stderr: '',
    spawnError: null,
    stdoutCap: false,
    aborted: false,
  }))
  const call = tool.presentCall({ pattern: 'x', path: 'src', include: '*.go' })
  assert.equal(call.kind, 'search')
  assert.match(call.title, /Grep x in src \(\*\.go\)/)
  const value = await tool.execute({ pattern: 'x' }, { cwd: root })
  const meta = tool.output.presentationMeta({ pattern: 'x' }, value)
  assert.equal(meta.shape, 'matches')
  const card = tool.presentResult({}, { meta })
  assert.equal(card.shape, 'matches')
  assert.equal(card.files.length, 1)
  assert.equal(tool.presentResult({}, { isError: true }), undefined)
  assert.equal(tool.presentResult({}, { meta: null }), undefined)
  assert.equal(tool.presentResult({}, { meta: { shape: 'other' } }), undefined)
  assert.equal(tool.presentResult({}, { meta: { shape: 'matches', truncated: false, total: 1, files: [{}] } }), undefined)
})

test('render carries the index sentence and the note', async () => {
  const tool = toolWith(async () => ({
    exit: 0,
    stdout: JSON.stringify(symbolGroup),
    stderr: 'note\n',
    spawnError: null,
    stdoutCap: false,
    aborted: false,
  }))
  const value = await tool.execute({ pattern: 'x' }, { cwd: root })
  const blocks = tool.output.render({ pattern: 'x' }, value)
  assert.equal(blocks[0].type, 'text')
  assert.match(blocks[0].text, /index used/)
  assert.match(blocks[0].text, /> note/)
  const plain = tool.output.render({ pattern: 'x' }, { ...value, notes: [] })
  assert.doesNotMatch(plain[0].text, /> note/)
})

test('resolveCwd falls back to the process directory', () => {
  assert.equal(resolveCwd({ cwd: '/x' }), '/x')
  assert.equal(resolveCwd({ agent: { session: { cwd: '/y' } } }), '/y')
  assert.equal(resolveCwd(null), process.cwd())
})

test('apply registers on live agents, created agents and the host once', () => {
  const calls = []
  const makeTools = label => ({
    register(tool) {
      calls.push(`${label}:${tool.name}`)
      return () => calls.push(`${label}:dispose`)
    },
    get: () => undefined,
  })
  const created = []
  const effects = []
  const agents = [{ ctx: { tools: makeTools('agent'), effect: fn => effects.push(fn) } }]
  const ctx = {
    tools: makeTools('host'),
    get: key => (key === 'agents' ? { list: () => agents } : undefined),
    on: (event, handler) => {
      created.push({ event, handler })
      return () => calls.push('stop-created')
    },
    effect: fn => effects.push(fn),
  }
  apply(ctx, {})
  assert.deepEqual(calls, ['agent:grep', 'host:grep'])
  assert.equal(created[0].event, 'agent/created')
  const late = { ctx: { tools: makeTools('late'), effect: fn => effects.push(fn) } }
  created[0].handler({ agent: late })
  assert.equal(calls.includes('late:grep'), true)
  created[0].handler({ agent: late })
  assert.equal(calls.filter(c => c === 'late:grep').length, 1)
  for (const effect of effects) {
    const cleanup = effect()
    if (typeof cleanup === 'function') cleanup()
  }
  assert.equal(calls.includes('stop-created'), true)
  assert.equal(calls.includes('host:dispose'), true)
})

test('apply skips the host when grep is already registered', () => {
  const calls = []
  const ctx = {
    tools: { register: () => calls.push('registered'), get: () => ({ name: 'grep' }) },
    get: () => undefined,
    on: () => () => {},
    effect: () => {},
  }
  apply(ctx, {})
  assert.deepEqual(calls, [])
})

test('apply logs a failing agent registration and keeps going', () => {
  const warnings = []
  const ctx = {
    tools: { register: () => () => {}, get: () => undefined },
    get: key => (key === 'agents' ? { list: () => [{ ctx: {
      tools: { register: () => { throw new Error('nope') }, get: () => undefined },
      logger: { warn: (...args) => warnings.push(args) },
      effect: () => {},
    } }] } : undefined),
    on: () => () => {},
    effect: () => {},
  }
  apply(ctx, {})
  assert.equal(warnings.length, 1)
  assert.match(warnings[0][0], /failed to register grep on agent/)
})

test('apply logs a failing host registration', () => {
  const warnings = []
  const ctx = {
    tools: { register: () => { throw new Error('nope') }, get: () => undefined },
    get: () => undefined,
    on: () => () => {},
    effect: () => {},
    logger: { warn: (...args) => warnings.push(args) },
  }
  apply(ctx, {})
  assert.equal(warnings.length, 1)
  assert.match(warnings[0][0], /failed to register grep on the host context/)
})