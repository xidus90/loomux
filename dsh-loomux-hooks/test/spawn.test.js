/**
 * Tests for the hook process runner. Every test spawns through a real shell.
 */

import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { runHook } from '../src/spawn.js'

const packageDir = path.resolve(import.meta.dirname, '..')
// Outside test/ so that `node --test` does not run it as a test file itself.
const fixture = path.join(packageDir, 'fixtures', 'echo-payload.mjs')
const node = `"${process.execPath}"`

const sleep = ms => new Promise(resolve => setTimeout(resolve, ms))

test('a command producing output writes stdout and exits 0', async () => {
  const result = await runHook(`${node} -e "console.log('hook-out')"`, {}, { cwd: packageDir })
  assert.equal(result.exitCode, 0)
  assert.equal(result.timedOut, false)
  assert.match(result.stdout, /hook-out\n/)
  assert.equal(result.stderr, '')
})

test('process.exit(2) yields exitCode 2 and keeps stderr', async () => {
  const result = await runHook(
    `${node} -e "console.error('boom'); process.exit(2)"`,
    {},
    { cwd: packageDir },
  )
  assert.equal(result.exitCode, 2)
  assert.equal(result.timedOut, false)
  assert.match(result.stderr, /boom/)
})

test('a child reading stdin receives the JSON payload', async () => {
  const payload = { hook_event_name: 'SessionStart', cwd: packageDir, n: 7 }
  const result = await runHook(`${node} "${fixture}"`, payload, { cwd: packageDir })
  assert.equal(result.exitCode, 0)
  assert.equal(result.timedOut, false)
  assert.deepEqual(JSON.parse(result.stdout), { received: payload })
})

test('timeoutMs kills a sleeping child', async () => {
  const started = Date.now()
  const result = await runHook(
    `${node} -e "setTimeout(() => process.exit(0), 60000)"`,
    {},
    { cwd: packageDir, timeoutMs: 150 },
  )
  assert.equal(result.timedOut, true)
  assert.equal(result.exitCode, null)
  assert.ok(Date.now() - started < 30000, 'the killed child resolves promptly')
})

test('a timeout kills the grandchild the shell started', async () => {
  // The hook command spawns a grandchild that inherits the shell's stdio and
  // appends to a marker file every 50 ms for 30 seconds, far beyond the
  // 500 ms test timeout. Killing only the cmd.exe wrapper leaves it writing
  // the marker and holding the pipes open; the whole tree has to die.
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'dsh-hooks-tree-'))
  // Forward slashes keep the path free of escapes for the nested shell layer.
  const marker = path.join(dir, 'marker.txt').replaceAll('\\', '/')
  // The grandchild source travels inside a template literal in the hook
  // source, which itself travels inside the double-quoted -e argument of the
  // shell command: single quotes and backticks only, no double quotes.
  const grandchild =
    "const fs = require('node:fs');" +
    `fs.writeFileSync('${marker}', 'x');` +
    `setInterval(() => fs.appendFileSync('${marker}', 'x'), 50);` +
    'setTimeout(() => process.exit(0), 30000)'
  const hook =
    "const cp = require('node:child_process');" +
    `cp.spawn(process.execPath, ['-e', \`${grandchild}\`], { stdio: 'inherit' });` +
    'setTimeout(() => process.exit(0), 15000)'
  const result = await runHook(`${node} -e "${hook}"`, {}, { cwd: packageDir, timeoutMs: 500 })
  assert.equal(result.timedOut, true)
  assert.equal(result.exitCode, null)
  await sleep(600) // let the tree killer finish its work
  const first = fs.statSync(marker).size
  assert.ok(first > 0, 'the grandchild wrote the marker while it lived')
  await sleep(400)
  const second = fs.statSync(marker).size
  await sleep(400)
  const third = fs.statSync(marker).size
  assert.equal(second, first, 'the grandchild stopped writing after the abort')
  assert.equal(third, second, 'the marker stays stable over a second interval')
  fs.rmSync(dir, { recursive: true, force: true })
})

test('a command path containing a # survives the Windows shell', async () => {
  // The checkout does not necessarily live under a directory named "#GIT",
  // and a forge checkout never does, so let a temporary directory carry the
  // #: the absolute fixture path must travel through the shell to node
  // unchanged wherever the repository lives.
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'dsh-hooks-hash#-'))
  const hashed = path.join(dir, 'echo-payload.mjs')
  assert.match(hashed, /#/)
  fs.copyFileSync(fixture, hashed)
  try {
    const result = await runHook(`${node} "${hashed}"`, { hook_event_name: 'hash-check' }, { cwd: packageDir })
    assert.equal(result.exitCode, 0)
    assert.equal(result.timedOut, false)
    assert.deepEqual(JSON.parse(result.stdout), { received: { hook_event_name: 'hash-check' } })
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('a spawn error resolves in the same shape instead of rejecting', async () => {
  const missing = path.join(packageDir, 'no-such-directory')
  const result = await runHook(`${node} -e "console.log(1)"`, {}, { cwd: missing })
  assert.equal(result.exitCode, null)
  assert.equal(result.timedOut, false)
  assert.equal(result.stdout, '')
  assert.match(result.stderr, /ENOENT/)
})

test('a stdin write the exited child never read still reports the result', async () => {
  // The shell exits without touching stdin; the multi-megabyte payload cannot
  // fit the pipe buffer, so the pending write fails once the child is gone.
  const payload = { blob: 'x'.repeat(4 * 1024 * 1024) }
  const result = await runHook('exit 3', payload, { cwd: packageDir })
  assert.equal(result.exitCode, 3)
  assert.equal(result.timedOut, false)
})
