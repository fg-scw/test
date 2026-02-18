export const isDefaultishValue = (value: string) => {
  const v = String(value ?? '').trim().toLowerCase()
  return (
    v === '' ||
    v === 'n/a' ||
    v === '-' ||
    v === '—' ||
    v === 'no' ||
    v === 'false' ||
    v === 'disabled'
  )
}

export const normalizeMappingRows = (
  entries: Array<{ source?: unknown; target?: unknown }>
): Array<{ source: string; target: string }> => {
  const normalizeTokens = (value: unknown): string[] => {
    if (!value) return []
    return String(value)
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
  }

  const rows: Array<{ source: string; target: string }> = []
  for (const entry of entries || []) {
    const sources = normalizeTokens(entry?.source)
    const targets = normalizeTokens(entry?.target)

    if (sources.length > 1 && targets.length > 1 && sources.length === targets.length) {
      for (let i = 0; i < sources.length; i += 1) {
        rows.push({ source: sources[i], target: targets[i] })
      }
      continue
    }

    const safeSources = sources.length ? sources : ['-']
    const safeTargets = targets.length ? targets : ['-']
    for (const source of safeSources) {
      for (const target of safeTargets) {
        rows.push({ source, target })
      }
    }
  }

  return rows
}
