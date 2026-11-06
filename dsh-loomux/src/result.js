/**
 * Shape a `loomux graph grep --json` answer into the tool's value.
 *
 * A group with a symbol carries symbol fields; a group without one is a file
 * hit and stays valid. The index state and the corpus counters belong in the
 * render text, not in the presentation metadata: the card shows a search, and
 * `presentationMeta` is what a replayed session reads back.
 */

const META_LINE_MAX_BYTES = 240
const MAX_META_BYTES = 32 * 1024

/**
 * Flatten the answer's groups into matches with optional symbol fields.
 *
 * @param {Record<string, unknown>} grepResult
 * @returns {Array<Record<string, unknown>>}
 */
export function toMatches(grepResult) {
  const groups = Array.isArray(grepResult?.groups) ? grepResult.groups : []
  const matches = []
  for (const group of groups) {
    const symbol = group?.symbol
    const hits = Array.isArray(group?.hits) ? group.hits : []
    for (const hit of hits) {
      const match = {
        path: String(group.path),
        lineNumber: hit.line,
        line: hit.text,
      }
      if (symbol && typeof symbol === 'object') {
        match.symbol = symbol.name
        match.kind = symbol.kind
        match.signature = symbol.signature
        match.inDegree = group.in_degree
      }
      matches.push(match)
    }
  }
  return matches
}

/**
 * One-line summary of what the corpus left out and which index state answered.
 */
export function describeIndex(grepResult) {
  const parts = []
  const index = typeof grepResult?.index === 'string' ? grepResult.index : 'none'
  const scope = Number(grepResult?.files_in_scope) || 0
  const candidates = Number(grepResult?.candidates) || 0
  if (index === 'used') {
    parts.push(`index used (${candidates} of ${scope} files in scope)`)
  } else if (index === 'scan') {
    parts.push(`scan (${scope} files in scope, index current but the pattern has no literal)`)
  } else if (index === 'stale') {
    parts.push(`scan (${index} index, ${scope} files in scope)`)
  } else {
    parts.push(`scan (no index, ${scope} files in scope)`)
  }
  const counters = []
  const ignored = Number(grepResult?.ignored) || 0
  const binary = Number(grepResult?.skipped_binary) || 0
  const size = Number(grepResult?.skipped_size) || 0
  const unreadable = Number(grepResult?.unreadable) || 0
  if (ignored > 0) counters.push(`${ignored} ignored by git`)
  if (binary > 0) counters.push(`${binary} binary`)
  if (size > 0) counters.push(`${size} over the size limit`)
  if (unreadable > 0) counters.push(`${unreadable} unreadable`)
  if (counters.length > 0) parts.push(`left out: ${counters.join(', ')}`)
  return parts.join('; ')
}

/**
 * Format the model-facing answer: matches with line numbers, then the index
 * sentence. Never drops a line silently.
 */
export function renderText(grepResult) {
  const matches = toMatches(grepResult)
  const lines = []
  if (matches.length === 0) {
    lines.push('(no matches)')
  } else {
    const files = new Set(matches.map(match => match.path))
    lines.push(`Found ${matches.length} matches across ${files.size} files for ${JSON.stringify(grepResult?.pattern ?? '')}:`)
    let path = ''
    for (const match of matches) {
      if (match.path !== path) {
        lines.push('')
        lines.push(`${match.path}:`)
        path = match.path
      }
      const symbol = match.symbol ? ` [${match.symbol}]` : ''
      lines.push(`  L${match.lineNumber}: ${match.line}${symbol}`)
    }
    const truncated = Number(grepResult?.truncated_hits) || 0
    if (truncated > 0) lines.push(`\n... and ${truncated} more hits truncated`)
  }
  lines.push('')
  lines.push(describeIndex(grepResult))
  return lines.join('\n')
}

/** Preview a line to the byte budget the session log retains. */
export function previewLine(line, maxBytes = META_LINE_MAX_BYTES) {
  if (typeof line !== 'string') return ''
  if (Buffer.byteLength(line, 'utf8') <= maxBytes) return line
  const ellipsis = '…'
  const budget = Math.max(0, maxBytes - Buffer.byteLength(ellipsis, 'utf8'))
  let out = ''
  let bytes = 0
  for (const char of line) {
    const size = Buffer.byteLength(char, 'utf8')
    if (bytes + size > budget) break
    out += char
    bytes += size
  }
  return `${out}${ellipsis}`
}

/** The `files` payload of the search card, grouped by path. */
export function groupMatchesByFile(matches) {
  const byFile = new Map()
  for (const match of matches) {
    let group = byFile.get(match.path)
    if (group === undefined) {
      group = []
      byFile.set(match.path, group)
    }
    group.push({ lineNumber: match.lineNumber, line: match.line })
  }
  return Array.from(byFile, ([path, fileMatches]) => ({ path, matches: fileMatches }))
}

function metaBytes(meta) {
  return Buffer.byteLength(JSON.stringify(meta), 'utf8')
}

/** Drop trailing file groups until the serialized metadata fits. */
export function capSearchMeta(meta, maxMetaBytes = MAX_META_BYTES) {
  if (metaBytes(meta) <= maxMetaBytes) return meta
  const files = [...meta.files]
  while (files.length > 1 && metaBytes({ ...meta, files, truncated: true }) > maxMetaBytes) files.pop()
  return { ...meta, files, truncated: true }
}

/**
 * Project the answer into the metadata the search card retains.
 *
 * @param {Record<string, unknown>} grepResult
 * @param {number} maxMatches
 */
export function presentationMeta(grepResult, maxMatches) {
  const matches = toMatches(grepResult)
  const retained = matches.slice(0, maxMatches).map(match => ({
    path: match.path,
    lineNumber: match.lineNumber,
    line: previewLine(match.line),
  }))
  return capSearchMeta({
    shape: 'matches',
    files: groupMatchesByFile(retained),
    truncated: matches.length > maxMatches,
    total: matches.length,
  })
}
