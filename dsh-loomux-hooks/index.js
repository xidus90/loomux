/**
 * dsh-loomux-hooks: native DeepSeek Harness hooks backed by loomux.
 *
 * The package skeleton so far is the hook process runner: src/spawn.js runs
 * one hook command the way the hook protocol expects and reports its exit
 * code, output and timeout state. The hook entries consuming it register
 * through cordis.patch.yml and arrive with the plugin wiring.
 */

export { runHook } from './src/spawn.js'

export const name = 'dsh-loomux-hooks'
export const inject = []
