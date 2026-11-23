/**
 * Tests for the hook definition resolver: the chain an explicit path walks
 * through the loomux, Claude and codex positions down to hooks/hooks.json,
 * and the Claude-dialect parser that normalizes both the matcher-group and
 * the flat handler shape into { event, matcher, command, timeoutMs } entries.
 */

import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { resolveConfig } from '../src/config.js'

/**
 * Build a throwaway workdir holding the given files. Keys are slash-relative
 * paths; an object value is written as JSON, a string value verbatim.
 */
function project(files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'dsh-hooks-config-'))
  for (const [rel, contents] of Object.entries(files)) {
    const target = path.join(dir, ...rel.split('/'))
    fs.mkdirSync(path.dirname(target), { recursive: true })
    fs.writeFileSync(target, typeof contents === 'string' ? contents : JSON.stringify(contents))
  }
  return dir
}

/** Collect the warnings a resolution reports instead of printing them. */
function recorder() {
  const warnings = []
  return { warnings, onWarn: text => { warnings.push(text) } }
}

test('the explicit config path wins over every default position', t => {
  const dir = project({
    'custom/my-hooks.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-explicit' }] }] } },
    '.loomux/dsh/hooks.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-loomux' }] }] } },
    '.claude/settings.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-claude' }] }] } },
    'hooks/hooks.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-codex' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const explicit = path.join(dir, 'custom', 'my-hooks.json')
  const { warnings, onWarn } = recorder()
  const result = resolveConfig(explicit, dir, onWarn)
  assert.equal(result.sourcePath, explicit)
  assert.deepEqual(result.entries, [
    { event: 'Stop', matcher: '.*', command: 'from-explicit', timeoutMs: undefined },
  ])
  assert.deepEqual(warnings, [])
})

test('.loomux/dsh/hooks.json wins over the later positions', t => {
  const dir = project({
    '.loomux/dsh/hooks.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-loomux' }] }] } },
    '.claude/settings.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-claude' }] }] } },
    'hooks/hooks.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-codex' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const result = resolveConfig(undefined, dir)
  assert.equal(result.sourcePath, path.join(dir, '.loomux', 'dsh', 'hooks.json'))
  assert.equal(result.entries[0].command, 'from-loomux')
})

test('.claude/settings.json wins when the loomux position is absent', t => {
  const dir = project({
    '.claude/settings.json': {
      permissions: { allow: ['Bash'] },
      hooks: { Stop: [{ matcher: 'x', hooks: [{ type: 'command', command: 'from-claude' }] }] },
    },
    'hooks/hooks.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-codex' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const result = resolveConfig(undefined, dir)
  assert.equal(result.sourcePath, path.join(dir, '.claude', 'settings.json'))
  assert.equal(result.entries[0].command, 'from-claude')
})

test('hooks/hooks.json is the last default position', t => {
  const dir = project({
    'hooks/hooks.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-codex' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  const result = resolveConfig(undefined, dir, onWarn)
  assert.equal(result.sourcePath, path.join(dir, 'hooks', 'hooks.json'))
  assert.equal(result.entries[0].command, 'from-codex')
  assert.deepEqual(warnings, [])
})

test('no file anywhere resolves to null without a warning', t => {
  const dir = project({})
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.deepEqual(warnings, [])
})

test('an explicit path that does not exist falls through to the defaults', t => {
  const dir = project({
    '.claude/settings.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-claude' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const missing = path.join(dir, 'nowhere.json')
  const result = resolveConfig(missing, dir)
  assert.equal(result.sourcePath, path.join(dir, '.claude', 'settings.json'))
})

test('a broken JSON file at the first position yields null and does not fall through', t => {
  const dir = project({
    '.loomux/dsh/hooks.json': '{ this is not json',
    '.claude/settings.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-claude' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.equal(warnings.length, 1)
  assert.match(warnings[0], /invalid JSON/)
  assert.match(warnings[0], /\.loomux/)
})

test('a file without a hooks object yields null with a warning', t => {
  const dir = project({
    '.claude/settings.json': { permissions: { allow: ['Bash'] } },
    'hooks/hooks.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'from-codex' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.equal(warnings.length, 1)
  assert.match(warnings[0], /"hooks" object/)
})

test('a hooks value that is not an object yields null with a warning', t => {
  const dir = project({ '.claude/settings.json': { hooks: [] } })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.equal(warnings.length, 1)
  assert.match(warnings[0], /"hooks" object/)
})

test('an empty hooks object suppresses all hooks without a warning', t => {
  const dir = project({ '.claude/settings.json': { hooks: {} } })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  const result = resolveConfig(undefined, dir, onWarn)
  assert.deepEqual(result, { entries: [], sourcePath: path.join(dir, '.claude', 'settings.json') })
  assert.deepEqual(warnings, [])
})

test('handlers of a type other than command are dropped with a warning', t => {
  const dir = project({
    '.claude/settings.json': {
      hooks: {
        Stop: [{
          matcher: '^done$',
          hooks: [
            { type: 'command', command: 'echo stop' },
            { type: 'prompt', prompt: 'wrap up' },
          ],
        }],
      },
    },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  const result = resolveConfig(undefined, dir, onWarn)
  assert.deepEqual(result.entries, [
    { event: 'Stop', matcher: '^done$', command: 'echo stop', timeoutMs: undefined },
  ])
  assert.equal(warnings.length, 1)
  assert.match(warnings[0], /"prompt"/)
})

test('a broken regex matcher rejects the whole file with a warning naming it', t => {
  const dir = project({
    '.claude/settings.json': {
      hooks: {
        PreToolUse: [{ matcher: 'Write[', hooks: [{ type: 'command', command: 'echo p' }] }],
        Stop: [{ hooks: [{ type: 'command', command: 'echo s' }] }],
      },
    },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.equal(warnings.length, 1)
  assert.match(warnings[0], /Write\[/)
})

test('a matcher that is not a string rejects the file', t => {
  const dir = project({
    '.claude/settings.json': { hooks: { Stop: [{ matcher: 42, hooks: [{ type: 'command', command: 'c' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.match(warnings[0], /matcher must be a string/)
})

test('events without a matcher normalize to .*', t => {
  const dir = project({
    '.claude/settings.json': {
      hooks: {
        SessionStart: [{ hooks: [{ type: 'command', command: 'echo s' }] }],
        Stop: [{ type: 'command', command: 'echo t' }],
      },
    },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const result = resolveConfig(undefined, dir)
  assert.deepEqual(result.entries, [
    { event: 'SessionStart', matcher: '.*', command: 'echo s', timeoutMs: undefined },
    { event: 'Stop', matcher: '.*', command: 'echo t', timeoutMs: undefined },
  ])
})

test('timeout seconds become timeoutMs and a missing timeout stays undefined', t => {
  const dir = project({
    '.claude/settings.json': {
      hooks: { Stop: [{ hooks: [
        { type: 'command', command: 'echo slow', timeout: 15 },
        { type: 'command', command: 'echo fast' },
      ] }] },
    },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const result = resolveConfig(undefined, dir)
  assert.deepEqual(result.entries, [
    { event: 'Stop', matcher: '.*', command: 'echo slow', timeoutMs: 15000 },
    { event: 'Stop', matcher: '.*', command: 'echo fast', timeoutMs: undefined },
  ])
  assert.deepEqual(Object.keys(result.entries[1]).sort(), ['command', 'event', 'matcher', 'timeoutMs'])
})

test('a timeout that is not a non-negative number of seconds rejects the file', t => {
  const dir = project({ '.claude/settings.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'c', timeout: '15' }] }] } } })
  const dir2 = project({ '.claude/settings.json': { hooks: { Stop: [{ hooks: [{ type: 'command', command: 'c', timeout: -5 }] }] } } })
  t.after(() => {
    fs.rmSync(dir, { recursive: true, force: true })
    fs.rmSync(dir2, { recursive: true, force: true })
  })
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.equal(resolveConfig(undefined, dir2, onWarn), null)
  assert.equal(warnings.length, 2)
  assert.match(warnings[0], /timeout/)
})

test('the flat variant normalizes to the same entries as the matcher-group variant', t => {
  const flat = project({
    '.claude/settings.json': { hooks: { PreToolUse: [{ type: 'command', command: 'echo flat', timeout: 5 }] } },
  })
  const grouped = project({
    'hooks.json': { hooks: { PreToolUse: [{ matcher: '.*', hooks: [{ type: 'command', command: 'echo flat', timeout: 5 }] }] } },
  })
  t.after(() => {
    fs.rmSync(flat, { recursive: true, force: true })
    fs.rmSync(grouped, { recursive: true, force: true })
  })
  const flatResult = resolveConfig(undefined, flat)
  const groupedResult = resolveConfig(path.join(grouped, 'hooks.json'), flat)
  assert.deepEqual(flatResult.entries, [
    { event: 'PreToolUse', matcher: '.*', command: 'echo flat', timeoutMs: 5000 },
  ])
  assert.deepEqual(groupedResult.entries, flatResult.entries)
})

test('a flat handler may carry its own matcher and a group matcher wins over it', t => {
  const dir = project({
    '.claude/settings.json': {
      hooks: {
        PostToolUse: [{ type: 'command', command: 'echo own', matcher: 'Edit|Write' }],
        PreToolUse: [{ matcher: 'G', hooks: [{ type: 'command', command: 'echo g', matcher: 'H' }] }],
      },
    },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const result = resolveConfig(undefined, dir)
  assert.deepEqual(result.entries, [
    { event: 'PostToolUse', matcher: 'Edit|Write', command: 'echo own', timeoutMs: undefined },
    { event: 'PreToolUse', matcher: 'G', command: 'echo g', timeoutMs: undefined },
  ])
})

test('events outside the known seven are kept as written', t => {
  const dir = project({
    'hooks/hooks.json': { hooks: { PreCompact: [{ hooks: [{ type: 'command', command: 'echo c' }] }] } },
  })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const result = resolveConfig(undefined, dir)
  assert.deepEqual(result.entries, [
    { event: 'PreCompact', matcher: '.*', command: 'echo c', timeoutMs: undefined },
  ])
})

test('an event whose value is not a list rejects the file', t => {
  const dir = project({ '.claude/settings.json': { hooks: { Stop: 'echo nope' } } })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.match(warnings[0], /hooks\.Stop is not a list/)
})

test('a list item that is not an object rejects the file', t => {
  const dir = project({ '.claude/settings.json': { hooks: { Stop: ['echo flat-but-string'] } } })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.match(warnings[0], /item is not an object/)
})

test('a matcher group whose hooks is not a list rejects the file', t => {
  const dir = project({ '.claude/settings.json': { hooks: { Stop: [{ matcher: 'm', hooks: 'nope' }] } } })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.match(warnings[0], /has no "hooks" list/)
})

test('a handler that is not an object rejects the file', t => {
  const dir = project({ '.claude/settings.json': { hooks: { Stop: [{ matcher: 'm', hooks: [7] }] } } })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.match(warnings[0], /handler is not an object/)
})

test('a command handler without a command string rejects the file', t => {
  const dir = project({ '.claude/settings.json': { hooks: { Stop: [{ hooks: [{ type: 'command' }] }] } } })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const { warnings, onWarn } = recorder()
  assert.equal(resolveConfig(undefined, dir, onWarn), null)
  assert.match(warnings[0], /non-empty string/)
})

test('without an onWarn callback a broken file still resolves to null silently', t => {
  const dir = project({ '.claude/settings.json': 'not json at all' })
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  assert.equal(resolveConfig(undefined, dir), null)
})
