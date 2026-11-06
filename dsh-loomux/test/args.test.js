import test from 'node:test'
import assert from 'node:assert/strict'

import { buildArgs, clampMaxResults } from '../src/args.js'

test('pattern is the positional argument and must be non-empty', () => {
  assert.throws(() => buildArgs({}), /pattern must be a non-empty string/)
  assert.throws(() => buildArgs({ pattern: '' }), /pattern must be a non-empty string/)
  assert.deepEqual(buildArgs({ pattern: 'func Search' }).slice(0, 3), ['graph', 'grep', 'func Search'])
})

test('max_results is clamped to 20..5000 with 300 as the fallback', () => {
  assert.equal(clampMaxResults(0), 300)
  assert.equal(clampMaxResults('abc'), 300)
  assert.equal(clampMaxResults(Number.NaN), 300)
  assert.equal(clampMaxResults(5), 20)
  assert.equal(clampMaxResults(6000), 5000)
  assert.equal(clampMaxResults(300), 300)
  assert.deepEqual(
    buildArgs({ pattern: 'x', max_results: 5 }).slice(-2),
    ['--max-hits', '20'],
  )
})

test('every argument list ends with JSON and the cloud channel', () => {
  const args = buildArgs({ pattern: 'x' })
  assert.deepEqual(args.slice(0, 6), ['graph', 'grep', 'x', '--json', '--channel', 'cloud'])
})

test('include stays one argument, never a shell split', () => {
  const args = buildArgs({ pattern: 'x', include: '*.{js,jsx}' })
  assert.equal(args[args.indexOf('--glob') + 1], '*.{js,jsx}')
  const spaced = buildArgs({ pattern: 'x', include: 'a b/*.md' })
  assert.equal(spaced[spaced.indexOf('--glob') + 1], 'a b/*.md')
})

test('include accepts brace alternation and rejects negation and top-level commas', () => {
  assert.doesNotThrow(() => buildArgs({ pattern: 'x', include: '{a,b}/**' }))
  assert.throws(() => buildArgs({ pattern: 'x', include: '!*.md' }), /positive glob filter/)
  assert.throws(() => buildArgs({ pattern: 'x', include: '*.md,*.txt' }), /not a comma-separated list/)
  assert.throws(() => buildArgs({ pattern: 'x', include: '   ' }), /include must be a non-empty string/)
})

test('any extension filter is passed through unchanged', () => {
  for (const include of ['*.md', '*.go', '**/*.py']) {
    const args = buildArgs({ pattern: 'x', include })
    assert.equal(args[args.indexOf('--glob') + 1], include)
  }
})

test('path becomes --in and may not be only whitespace', () => {
  const args = buildArgs({ pattern: 'x', path: 'internal/code' })
  assert.equal(args[args.indexOf('--in') + 1], 'internal/code')
  assert.throws(() => buildArgs({ pattern: 'x', path: '   ' }), /path must be a non-empty string/)
})

test('case_insensitive becomes -i and is absent otherwise', () => {
  assert.ok(buildArgs({ pattern: 'x', case_insensitive: true }).includes('-i'))
  assert.ok(!buildArgs({ pattern: 'x' }).includes('-i'))
})

test('a missing input object is rejected', () => {
  assert.throws(() => buildArgs(null), /arguments must be an object/)
  assert.throws(() => buildArgs('x'), /arguments must be an object/)
})

test('no input can reach the local channel', () => {
  const hostile = {
    pattern: 'x',
    channel: 'local',
    args: ['--channel', 'local'],
    extraArgs: ['--channel', 'local'],
  }
  const args = buildArgs(hostile)
  assert.equal(args.filter(a => a === 'cloud').length, 1)
  assert.equal(args.filter(a => a === 'local').length, 0)
})
