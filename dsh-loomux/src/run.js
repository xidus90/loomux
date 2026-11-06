/**
 * Run one `loomux graph grep --json` call.
 *
 * The child is spawned with an argument list, never a shell, and stdout is
 * capped by byte length: an answer over the cap is a failure, not a silently
 * shortened result.
 */

import { spawn } from 'node:child_process'

export const STDOUT_CAP_BYTES = 20 * 1024 * 1024

/**
 * @param {string[]} args
 * @param {{ cwd?: string, signal?: AbortSignal, binary?: string, capBytes?: number }} options
 * @returns {Promise<{ exit: number|null, stdout: string, stderr: string, spawnError: object|null, stdoutCap: boolean, aborted: boolean }>}
 */
export function runLoomux(args, options = {}) {
  const binary = options.binary || process.env.LOOMUX_BIN || 'loomux'
  const capBytes = options.capBytes || STDOUT_CAP_BYTES
  const signal = options.signal
  return new Promise(resolve => {
    const child = spawn(binary, args, {
      cwd: options.cwd,
      signal,
      stdio: ['ignore', 'pipe', 'pipe'],
    })
    let stdout = ''
    let stderr = ''
    let stdoutBytes = 0
    let stdoutCap = false
    let spawnError = null
    let aborted = false
    let settled = false

    const finish = value => {
      if (settled) return
      settled = true
      resolve(value)
    }

    child.stdout?.setEncoding('utf8')
    child.stderr?.setEncoding('utf8')
    child.stdout?.on('data', chunk => {
      stdoutBytes += Buffer.byteLength(chunk, 'utf8')
      if (stdoutBytes > capBytes) {
        stdoutCap = true
        try { child.kill('SIGTERM') } catch { /* ignore */ }
        return
      }
      stdout += chunk
    })
    child.stderr?.on('data', chunk => {
      stderr += chunk
    })
    child.on('error', error => {
      if (error.name === 'AbortError') {
        aborted = true
      } else {
        spawnError = error
      }
    })
    child.on('close', exit => {
      finish({ exit, stdout, stderr, spawnError, stdoutCap, aborted })
    })
  })
}
