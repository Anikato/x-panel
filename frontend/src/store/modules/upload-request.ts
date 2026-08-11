export function buildUploadRequestHeaders(token: string, nodeID: number): Record<string, string> {
  const headers: Record<string, string> = { Authorization: token }
  if (nodeID > 0) headers['X-Node-ID'] = String(nodeID)
  return headers
}
