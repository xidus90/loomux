import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  capSearchMeta,
  describeIndex,
  groupMatchesByFile,
  presentationMeta,
  previewLine,
  renderText,
  toMatches,
} from '../src/result.js'

function fixture(name) {
  return JSON.parse(readFileSync(new URL(`./fixtures/${name}`, import.meta.url), 'utf8'))
}

test('a group without a symbol stays a valid file hit', () => {
  const matches = toMatches(fixture('grep-file-group.json'))
  assert.equal(matches.length, 3)
  assert.equal(matches[0].path, 'docs/de/benchmarks.md')
  assert.equal(matches[0].lineNumber, 3961)
  assert.equal(typeof matches[0].line, 'string')
  assert.equal(matches[0].symbol, undefined)
  assert.equal(matches[0].kind, undefined)
})

test('a group with a symbol carries name, kind, signature and inDegree', () => {
  const [match] = toMatches(fixture('grep-symbol-group.json'))
  assert.equal(match.path, 'internal/code/query/grep.go')
  assert.equal(match.lineNumber, 284)
  assert.equal(match.symbol, 'GrepReport')
  assert.equal(match.kind, 'function')
  assert.equal(match.signature, 'func GrepReport(r grep.Result) string')
  assert.equal(match.inDegree, 6)
})

test('an empty answer is no matches with the index sentence', () => {
  const text = renderText(fixture('grep-empty.json'))
  assert.match(text, /\(no matches\)/)
  assert.match(text, /index used \(0 of 7656 files in scope\)/)
  assert.match(text, /left out: 12 ignored by git, 6 binary/)
})

test('the render text names the index state and the counters', () => {
  const text = renderText(fixture('grep-symbol-group.json'))
  assert.match(text, /index used \(2 of 7653 files in scope\)/)
  assert.match(text, /left out: 12 ignored by git, 6 binary/)
  assert.match(text, /GrepReport/)
})

test('describeIndex covers every index state', () => {
  assert.match(describeIndex({ index: 'scan', files_in_scope: 7 }), /index current but the pattern has no literal/)
  assert.match(describeIndex({ index: 'stale', files_in_scope: 7 }), /stale index/)
  assert.match(describeIndex({ index: 'none', files_in_scope: 7 }), /no index/)
  assert.equal(describeIndex({}), 'scan (no index, 0 files in scope)')
  assert.match(describeIndex({ index: 'used', skipped_size: 2, unreadable: 1 }), /2 over the size limit, 1 unreadable/)
})

test('presentationMeta keeps the card shape and previews the lines', () => {
  const meta = presentationMeta(fixture('grep-symbol-group.json'), 300)
  assert.equal(meta.shape, 'matches')
  assert.equal(meta.truncated, false)
  assert.equal(meta.total, 1)
  assert.equal(meta.files[0].path, 'internal/code/query/grep.go')
  assert.deepEqual(Object.keys(meta.files[0].matches[0]), ['lineNumber', 'line'])
  assert.equal(meta.files[0].matches[0].lineNumber, 284)
})

test('presentationMeta truncates beyond maxMatches and keeps the total', () => {
  const meta = presentationMeta(fixture('grep-file-group.json'), 1)
  assert.equal(meta.truncated, true)
  assert.equal(meta.total, 3)
  assert.equal(meta.files[0].matches.length, 1)
})

test('previewLine caps by UTF-8 bytes with an ellipsis', () => {
  assert.equal(previewLine('short'), 'short')
  assert.equal(previewLine(undefined), '')
  const long = 'ä'.repeat(200)
  const cut = previewLine(long, 10)
  assert.ok(Buffer.byteLength(cut, 'utf8') <= 10)
  assert.ok(cut.endsWith('…'))
})

test('groupMatchesByFile groups in order', () => {
  const files = groupMatchesByFile([
    { path: 'a', lineNumber: 1, line: 'x' },
    { path: 'b', lineNumber: 2, line: 'y' },
    { path: 'a', lineNumber: 3, line: 'z' },
  ])
  assert.deepEqual(files, [
    { path: 'a', matches: [{ lineNumber: 1, line: 'x' }, { lineNumber: 3, line: 'z' }] },
    { path: 'b', matches: [{ lineNumber: 2, line: 'y' }] },
  ])
})

test('capSearchMeta drops trailing files and flips truncated', () => {
  const meta = {
    shape: 'matches',
    truncated: false,
    total: 3,
    files: [
      { path: 'a'.repeat(40), matches: [{ lineNumber: 1, line: 'x' }] },
      { path: 'b'.repeat(40), matches: [{ lineNumber: 2, line: 'y' }] },
    ],
  }
  const capped = capSearchMeta(meta, 80)
  assert.equal(capped.truncated, true)
  assert.equal(capped.total, 3)
  assert.ok(capped.files.length >= 1)
  assert.ok(Buffer.byteLength(JSON.stringify(capped), 'utf8') <= 80 || capped.files.length === 1)
})
