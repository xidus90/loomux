/**
 * The hook process runner: how one hook command is executed.
 *
 * A hook runs through a shell, in the project directory, with `CLAUDE_PROJECT_DIR`
 * pointing at it and the JSON payload written to stdin followed by a newline.
 * The runner never rejects: a timeout kills the child and reports `timedOut`,
 * a spawn failure lands in the same result shape with its message on stderr.
 */

import { spawn } from 'node:child_process'

/**
 * Run one hook command and collect its result.
 *
 * @param {string} command the shell command line of the hook
 * @param {unknown} payload the JSON payload written to the child's stdin
 * @param {{ cwd?: string, timeoutMs?: number }} [options]
 * @returns {Promise<{ exitCode: number|null, stdout: string, stderr: string, timedOut: boolean }>}
 */
export function runHook(command, payload, { cwd = process.cwd(), timeoutMs } = {}) {
  return new Promise(resolve => {
    const signal = typeof timeoutMs === 'number' && timeoutMs > 0
      ? AbortSignal.timeout(timeoutMs)
      : undefined
    const child = spawn(command, {
      shell: true,
      cwd,
      env: { ...process.env, CLAUDE_PROJECT_DIR: cwd },
      signal,
    })

    let stdout = ''
    let stderr = ''
    let settled = false

    const finish = exitCode => {
      if (settled) return
      settled = true
      const timedOut = signal?.aborted === true
      resolve({ exitCode: timedOut ? null : exitCode, stdout, stderr, timedOut })
    }

    child.stdout?.on('data', chunk => { stdout += chunk })
    child.stderr?.on('data', chunk => { stderr += chunk })

    if (child.stdin) {
      // The child may exit before reading stdin; a failed write must not throw.
      child.stdin.on('error', () => { /* ignore */ })
      child.stdin.end(JSON.stringify(payload) + '\n')
    }

    child.on('error', error => {
      if (signal?.aborted !== true && error?.message) stderr += error.message
      finish(null)
    })
    // A killed child may keep its stdio pipes open through a shell grandchild;
    // the exit event of the shell itself settles a timed-out hook regardless.
    child.on('exit', code => {
      if (signal?.aborted === true) finish(code)
    })
    child.on('close', code => { finish(code) })
  })
}
