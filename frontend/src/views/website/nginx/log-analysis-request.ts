export type AnalysisRequestState = {
  seq: number
  controller: AbortController | null
}

export function createAnalysisRequestState(): AnalysisRequestState {
  return { seq: 0, controller: null }
}

export function beginAnalysisRequest(state: AnalysisRequestState): { seq: number; signal: AbortSignal } {
  state.controller?.abort()
  state.seq += 1
  state.controller = new AbortController()
  return { seq: state.seq, signal: state.controller.signal }
}

export function isCurrentAnalysisRequest(state: AnalysisRequestState, seq: number): boolean {
  return seq === state.seq
}

export function abortAnalysisRequest(state: AnalysisRequestState) {
  state.controller?.abort()
  state.controller = null
}

export type AnalysisViewKind = 'loading' | 'error' | 'updating' | 'stale' | 'ready' | 'empty'

export function analysisViewState(opts: {
  hasData: boolean
  loading: boolean
  error: string | null
}): AnalysisViewKind {
  if (opts.loading && opts.hasData) return 'updating'
  if (opts.loading && !opts.hasData) return 'loading'
  if (opts.error && opts.hasData) return 'stale'
  if (opts.error && !opts.hasData) return 'error'
  if (opts.hasData) return 'ready'
  return 'empty'
}
