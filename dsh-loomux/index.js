/**
 * dsh-loomux: the grep tool of a DeepSeek Harness profile, served by
 * `loomux graph grep`.
 *
 * The plugin is a facade. It maps the tool schema to arguments, runs one
 * process and shapes the answer back into `matches`. It contains no search
 * logic, no privacy logic and no fallback: a failure is an error with a hint,
 * never an empty list.
 */

import { buildArgs, clampMaxResults } from './src/args.js'
import { classify, errorFor, noteFromStderr } from './src/errors.js'
import { route } from './src/route.js'
import { presentationMeta, renderText, toMatches } from './src/result.js'
import { runLoomux } from './src/run.js'

export const name = 'dsh-loomux'
export const inject = ['tools']

const TIMEOUT_MS = 30_000

/** The session's working directory, as the host exposes it. */
export function resolveCwd(exec) {
  try {
    const session = exec?.agent?.session ?? exec?.session
    const cwd =
      session?.header?.cwd ??
      session?.cwd ??
      exec?.cwd ??
      exec?.workspace?.cwd
    if (typeof cwd === 'string' && cwd.length > 0) return cwd
  } catch { /* ignore */ }
  return process.cwd()
}

/** Narrow replayed metadata into the search-card payload; never throws. */
export function narrowSearchFiles(value) {
  if (!Array.isArray(value)) return undefined
  const files = []
  for (const file of value) {
    if (typeof file !== 'object' || file === null || Array.isArray(file)) return undefined
    const { path, matches } = file
    if (typeof path !== 'string' || !Array.isArray(matches)) return undefined
    const narrowed = []
    for (const match of matches) {
      if (typeof match !== 'object' || match === null || Array.isArray(match)) return undefined
      const { lineNumber, line } = match
      if (!Number.isInteger(lineNumber) || lineNumber < 1 || typeof line !== 'string') return undefined
      narrowed.push({ lineNumber, line })
    }
    files.push({ path, matches: narrowed })
  }
  return files
}

/**
 * Parse the stdout of one call; a body that is not the documented shape fails
 * instead of degrading into an empty answer.
 */
export function parseAnswer(stdout) {
  let parsed
  try {
    parsed = JSON.parse(stdout)
  } catch (error) {
    throw new Error(`loomux graph grep returned no JSON: ${error.message}`)
  }
  if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('loomux graph grep returned no JSON object')
  }
  if (parsed.groups !== null && !Array.isArray(parsed.groups)) {
    throw new Error('loomux graph grep returned no groups array')
  }
  return parsed
}

/**
 * The model-facing tool definition.
 *
 * @param {{ maxResults?: number, binary?: string, trustIndex?: boolean }} [config]
 */
export function createGrepTool(config = {}) {
  const defaultMax = clampMaxResults(config.maxResults ?? 300)
  const run = config.run ?? runLoomux
  return {
    name: 'grep',
    description:
      'Search the repository with loomux graph grep. Covers every text file git does not ignore, ' +
      'through the trigram index when the pattern has a literal. Returns matching lines with line numbers, ' +
      'grouped by file, with the symbol of the enclosing node when there is one.',
    timeoutMs: TIMEOUT_MS,
    parameters: {
      type: 'object',
      properties: {
        pattern: {
          type: 'string',
          description: 'Regular expression or literal text to search for (RE2 syntax).',
        },
        path: {
          type: 'string',
          description: 'Directory or file inside the workspace to search (default: the workspace root).',
        },
        include: {
          type: 'string',
          description: 'One glob filter for file names (e.g. "*.ts", "src/**/*.go"). Not a list; brace alternation and negation are not supported.',
        },
        case_insensitive: {
          type: 'boolean',
          description: 'Case-insensitive search.',
        },
        max_results: {
          type: 'number',
          description: `Soft limit on returned matches, clamped to 20–5000 (default ${defaultMax}).`,
        },
      },
      required: ['pattern'],
    },
    output: {
      schema: {
        type: 'object',
        additionalProperties: true,
        properties: {
          matches: {
            type: 'array',
            items: {
              type: 'object',
              additionalProperties: true,
              properties: {
                path: { type: 'string' },
                lineNumber: { type: 'integer' },
                line: { type: 'string' },
                symbol: { type: 'string' },
                kind: { type: 'string' },
                signature: { type: 'string' },
                inDegree: { type: 'number' },
              },
              required: ['path', 'lineNumber', 'line'],
            },
          },
          notes: { type: 'array', items: { type: 'string' } },
        },
        required: ['matches'],
      },
      render: (_args, value) => {
        const text = renderText(value)
        const notes = Array.isArray(value?.notes) ? value.notes : []
        return [{ type: 'text', text: notes.length === 0 ? text : `${text}\n\n> ${notes.join('\n> ')}` }]
      },
      presentationMeta: (args, value) => presentationMeta(value, clampMaxResults(args?.max_results ?? defaultMax)),
    },
    presentCall: args => {
      const pattern = typeof args?.pattern === 'string' ? args.pattern : ''
      const where = typeof args?.path === 'string' && args.path !== '' ? ` in ${args.path}` : ''
      const filter = typeof args?.include === 'string' && args.include !== '' ? ` (${args.include})` : ''
      return { card: 'generic', title: `Grep ${pattern}${where}${filter}`, kind: 'search', rawInput: pattern }
    },
    presentResult: (_args, result) => {
      if (result?.isError) return undefined
      const meta = result?.meta
      if (meta === null || typeof meta !== 'object' || Array.isArray(meta)) return undefined
      if (meta.shape !== 'matches') return undefined
      if (typeof meta.truncated !== 'boolean') return undefined
      if (!Number.isInteger(meta.total) || meta.total < 0) return undefined
      const files = narrowSearchFiles(meta.files)
      if (files === undefined) return undefined
      return { card: 'search', shape: 'matches', files, truncated: meta.truncated, total: meta.total }
    },
    async execute(args, exec) {
      const cwd = resolveCwd(exec)
      const routed = route(args, cwd)
      const input = { ...args, max_results: args?.max_results ?? defaultMax }
      // The routed path replaces the given one; none means the whole corpus.
      if (routed.path !== undefined) input.path = routed.path
      else delete input.path
      const cliArgs = buildArgs(input, { trustIndex: config.trustIndex === true })
      const finished = await run(cliArgs, {
        cwd,
        signal: exec?.signal,
        binary: config.binary,
      })
      const code = classify(finished.exit, finished.stderr, {
        aborted: finished.aborted,
        stdoutCap: finished.stdoutCap,
        spawnError: finished.spawnError,
      })
      if (code !== 'ok') {
        throw errorFor(code, finished.stderr)
      }
      const parsed = parseAnswer(finished.stdout)
      const note = noteFromStderr(finished.stderr)
      return note === '' ? { ...parsed, matches: toMatches(parsed) } : { ...parsed, matches: toMatches(parsed), notes: [note] }
    },
  }
}

/**
 * Register the tool the way the host expects: on every agent's own tool scope,
 * on live agents, and on the host context when `grep` is not yet registered.
 */
export function apply(ctx, config = {}) {
  const tool = createGrepTool(config)
  const disposers = new Map()

  function attachToAgent(agent) {
    if (!agent || disposers.has(agent)) return
    const agentCtx = agent.ctx ?? agent
    const tools = agentCtx.tools ?? agentCtx.get?.('tools')
    if (!tools || typeof tools.register !== 'function') return
    try {
      const disposeTool = tools.register(tool)
      if (typeof disposeTool === 'function') {
        const cleanup = () => {
          try { disposeTool() } catch { /* ignore */ }
          disposers.delete(agent)
        }
        disposers.set(agent, cleanup)
        agentCtx.effect?.(() => cleanup)
      }
    } catch (error) {
      agentCtx.logger?.warn?.('[dsh-loomux] failed to register grep on agent', error)
    }
  }

  const stopCreated = ctx.on?.('agent/created', payload => {
    attachToAgent(payload?.agent ?? payload)
  })
  try {
    const agents = ctx.get?.('agents') ?? ctx.agents
    if (agents && typeof agents.list === 'function') {
      for (const agent of agents.list()) attachToAgent(agent)
    }
  } catch { /* ignore */ }

  let disposeHost = null
  const hostTools = ctx.tools ?? ctx.get?.('tools')
  if (hostTools && typeof hostTools.register === 'function' && !hostTools.get?.('grep')) {
    try {
      disposeHost = hostTools.register(tool)
    } catch (error) {
      ctx.logger?.warn?.('[dsh-loomux] failed to register grep on the host context', error)
    }
  }

  ctx.effect?.(() => () => {
    if (typeof stopCreated === 'function') {
      try { stopCreated() } catch { /* ignore */ }
    }
    if (typeof disposeHost === 'function') {
      try { disposeHost() } catch { /* ignore */ }
    }
    for (const cleanup of disposers.values()) {
      try { cleanup() } catch { /* ignore */ }
    }
    disposers.clear()
  })
}
