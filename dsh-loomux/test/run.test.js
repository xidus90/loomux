import test from 'node:test'
import assert from 'node:assert/strict'

import { STDOUT_CAP_BYTES, runLoomux } from '../src/run.js'

const node = process.execPath

test('the stdout cap is 20 MiB', () => {
  assert.equal(STDOUT_CAP_BYTES, 20 * 1024 * 1024)
})

test('a successful child returns its streams and exit zero', async () => {
  const run = await runLoomux(['-e', 'process.stdout.write("{\\"groups\\":null}"); process.stderr.write("note\\n")'], {
    binary: node,
  })
  assert.equal(run.exit, 0)
  assert.equal(run.stdout, '{"groups":null}')
  assert.equal(run.stderr, 'note\n')
  assert.equal(run.spawnError, null)
  assert.equal(run.stdoutCap, false)
  assert.equal(run.aborted, false)
})

test('a failing child keeps stderr and the exit code', async () => {
  const run = await runLoomux(['-e', 'process.stderr.write("invalid regex: bad\\n"); process.exit(1)'], { binary: node })
  assert.equal(run.exit, 1)
  assert.match(run.stderr, /invalid regex/)
})

test('a missing binary is reported as a spawn error', async () => {
  const run = await runLoomux([], { binary: 'definitely-not-a-binary-xyz' })
  assert.equal(run.spawnError.code, 'ENOENT')
})

test('output above the cap fails the run instead of truncating', async () => {
  const run = await runLoomux(['-e', 'process.stdout.write("x".repeat(200000))'], {
    binary: node,
    capBytes: 1024,
  })
  assert.equal(run.stdoutCap, true)
  assert.ok(run.stdout.length < 200000)
})

test('an abort settles the run as aborted', async () => {
  const controller = new AbortController()
  const pending = runLoomux(['-e', 'setTimeout(() => {}, 60000)'], { binary: node, signal: controller.signal })
  controller.abort()
  const run = await pending
  assert.equal(run.aborted, true)
  assert.equal(run.spawnError, null)
})
