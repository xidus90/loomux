import test from 'node:test'
import assert from 'node:assert/strict'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { route, underPrefix } from '../src/route.js'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')

test('a call without a path runs on loomux without a prefix', () => {
  assert.deepEqual(route({ pattern: 'x' }, root), { engine: 'loomux' })
})

test('a path inside the root becomes a repository-relative prefix', () => {
  assert.deepEqual(route({ pattern: 'x', path: 'internal/code' }, root), {
    engine: 'loomux',
    path: 'internal/code',
  })
  assert.deepEqual(route({ pattern: 'x', path: root }, root), { engine: 'loomux' })
  assert.deepEqual(route({ pattern: 'x', path: '.' }, root), { engine: 'loomux' })
})

test('a path outside the root is an error with a hint', () => {
  assert.throws(() => route({ pattern: 'x', path: '..' }, root), /outside the workspace root/)
  assert.throws(() => route({ pattern: 'x', path: resolve(root, '..', 'elsewhere') }, root), /outside/)
})

test('an empty path is rejected', () => {
  assert.throws(() => route({ pattern: 'x', path: '' }, root), /non-empty string/)
  assert.throws(() => route({ pattern: 'x', path: 42 }, root), /non-empty string/)
})

test('a missing workspace root is an error', () => {
  assert.throws(() => route({ pattern: 'x' }, ''), /workspace root is unknown/)
  assert.throws(() => route({ pattern: 'x' }, undefined), /workspace root is unknown/)
})

test('any extension filter still routes to loomux', () => {
  for (const include of ['*.md', '*.go', '**/*.py']) {
    assert.deepEqual(route({ pattern: 'x', include }, root), { engine: 'loomux' })
  }
})

test('the prefix boundary is a path segment', () => {
  assert.ok(underPrefix('internal/code', 'internal/code'))
  assert.ok(underPrefix('internal/code/query/grep.go', 'internal/code'))
  assert.ok(!underPrefix('internal/codex/file.go', 'internal/code'))
  assert.ok(underPrefix('anything', ''))
  assert.ok(underPrefix('anything', '.'))
  assert.deepEqual(route({ pattern: 'x', path: join('internal', 'code') }, root), {
    engine: 'loomux',
    path: 'internal/code',
  })
})
