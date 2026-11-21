/**
 * The hook process runner: how one hook command is executed.
 *
 * A hook runs through a shell, in the project directory, with `CLAUDE_PROJECT_DIR`
 * pointing at it and the JSON payload written to stdin followed by a newline.
 * The runner never rejects: a timeout kills the whole child process tree and
 * reports `timedOut`, a spawn failure lands in the same result shape with its
 * message on stderr.
 */

import { spawn } from 'node:child_process'

/**
 * Error sink for child processes whose failure never changes the result:
 * the hook's stdin pipe once the child is gone, the launcher of the tree
 * killer once its target is gone.
 */
const swallow = () => { /* the process is already gone; nothing to report */ }

/**
 * Kill the hook and everything the shell started below it.
 *
 * On Windows the shell is cmd.exe wrapping the hook program, so killing only
 * the shell would leave the hook alive as a grandchild: holding the stdio
 * pipes open and running a second time when the next hook invocation starts.
 * `taskkill /T /F` terminates the whole tree by pid, and the shell may already
 * be gone when it runs, so its errors are swallowed. There is no taskkill on
 * POSIX: the spawn fails harmlessly, its error is swallowed, and the SIGKILL
 * reaches the shell, which is the root of the process tree there. Both forms
 * execute on every platform, so every line of this function stays covered on
 * every host.
 *
 * @param {{ pid?: number, kill: (signal?: NodeJS.Signals) => boolean }} child
 *   the running shell child process
 */
function killTree(child) {
  const killer = spawn('taskkill', ['/T', '/F', '/PID', String(child.pid)], {
    shell: false,
    stdio: 'ignore',
  })
  killer.on('error', swallow)
  killer.unref()
  // On Windows taskkill owns the tree kill; short-circuiting keeps a direct
  // SIGKILL from racing taskkill's tree walk while it enumerates the shell.
  process.platform === 'win32' || child.kill('SIGKILL')
}

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
    const child = spawn(command, {
      shell: true,
      cwd,
      env: { ...process.env, CLAUDE_PROJECT_DIR: cwd },
    })

    let stdout = ''
    let stderr = ''
    let settled = false
    let timedOut = false

    // The timer kills the tree itself: an AbortSignal handed to spawn would
    // only terminate the shell wrapper and let the hook program survive.
    const timer = typeof timeoutMs === 'number' && timeoutMs > 0
      ? setTimeout(() => { timedOut = true; killTree(child) }, timeoutMs)
      : undefined

    const finish = exitCode => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      resolve({ exitCode: timedOut ? null : exitCode, stdout, stderr, timedOut })
    }

    child.stdout?.on('data', chunk => { stdout += chunk })
    child.stderr?.on('data', chunk => { stderr += chunk })

    if (child.stdin) {
      // The child may exit before reading stdin; a failed write must not throw.
      child.stdin.on('error', swallow)
      child.stdin.end(JSON.stringify(payload) + '\n')
    }

    child.on('error', error => {
      if (!timedOut && error?.message) stderr += error.message
      finish(null)
    })
    // A killed child may keep its stdio pipes open through a shell grandchild;
    // the exit event of the shell itself settles a timed-out hook regardless.
    child.on('exit', code => {
      if (timedOut) finish(code)
    })
    child.on('close', code => { finish(code) })
  })
}
