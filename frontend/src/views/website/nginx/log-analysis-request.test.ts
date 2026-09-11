import assert from 'node:assert/strict'
import test from 'node:test'
import {
  abortAnalysisRequest,
  analysisViewState,
  beginAnalysisRequest,
  createAnalysisRequestState,
  isCurrentAnalysisRequest,
} from './log-analysis-request.ts'

test('later request keeps loading even if earlier request finishes', () => {
  const state = createAnalysisRequestState()
  const first = beginAnalysisRequest(state)
  const second = beginAnalysisRequest(state)
  assert.equal(isCurrentAnalysisRequest(state, first.seq), false)
  assert.equal(isCurrentAnalysisRequest(state, second.seq), true)
  assert.equal(first.signal.aborted, true)
  assert.equal(second.signal.aborted, false)
})

test('first failure is an error, refresh failure keeps stale data', () => {
  assert.equal(analysisViewState({ hasData: false, loading: false, error: 'failed' }), 'error')
  assert.equal(analysisViewState({ hasData: true, loading: false, error: 'failed' }), 'stale')
  assert.equal(analysisViewState({ hasData: true, loading: true, error: null }), 'updating')
  assert.equal(analysisViewState({ hasData: false, loading: true, error: null }), 'loading')
})

test('unmount aborts the in-flight request', () => {
  const state = createAnalysisRequestState()
  const req = beginAnalysisRequest(state)
  abortAnalysisRequest(state)
  assert.equal(req.signal.aborted, true)
})
