import test from 'node:test'
import assert from 'node:assert/strict'

import { CLASSES, classify, errorFor, noteFromStderr } from '../src/errors.js'

test('a missing binary is its own class with a hint', () => {
  assert.equal(classify(null, '', { spawnError: { code: 'ENOENT' } }), 'no-binary')
  assert.match(errorFor('no-binary').message, /not on PATH/)
})

test('a missing graph names the build command', () => {
  const stderr = 'loomux graph grep: no graph. Run `loomux graph build` first.\n'
  assert.equal(classify(1, stderr), 'no-graph')
  assert.match(errorFor('no-graph').message, /loomux graph build/)
})

test('invalid and unsupported patterns are distinct', () => {
  assert.equal(classify(1, 'loomux graph grep: invalid regex: error parsing regexp'), 'invalid-regex')
  assert.equal(classify(1, 'loomux graph grep: lookaround not supported in RE2 regex: (?=x)'), 'unsupported')
  assert.match(errorFor('unsupported').message, /lookaround or backreference/)
})

test('an uncovered prefix is a failure, never an empty answer', () => {
  assert.equal(classify(1, 'loomux graph grep: prefix not indexed: vendor'), 'prefix-not-indexed')
  assert.match(errorFor('prefix-not-indexed').message, /search corpus/)
})

test('exit 2 is a usage error of the plugin', () => {
  assert.equal(classify(2, 'flag provided but not defined: -nope'), 'usage')
  assert.match(errorFor('usage').message, /bug in dsh-loomux/)
})

test('an unknown exit 1 is an error, never ok', () => {
  assert.equal(classify(1, 'something else'), 'unknown')
  assert.equal(errorFor('unknown', 'boom').message.includes('boom'), true)
})

test('exit 0 is ok and its stderr becomes a note', () => {
  assert.equal(classify(0, 'loomux graph grep: 9 files moved, rebuilding the graph\n'), 'ok')
  assert.match(noteFromStderr('note\n'), /note/)
  assert.equal(noteFromStderr(undefined), '')
})

test('abort, timeout and the stdout cap fail instead of truncating', () => {
  assert.equal(classify(null, '', { aborted: true }), 'timeout')
  assert.equal(classify(null, '', { stdoutCap: true }), 'stdout-cap')
  assert.match(errorFor('stdout-cap').message, /output cap/)
})

test('every class has a hint and the list is stable', () => {
  assert.deepEqual(CLASSES, [
    'no-binary',
    'no-graph',
    'invalid-regex',
    'unsupported',
    'prefix-not-indexed',
    'usage',
    'timeout',
    'stdout-cap',
    'unknown',
  ])
  for (const code of CLASSES) {
    assert.equal(typeof errorFor(code).message, 'string')
  }
})
