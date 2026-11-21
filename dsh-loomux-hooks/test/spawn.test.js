/**
 * Tests for the hook process runner. Every test spawns through a real shell.
 */

import assert from 'node:assert/strict'
import path from 'node:path'
import test from 'node:test'

import { runHook } from '../src/spawn.js'

const packageDir = path.resolve(import.meta.dirname, '..')
// Outside test/ so that `node --test` does not run it as a test file itself.
const fixture = path.join(packageDir, 'fixtures', 'echo-payload.mjs')
const node = `"${process.execPath}"`

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
