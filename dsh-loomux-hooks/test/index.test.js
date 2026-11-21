/**
 * Tests for the plugin entry: the package index exposes the runner and the
 * plugin identity the cordis patch registers under.
 */

import assert from 'node:assert/strict'
import test from 'node:test'

import * as plugin from '../index.js'
import { runHook } from '../src/spawn.js'

test('index exposes the plugin identity and re-exports the runner', () => {
  assert.equal(plugin.name, 'dsh-loomux-hooks')
  assert.deepEqual(plugin.inject, [])
  assert.equal(plugin.runHook, runHook)
})
