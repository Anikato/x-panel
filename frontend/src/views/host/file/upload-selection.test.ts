import assert from 'node:assert/strict'
import test from 'node:test'

import {
  MAX_UPLOAD_FILES,
  collectDroppedCandidates,
  collectSelectedCandidates,
  excludeConflicts,
  type DroppedEntry,
} from './upload-selection.ts'

function makeFile(name: string, relativePath = ''): File {
  const result = new File(['data'], name)
  Object.defineProperty(result, 'webkitRelativePath', { value: relativePath })
  return result
}

function fileEntry(value: File): DroppedEntry {
  return {
    name: value.name,
    isFile: true,
    isDirectory: false,
    file: (success) => success(value),
  }
}

function directoryEntry(name: string, batches: DroppedEntry[][]): DroppedEntry {
  let readerCreated = false
  return {
    name,
    isFile: false,
    isDirectory: true,
    createReader: () => {
      assert.equal(readerCreated, false, 'a directory reader must be reused until it is exhausted')
      readerCreated = true
      return {
        readEntries: (success) => success(batches.shift() ?? []),
      }
    },
  }
}

test('collectSelectedCandidates preserves folder relative paths', () => {
  const result = collectSelectedCandidates([makeFile('app.js', 'site/assets/app.js')])
  assert.equal(result[0].relativePath, 'site/assets/app.js')
})

test('collectDroppedCandidates reads every directory batch', async () => {
  const root = directoryEntry('site', [
    [fileEntry(makeFile('index.html'))],
    [directoryEntry('assets', [[fileEntry(makeFile('app.js'))], []])],
    [],
  ])

  const result = await collectDroppedCandidates({
    items: [{ kind: 'file', webkitGetAsEntry: () => root }],
    files: [],
  })

  assert.deepEqual(result.map((item) => item.relativePath), [
    'site/index.html',
    'site/assets/app.js',
  ])
})

test('collectSelectedCandidates rejects duplicate destination paths', () => {
  assert.throws(
    () => collectSelectedCandidates([makeFile('a.txt'), makeFile('a.txt')]),
    /重复路径/,
  )
})

test('collectSelectedCandidates rejects more than the upload limit', () => {
  const files = Array.from({ length: MAX_UPLOAD_FILES + 1 }, (_, index) => makeFile(`${index}.txt`))
  assert.throws(() => collectSelectedCandidates(files), /最多上传/)
})

test('excludeConflicts matches complete relative paths', () => {
  const candidates = collectSelectedCandidates([
    makeFile('config.json', 'a/config.json'),
    makeFile('config.json', 'b/config.json'),
  ])
  const result = excludeConflicts(candidates, ['a/config.json'])
  assert.deepEqual(result.map((item) => item.relativePath), ['b/config.json'])
})
