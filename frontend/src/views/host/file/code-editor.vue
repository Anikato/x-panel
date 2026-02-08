<template>
  <el-drawer
    v-model="visible"
    :title="title"
    size="80%"
    direction="rtl"
    :before-close="handleClose"
    destroy-on-close
    class="code-editor-drawer"
  >
    <template #header>
      <div class="editor-header">
        <span class="editor-title">{{ title }}</span>
        <div class="editor-actions">
          <el-select v-model="language" size="small" style="width: 130px" @change="updateLanguage">
            <el-option v-for="l in languages" :key="l.value" :label="l.label" :value="l.value" />
          </el-select>
          <el-select v-model="theme" size="small" style="width: 130px" @change="updateTheme">
            <el-option label="Dark" value="vs-dark" />
            <el-option label="Light" value="vs" />
            <el-option label="High Contrast" value="hc-black" />
          </el-select>
          <el-button size="small" @click="resetContent">{{ t('file.reset') }}</el-button>
          <el-button size="small" type="primary" :loading="saving" @click="saveContent">
            {{ t('file.saveFile') }} (Ctrl+S)
          </el-button>
        </div>
      </div>
    </template>
    <div ref="editorContainer" class="editor-container" />
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, onBeforeUnmount, nextTick, watch } from 'vue'
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getFileContent, saveFileContent } from '@/api/modules/file'

const { t } = useI18n()

// Setup Monaco Environment
self.MonacoEnvironment = {
  getWorker(_: any, label: string) {
    if (label === 'json') return new jsonWorker()
    if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker()
    if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker()
    if (label === 'typescript' || label === 'javascript') return new tsWorker()
    return new editorWorker()
  },
}

const emit = defineEmits(['saved'])
const visible = ref(false)
const title = ref('')
const filePath = ref('')
const saving = ref(false)
const originalContent = ref('')
const editorContainer = ref<HTMLElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null

const theme = ref('vs-dark')
const language = ref('plaintext')

const languages = [
  { label: 'Plain Text', value: 'plaintext' },
  { label: 'JavaScript', value: 'javascript' },
  { label: 'TypeScript', value: 'typescript' },
  { label: 'JSON', value: 'json' },
  { label: 'HTML', value: 'html' },
  { label: 'CSS', value: 'css' },
  { label: 'SCSS', value: 'scss' },
  { label: 'Python', value: 'python' },
  { label: 'Go', value: 'go' },
  { label: 'Rust', value: 'rust' },
  { label: 'Shell', value: 'shell' },
  { label: 'YAML', value: 'yaml' },
  { label: 'XML', value: 'xml' },
  { label: 'Markdown', value: 'markdown' },
  { label: 'SQL', value: 'sql' },
  { label: 'PHP', value: 'php' },
  { label: 'Java', value: 'java' },
  { label: 'C / C++', value: 'cpp' },
  { label: 'Dockerfile', value: 'dockerfile' },
  { label: 'Nginx', value: 'plaintext' },
  { label: 'INI / Conf', value: 'ini' },
  { label: 'Log', value: 'plaintext' },
]

const extLanguageMap: Record<string, string> = {
  js: 'javascript', mjs: 'javascript', cjs: 'javascript',
  ts: 'typescript', tsx: 'typescript',
  json: 'json', jsonc: 'json',
  html: 'html', htm: 'html',
  css: 'css', scss: 'scss', less: 'less',
  py: 'python',
  go: 'go',
  rs: 'rust',
  sh: 'shell', bash: 'shell', zsh: 'shell',
  yml: 'yaml', yaml: 'yaml',
  xml: 'xml', svg: 'xml',
  md: 'markdown',
  sql: 'sql',
  php: 'php',
  java: 'java',
  c: 'c', h: 'c', cpp: 'cpp', cc: 'cpp', cxx: 'cpp',
  dockerfile: 'dockerfile',
  ini: 'ini', conf: 'ini', cfg: 'ini', toml: 'ini',
  vue: 'html',
  rb: 'ruby',
  lua: 'lua',
  r: 'r',
}

function detectLanguage(filename: string): string {
  const lower = filename.toLowerCase()
  // Special filenames
  if (lower === 'dockerfile') return 'dockerfile'
  if (lower === 'makefile' || lower === 'gnumakefile') return 'plaintext'
  if (lower.endsWith('.env') || lower.startsWith('.env')) return 'ini'
  if (lower === 'nginx.conf' || lower.includes('nginx')) return 'plaintext'

  const ext = lower.split('.').pop() || ''
  return extLanguageMap[ext] || 'plaintext'
}

const open = async (path: string) => {
  filePath.value = path
  const name = path.split('/').pop() || path
  title.value = name
  language.value = detectLanguage(name)

  try {
    const res: any = await getFileContent({ path })
    originalContent.value = res.data?.content || ''
    visible.value = true
    await nextTick()
    setTimeout(initEditor, 100)
  } catch {
    ElMessage.error('Failed to load file')
  }
}

function initEditor() {
  if (!editorContainer.value) return
  if (editor) {
    editor.dispose()
    editor = null
  }

  editor = monaco.editor.create(editorContainer.value, {
    value: originalContent.value,
    language: language.value,
    theme: theme.value,
    fontSize: 14,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
    minimap: { enabled: true },
    wordWrap: 'on',
    scrollBeyondLastLine: false,
    automaticLayout: true,
    tabSize: 2,
    renderWhitespace: 'selection',
    lineNumbers: 'on',
    folding: true,
    bracketPairColorization: { enabled: true },
  })

  // Ctrl+S to save
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    saveContent()
  })
}

function updateLanguage(lang: string) {
  if (editor) {
    const model = editor.getModel()
    if (model) {
      monaco.editor.setModelLanguage(model, lang)
    }
  }
}

function updateTheme(t: string) {
  monaco.editor.setTheme(t)
}

function resetContent() {
  if (editor) {
    editor.setValue(originalContent.value)
  }
}

async function saveContent() {
  if (!editor) return
  saving.value = true
  try {
    await saveFileContent({ path: filePath.value, content: editor.getValue() })
    originalContent.value = editor.getValue()
    ElMessage.success(t('commons.success'))
    emit('saved')
  } catch { /* handled by interceptor */ } finally {
    saving.value = false
  }
}

function hasChanges(): boolean {
  if (!editor) return false
  return editor.getValue() !== originalContent.value
}

async function handleClose(done: () => void) {
  if (hasChanges()) {
    try {
      await ElMessageBox.confirm(t('file.unsavedChanges'), t('commons.tip'), { type: 'warning' })
      cleanup()
      done()
    } catch {
      // cancelled
    }
  } else {
    cleanup()
    done()
  }
}

function cleanup() {
  if (editor) {
    editor.dispose()
    editor = null
  }
}

onBeforeUnmount(() => cleanup())

defineExpose({ open })
</script>

<style lang="scss" scoped>
.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;

  .editor-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--xp-text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 300px;
  }

  .editor-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}

.editor-container {
  height: calc(100vh - 120px);
  border: 1px solid var(--xp-border-light);
  border-radius: 4px;
  overflow: hidden;
}
</style>
