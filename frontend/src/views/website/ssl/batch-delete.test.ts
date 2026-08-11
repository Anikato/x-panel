import assert from 'node:assert/strict'
import test from 'node:test'

import {
  formatCertificateDeleteSummary,
  pageAfterCertificateDelete,
  type CertificateDeleteResult,
} from './batch-delete.ts'

test('formatCertificateDeleteSummary reports deleted skipped and failed counts', () => {
  const result: CertificateDeleteResult = {
    deletedCount: 18,
    skipped: [
      { id: 1, domain: 'a.example.com', reason: 'used' },
      { id: 2, domain: 'b.example.com', reason: 'used' },
      { id: 3, domain: 'c.example.com', reason: 'used' },
    ],
    failed: [{ id: 4, domain: 'd.example.com', reason: 'failed' }],
  }
  assert.equal(formatCertificateDeleteSummary(result), '已删除 18 张，跳过 3 张，失败 1 张')
})

test('pageAfterCertificateDelete backs up only when the current page becomes empty', () => {
  assert.equal(pageAfterCertificateDelete(3, 20, 20), 2)
  assert.equal(pageAfterCertificateDelete(3, 20, 5), 3)
  assert.equal(pageAfterCertificateDelete(1, 20, 20), 1)
})
