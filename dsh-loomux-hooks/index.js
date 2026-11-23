/**
 * dsh-loomux-hooks: native DeepSeek Harness hooks backed by loomux.
 *
 * The plugin registers the six harness seams and drives each one through the
 * loomux hook protocol: the hook entries resolve from the chain, every
 * matching command runs as a real hook process (src/spawn.js), and the exit
 * and JSON dialect decide the seam's answer (src/decision.js). Context, deny,
 * block and steer land as the harness-native actions the live spike pinned
 * down: inject, deny, block-feedback and steer. Every run appends the
 * core-known hook/invoked and hook/result records to the session log.
 */

import path from 'node:path'

import { decide, SEAM_EVENTS } from './src/decision.js'
import { resolveConfig as resolveChain } from './src/config.js'
import { runHook as runHookImpl } from './src/spawn.js'

export { runHookImpl as runHook }

export const name = 'dsh-loomux-hooks'
export const inject = []

/** The seam each decision-machine key registers at on the harness. */
const SEAM_POINTS = {
  sessionStart: 'agent/created',
  preToolUse: 'tools/pre-execute',
  postToolUse: 'tools/post-execute',
  stop: 'agent/turn-stopping',
  subagentStart: 'subagent/start',
  subagentStop: 'subagent/end',
}

const DEFAULT_TIMEOUT_MS = 60_000
const MAX_STOP_STEERS = 8
const STDERR_LIMIT = 500
/** The audit handlerId prefix: short and stable, the way the dialect names hooks. */
const HANDLER_PREFIX = 'loomux'

/**
 * The session's working directory, as the host exposes it — the same
 * fallback chain the sibling dsh-loomux plugin resolved against.
 *
 * @param {object} carrier an object holding the agent or session
 * @returns {string} the working directory, process.cwd() as the last resort
 */
function resolveCwd(carrier) {
  try {
    const session = carrier?.agent?.session ?? carrier?.session
    const cwd = session?.header?.cwd ?? session?.cwd ?? carrier?.cwd ?? carrier?.workspace?.cwd
    if (typeof cwd === 'string' && cwd.length > 0) return cwd
  } catch { /* a hostile getter is just an absent cwd */ }
  return process.cwd()
}

/**
 * Flatten a tool result into the plain text the hook protocol's
 * tool_response field carries; anything else is serialized as observed.
 *
 * @param {unknown} result the tool result the seam saw
 * @returns {string} the joined text blocks or the serialized result
 */
function toolResponse(result) {
  const content = result?.content
  if (Array.isArray(content)) {
    const texts = content
      .filter(block => block?.type === 'text' && typeof block.text === 'string')
      .map(block => block.text)
    if (texts.length > 0) return texts.join('\n')
  }
  try {
    return JSON.stringify(result ?? '')
  } catch {
    return String(result)
  }
}

/**
 * The hook file is bash-dialect, the way loomux writes it and Claude Code
 * runs it; the harness runs commands through cmd.exe, which leaves
 * `${VAR}` and `$(…)` untouched. Resolve the known placeholders here, and
 * leave them standing on hosts whose shell expands them itself.
 *
 * @param {string} command the command as the hook file holds it
 * @param {string} cwd the workspace directory of the payload
 * @returns {string} the command with every resolvable placeholder replaced
 */
function expandPlaceholders(command, cwd) {
  if (!command.includes('${LOCALAPPDATA}') && !command.includes('${CLAUDE_PROJECT_DIR}')) return command
  let out = command
  const localAppData = process.env.LOCALAPPDATA
  if (localAppData) out = out.replaceAll('${LOCALAPPDATA}', localAppData)
  if (cwd !== '') {
    out = out.replaceAll('$(basename "${CLAUDE_PROJECT_DIR}")', path.basename(cwd))
    out = out.replaceAll('${CLAUDE_PROJECT_DIR}', cwd)
  }
  return out
}

/**
 * Build the seam handlers. The dependencies are injected so the tests run
 * the real wiring against fakes; apply() passes the real ones. The host
 * process's own directory says nothing about the projects under way, so the
 * chain resolves lazily per session directory and caches what it found.
 *
 * @param {{ runHook: Function, resolveConfig: Function, getAgents?: () => ({ get(id: string): object|undefined }|undefined), logger?: object, config?: { configPath?: string, binary?: string, defaultTimeoutMs?: number } }} deps
 */
export function createWiring({ runHook: run, resolveConfig, getAgents, logger = {}, config = {} }) {
  const warn = (message) => logger.warn?.(`[${name}] ${message}`)

  /** Steer counts per agent run; the stop loop bound lives here. */
  const steers = new Map()

  /** Hook entries per session directory: by Claude event, in file order, with their file index. */
  const eventMaps = new Map()
  function byEventFor(cwd) {
    if (eventMaps.has(cwd)) return eventMaps.get(cwd)
    const resolved = resolveConfig(config.configPath, cwd, warn)
    const byEvent = new Map()
    const list0 = resolved?.entries ?? []
    for (let index = 0; index < list0.length; index++) {
      const hook = list0[index]
      const list = byEvent.get(hook.event) ?? []
      list.push({ index, hook })
      byEvent.set(hook.event, list)
    }
    eventMaps.set(cwd, byEvent)
    return byEvent
  }

  function audit(agent, type, data) {
    const emit = typeof agent?.session?.append === 'function'
      ? (t, d) => agent.session.append(t, d)
      : (t, d) => logger.info?.(t, d)
    try {
      emit(type, data)
    } catch { /* a dead log never breaks the hook path */ }
  }

  function timeoutOf(hook) {
    if (typeof hook.timeoutMs === 'number') return hook.timeoutMs
    return typeof config.defaultTimeoutMs === 'number' ? config.defaultTimeoutMs : DEFAULT_TIMEOUT_MS
  }

  function basePayload(seam, agent) {
    return {
      session_id: String(agent?.session?.id ?? ''),
      transcript_path: '',
      hook_event_name: SEAM_EVENTS[seam],
      cwd: resolveCwd(agent),
    }
  }

  async function runOne(seam, { index, hook }, subject, payload, turn, agent) {
    if (!new RegExp(hook.matcher).test(subject)) return { kind: 'none' }
    const point = SEAM_POINTS[seam]
    const handlerId = `${HANDLER_PREFIX}:${SEAM_EVENTS[seam]}:${index}`
    audit(agent, 'hook/invoked', { turn, point, dialect: 'claude', handlerId, matcher: hook.matcher })
    const started = Date.now()
    const result = await run(expandPlaceholders(hook.command, payload.cwd), payload, { cwd: payload.cwd, timeoutMs: timeoutOf(hook) })
    const decision = decide(seam, result)
    const data = { turn, point, handlerId, decision: decision.kind, durationMs: Date.now() - started, exitCode: result.exitCode }
    if (result.stderr !== '') data.stderrSummary = result.stderr.slice(0, STDERR_LIMIT)
    audit(agent, 'hook/result', data)
    return decision
  }

  /**
   * Run every matching entry of one seam: the first decisive answer wins,
   * context decisions accumulate; no entry may escape an error.
   */
  async function runEntries(seam, subject, payload, turn, agent) {
    const contexts = []
    let decision
    for (const item of byEventFor(payload.cwd).get(SEAM_EVENTS[seam]) ?? []) {
      const one = await runOne(seam, item, subject, payload, turn, agent)
      if (one.kind === 'context') contexts.push(one.context)
      else if (one.kind !== 'none' && decision === undefined) decision = one
    }
    return { decision: decision ?? { kind: 'none' }, contexts }
  }

  function injectText(agent, text) {
    agent?.inject?.({ content: [{ type: 'text', text }], source: { kind: 'user' } })
  }

  async function created(agent) {
    const { contexts } = await runEntries('sessionStart', '', basePayload('sessionStart', agent), 0, agent)
    for (const context of contexts) injectText(agent, context)
  }

  async function preTool(exec, next) {
    const payload = {
      ...basePayload('preToolUse', exec?.agent),
      tool_name: exec?.name,
      tool_input: exec?.arguments,
      tool_use_id: exec?.callId ?? '',
    }
    const { decision } = await runEntries('preToolUse', exec?.name ?? '', payload, exec?.turn ?? 0, exec?.agent)
    if (decision.kind === 'deny') return { kind: 'deny', reason: decision.reason ?? 'blocked by loomux hook' }
    if (decision.kind === 'ask') return { kind: 'ask' }
    return await next()
  }

  async function postTool(exec, result, next) {
    const payload = {
      ...basePayload('postToolUse', exec?.agent),
      tool_name: exec?.name,
      tool_input: exec?.arguments,
      tool_use_id: exec?.callId ?? '',
      tool_response: toolResponse(result),
    }
    const { decision, contexts } = await runEntries('postToolUse', exec?.name ?? '', payload, exec?.turn ?? 0, exec?.agent)
    if (decision.kind === 'block') {
      return { kind: 'block', feedback: [{ type: 'text', text: decision.reason ?? 'blocked by loomux hook' }] }
    }
    const outcome = await next(result)
    for (const context of contexts) injectText(exec?.agent, context)
    return outcome
  }

  async function stop(agent, turn) {
    const key = agent?.id
    const count = (steers.get(key) ?? 0) + 1
    if (key !== undefined) steers.set(key, count)
    const payload = { ...basePayload('stop', agent), stop_hook_active: count > 1 }
    const { decision } = await runEntries('stop', '', payload, turn ?? 0, agent)
    if (decision.kind === 'steer' && count <= MAX_STOP_STEERS) {
      agent?.steer?.({
        content: [{ type: 'text', text: decision.reason ?? 'loomux stop hook' }],
        source: { kind: 'user' },
      })
    }
  }

  async function subagentStart(info) {
    const agent = getAgents?.()?.get?.(info?.id)
    const payload = {
      ...basePayload('subagentStart', agent),
      agent_id: info?.id,
      agent_type: info?.provider ?? 'general-purpose',
    }
    const { contexts } = await runEntries('subagentStart', payload.agent_type, payload, 0, agent)
    for (const context of contexts) injectText(agent, context)
  }

  async function subagentEnd(info) {
    const agent = getAgents?.()?.get?.(info?.id)
    const payload = {
      ...basePayload('subagentStop', agent),
      agent_id: info?.id,
      agent_type: info?.provider ?? 'general-purpose',
      stop_hook_active: false,
    }
    await runEntries('subagentStop', payload.agent_type, payload, 0, agent)
  }

  function forget(agentId) {
    steers.delete(agentId)
  }

  return { created, preTool, postTool, stop, subagentStart, subagentEnd, forget }
}

/**
 * Register the seams on a host context. The seams are always registered —
 * only a session's own directory reveals whether it has hook entries, and
 * where the chain finds none the seams stay silent.
 *
 * @param {object} ctx the harness plugin context
 * @param {{ configPath?: string, binary?: string, defaultTimeoutMs?: number }} [config]
 * @param {{ runHook?: Function, resolveConfig?: Function }} [deps] test seams for the real entry points
 */
export function apply(ctx, config = {}, deps = {}) {
  const wiring = createWiring({
    runHook: deps.runHook ?? runHookImpl,
    resolveConfig: deps.resolveConfig ?? resolveChain,
    getAgents: () => {
      try {
        return ctx.agents ?? ctx.get?.('agents')
      } catch { /* a host that has not opened its agents yet has none for now */ }
      return undefined
    },
    logger: ctx.logger ?? {},
    config,
  })
  const disposers = []
  const on = (event, handler, opts) => {
    const dispose = ctx.on?.(event, handler, opts)
    if (typeof dispose === 'function') disposers.push(dispose)
  }
  on('agent/created', async ({ agent }) => wiring.created(agent), { global: true })
  on('tools/pre-execute', (exec, next) => wiring.preTool(exec, next))
  on('tools/post-execute', (exec, result, next) => wiring.postTool(exec, result, next))
  on('agent/turn-stopping', async ({ agent, turn }) => wiring.stop(agent, turn))
  on('subagent/start', async (info) => wiring.subagentStart(info))
  on('subagent/end', async (info) => wiring.subagentEnd(info))
  on('agent/disposed', ({ agent }) => wiring.forget(agent?.id))
  ctx.effect?.(() => () => {
    for (const dispose of disposers) {
      try {
        dispose()
      } catch { /* one disposer failing does not strand the rest */ }
    }
  })
}
