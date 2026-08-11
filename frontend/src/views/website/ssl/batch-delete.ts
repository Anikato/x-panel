export interface CertificateDeleteIssue {
  id: number
  domain: string
  reason: string
}

export interface CertificateDeleteResult {
  deletedCount: number
  skipped: CertificateDeleteIssue[]
  failed: CertificateDeleteIssue[]
}

export function formatCertificateDeleteSummary(result: CertificateDeleteResult): string {
  return `已删除 ${result.deletedCount} 张，跳过 ${result.skipped.length} 张，失败 ${result.failed.length} 张`
}

export function pageAfterCertificateDelete(
  currentPage: number,
  currentPageItemCount: number,
  deletedCount: number,
): number {
  if (currentPage > 1 && currentPageItemCount > 0 && deletedCount >= currentPageItemCount) {
    return currentPage - 1
  }
  return currentPage
}
