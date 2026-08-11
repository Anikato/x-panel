export const MAX_UPLOAD_FILES = 1000

export interface UploadCandidate {
  file: File
  relativePath: string
}

export interface DroppedEntry {
  name: string
  isFile: boolean
  isDirectory: boolean
  file?: (success: (file: File) => void, failure?: (error: DOMException) => void) => void
  createReader?: () => DroppedEntryReader
}

interface DroppedEntryReader {
  readEntries: (
    success: (entries: DroppedEntry[]) => void,
    failure?: (error: DOMException) => void,
  ) => void
}

interface DroppedItem {
  kind: string
  getAsEntry?: () => DroppedEntry | null
  webkitGetAsEntry?: () => DroppedEntry | null
}

interface DroppedData {
  items: ArrayLike<DroppedItem>
  files: ArrayLike<File>
}

function normalizeUploadPath(value: string): string {
  if (/^[\\/]/.test(value)) throw new Error(`非法上传路径：${value}`)
  const parts = value.replace(/\\/g, '/').split('/').filter((part) => part && part !== '.')
  if (parts.length === 0 || parts.some((part) => part === '..' || part.includes('\0') || /[\r\n]/.test(part))) {
    throw new Error(`非法上传路径：${value}`)
  }
  return parts.join('/')
}

function finalize(candidates: UploadCandidate[]): UploadCandidate[] {
  if (candidates.length > MAX_UPLOAD_FILES) {
    throw new Error(`单次最多上传 ${MAX_UPLOAD_FILES} 个文件`)
  }

  const seen = new Set<string>()
  return candidates.map((candidate) => {
    const relativePath = normalizeUploadPath(candidate.relativePath)
    if (seen.has(relativePath)) throw new Error(`存在重复路径：${relativePath}`)
    seen.add(relativePath)
    return { ...candidate, relativePath }
  })
}

export function collectSelectedCandidates(files: ArrayLike<File>): UploadCandidate[] {
  return finalize(Array.from(files, (file) => ({
    file,
    relativePath: file.webkitRelativePath || file.name,
  })))
}

export function excludeConflicts(candidates: UploadCandidate[], conflicts: string[]): UploadCandidate[] {
  const conflictPaths = new Set(conflicts)
  return candidates.filter((candidate) => !conflictPaths.has(candidate.relativePath))
}

function readFile(entry: DroppedEntry): Promise<File> {
  if (!entry.file) return Promise.reject(new Error(`无法读取文件：${entry.name}`))
  return new Promise((resolve, reject) => entry.file?.(resolve, reject))
}

function readEntries(reader: DroppedEntryReader): Promise<DroppedEntry[]> {
  return new Promise((resolve, reject) => reader.readEntries(resolve, reject))
}

async function walk(entry: DroppedEntry, parent: string, result: UploadCandidate[]): Promise<void> {
  const relativePath = parent ? `${parent}/${entry.name}` : entry.name
  if (entry.isFile) {
    result.push({ file: await readFile(entry), relativePath })
    if (result.length > MAX_UPLOAD_FILES) {
      throw new Error(`单次最多上传 ${MAX_UPLOAD_FILES} 个文件`)
    }
    return
  }
  if (!entry.isDirectory || !entry.createReader) return

  const reader = entry.createReader()
  while (true) {
    const entries = await readEntries(reader)
    if (entries.length === 0) return
    for (const child of entries) await walk(child, relativePath, result)
  }
}

export async function collectDroppedCandidates(data: DroppedData): Promise<UploadCandidate[]> {
  const entries = Array.from(data.items)
    .filter((item) => item.kind === 'file')
    .map((item) => item.getAsEntry?.() ?? item.webkitGetAsEntry?.())
    .filter((entry): entry is DroppedEntry => !!entry)

  if (entries.length === 0) return collectSelectedCandidates(data.files)

  const result: UploadCandidate[] = []
  for (const entry of entries) await walk(entry, '', result)
  return finalize(result)
}
