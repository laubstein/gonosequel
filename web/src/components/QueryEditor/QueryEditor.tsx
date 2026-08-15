import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import { autocompletion } from '@codemirror/autocomplete'
import type { EditorView } from '@codemirror/view'
import styles from './QueryEditor.module.css'
import type { ExtJSONDocument, FindQuery } from '../../types'
import { exportURL } from '../../api/http'
import { api } from '../../api/client'
import { useCollectionSchema } from '../../hooks/useCollectionSchema'
import { useConnectionInfo } from '../../hooks/useConnectionInfo'
import { useIsDarkMode } from '../../hooks/useIsDarkMode'
import { fieldCompletionSource } from './fieldCompletion'
import { buildPresets } from './presets'
import type { Preset } from '../../types'

type Mode = 'find' | 'aggregate' | 'update'

// containsCollscan walks an explain result looking for a COLLSCAN stage
// anywhere in the plan tree — recursing into every object/array rather
// than only checking queryPlanner.winningPlan.stage, since the stage can
// be nested under inputStage (e.g. behind a SORT or PROJECTION stage) or,
// on a sharded cluster, under a per-shard entry instead of at the top level.
function containsCollscan(value: unknown): boolean {
  if (Array.isArray(value)) return value.some(containsCollscan)
  if (value && typeof value === 'object') {
    const obj = value as Record<string, unknown>
    if (obj.stage === 'COLLSCAN') return true
    return Object.values(obj).some(containsCollscan)
  }
  return false
}

interface Props {
  db: string
  coll: string
  query: FindQuery
  onRun: (filter: string, sort: string) => void
  onNewDocument: () => void
  onAggregateResult: (documents: ExtJSONDocument[] | null) => void
}

export function QueryEditor({ db, coll, query, onRun, onNewDocument, onAggregateResult }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [mode, setMode] = useState<Mode>('find')
  const [filterText, setFilterText] = useState(query.filter ?? '{}')
  const [sortText, setSortText] = useState(query.sort ?? '')
  const [pipelineText, setPipelineText] = useState('[]')
  const [updateText, setUpdateText] = useState('{\n  "$set": {}\n}')
  const [error, setError] = useState<string | null>(null)
  const [explainResult, setExplainResult] = useState<string | null>(null)
  const [explainCollscan, setExplainCollscan] = useState(false)
  const [explaining, setExplaining] = useState(false)
  const [aggregating, setAggregating] = useState(false)
  const [updating, setUpdating] = useState(false)
  const [updateResult, setUpdateResult] = useState<{ matched: number; modified: number } | null>(null)
  const [presetIndex, setPresetIndex] = useState('')

  // QueryEditor only renders for Mongo-shaped connections now — App.tsx
  // swaps in RedisCommandRunner entirely for redis/valkey (see its own
  // doc comment), so capability gating here is just for Explain/Aggregate/
  // UpdateMany specifically, not a driver check.
  const { data: connection } = useConnectionInfo()
  const canAggregate = connection?.capabilities.includes('aggregate') ?? true
  const canExplain = connection?.capabilities.includes('explain') ?? true
  const canUpdateMany = connection?.capabilities.includes('updateMany') ?? true

  const { data: schemaFields } = useCollectionSchema(db, coll)
  const extensions = [json(), autocompletion({ override: [fieldCompletionSource(schemaFields ?? [])] })]
  const isDark = useIsDarkMode()

  const presets = buildPresets(schemaFields ?? [])

  function applyPreset(index: string) {
    setPresetIndex(index)
    if (index === '') return
    const preset: Preset | undefined = presets[Number(index)]
    if (!preset) return

    setMode(preset.mode)
    setError(null)
    setExplainResult(null)
    setExplainCollscan(false)
    setUpdateResult(null)
    if (preset.mode === 'find') {
      setFilterText(preset.filter ?? '{}')
      setSortText(preset.sort ?? '')
    } else if (preset.mode === 'update') {
      setFilterText(preset.filter ?? '{}')
      setUpdateText(preset.update ?? '{\n  "$set": {}\n}')
    } else {
      setPipelineText(preset.pipeline ?? '[]')
    }
  }

  function switchMode(next: Mode) {
    setMode(next)
    setError(null)
    setExplainResult(null)
    setExplainCollscan(false)
    setUpdateResult(null)
    if (next === 'find') onAggregateResult(null)
  }

  // If the user has a non-empty text selection in the active query/pipeline
  // editor, running only executes the selected text instead of the whole
  // box — same convention as RedisCommandRunner's command runner.
  // queryViewRef points at whichever CodeMirror instance (filter or
  // pipeline) is currently mounted for find/aggregate mode; updateViewRef
  // is separate because update mode shows the filter box *and* the update
  // document box at once, so there are two live editors to track instead
  // of one.
  const queryViewRef = useRef<EditorView | null>(null)
  const updateViewRef = useRef<EditorView | null>(null)
  function textFromView(view: EditorView | null, fallback: string): string {
    const sel = view?.state.selection.main
    if (sel && !sel.empty) return view!.state.sliceDoc(sel.from, sel.to)
    return fallback
  }
  function textToRun(fallback: string): string {
    return textFromView(queryViewRef.current, fallback)
  }

  function run() {
    if (mode === 'aggregate') {
      void runAggregate()
      return
    }
    if (mode === 'update') {
      void runUpdateMany()
      return
    }
    const filter = textToRun(filterText)
    try {
      if (filter.trim()) JSON.parse(filter)
      if (sortText.trim()) JSON.parse(sortText)
      setError(null)
      setExplainResult(null)
    setExplainCollscan(false)
      onAggregateResult(null)
      onRun(filter.trim() || '{}', sortText.trim())
    } catch (e) {
      setError(e instanceof Error ? e.message : t('queryEditor.invalidJson'))
    }
  }

  async function runAggregate() {
    const pipeline = textToRun(pipelineText)
    try {
      JSON.parse(pipeline)
      setError(null)
      setAggregating(true)
      const result = await api.aggregate(db, coll, pipeline)
      onAggregateResult(result.documents)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('queryEditor.invalidJson'))
    } finally {
      setAggregating(false)
    }
  }

  async function runUpdateMany() {
    const filter = textToRun(filterText)
    const update = textFromView(updateViewRef.current, updateText)
    try {
      JSON.parse(filter.trim() || '{}')
      JSON.parse(update)
      setError(null)
      setUpdateResult(null)
      setUpdating(true)
      const result = await api.updateMany(db, coll, filter.trim() || '{}', update)
      setUpdateResult(result)
      // The document table (Results/Pagination) fetches through
      // useDocuments's ['documents', db, coll, query] key — a bulk write
      // outside that hook's own query/mutation flow needs an explicit
      // invalidation, same as DocumentEditor/RedisValueEditor do after a
      // single-document save.
      void queryClient.invalidateQueries({ queryKey: ['documents', db, coll] })
    } catch (e) {
      setError(e instanceof Error ? e.message : t('queryEditor.invalidJson'))
    } finally {
      setUpdating(false)
    }
  }

  async function explain() {
    const filter = textToRun(filterText)
    try {
      if (filter.trim()) JSON.parse(filter)
      setError(null)
      setExplaining(true)
      const result = await api.explain(db, coll, filter.trim() || '{}')
      setExplainResult(JSON.stringify(result, null, 2))
      setExplainCollscan(containsCollscan(result))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('queryEditor.invalidJson'))
    } finally {
      setExplaining(false)
    }
  }

  // A native, window-level capture-phase listener — not a CodeMirror
  // keymap extension, and not a React onKeyDownCapture prop either.
  // CodeMirror 6 installs its own native keydown handler directly on its
  // content DOM node, which fires and calls stopPropagation before a
  // React synthetic capture handler (attached at the React root, not
  // document) ever sees the event; a keymap.of([...]) extension has the
  // same problem competing against autocomplete's/basicSetup's own
  // keymaps for facet precedence. Capturing at `window` runs before both.
  const wrapperRef = useRef<HTMLDivElement>(null)
  const runRef = useRef(run)
  runRef.current = run

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (!(e.ctrlKey || e.metaKey) || e.key !== 'Enter') return
      if (!wrapperRef.current?.contains(e.target as Node)) return
      e.preventDefault()
      e.stopPropagation()
      runRef.current()
    }
    window.addEventListener('keydown', onKeyDown, true)
    return () => window.removeEventListener('keydown', onKeyDown, true)
  }, [])

  const running = mode === 'aggregate' ? aggregating : mode === 'update' ? updating : false

  return (
    <div className={styles.editor} ref={wrapperRef}>
      <div className={styles.row}>
        <select
          className={styles.presetSelect}
          value={presetIndex}
          onChange={(e) => applyPreset(e.target.value)}
        >
          <option value="">{t('queryEditor.presetPlaceholder')}</option>
          {presets.map((preset, i) => (
            <option key={i} value={i}>
              {t(preset.labelKey, preset.labelParams)}
            </option>
          ))}
        </select>
      </div>
      <div className={styles.row}>
        <button
          className={mode === 'find' ? styles.modeButtonActive : styles.modeButton}
          onClick={() => switchMode('find')}
        >
          {t('queryEditor.modeFind')}
        </button>
        {canAggregate && (
          <button
            className={mode === 'aggregate' ? styles.modeButtonActive : styles.modeButton}
            onClick={() => switchMode('aggregate')}
          >
            {t('queryEditor.modeAggregate')}
          </button>
        )}
        {canUpdateMany && (
          <button
            className={mode === 'update' ? styles.modeButtonActive : styles.modeButton}
            onClick={() => switchMode('update')}
          >
            {t('queryEditor.modeUpdate')}
          </button>
        )}
      </div>

      {mode === 'aggregate' ? (
        <CodeMirror
          value={pipelineText}
          height="120px"
          extensions={extensions}
          theme={isDark ? 'dark' : 'light'}
          onChange={(value) => setPipelineText(value)}
          onCreateEditor={(view) => {
            queryViewRef.current = view
          }}
          placeholder={t('queryEditor.pipelinePlaceholder')}
          basicSetup={{ lineNumbers: false, foldGutter: false }}
        />
      ) : (
        <CodeMirror
          value={filterText}
          height="80px"
          extensions={extensions}
          theme={isDark ? 'dark' : 'light'}
          onChange={(value) => setFilterText(value)}
          onCreateEditor={(view) => {
            queryViewRef.current = view
          }}
          placeholder={t('queryEditor.filterPlaceholder')}
          basicSetup={{ lineNumbers: false, foldGutter: false }}
        />
      )}

      {mode === 'update' && (
        <CodeMirror
          value={updateText}
          height="80px"
          extensions={extensions}
          theme={isDark ? 'dark' : 'light'}
          onChange={(value) => setUpdateText(value)}
          onCreateEditor={(view) => {
            updateViewRef.current = view
          }}
          placeholder={t('queryEditor.updatePlaceholder')}
          basicSetup={{ lineNumbers: false, foldGutter: false }}
        />
      )}

      {mode === 'find' && (
        <div className={styles.row}>
          <input
            className={styles.textarea}
            style={{ minHeight: 'unset', flex: 1 }}
            value={sortText}
            onChange={(e) => setSortText(e.target.value)}
            placeholder={t('queryEditor.sortPlaceholder')}
          />
        </div>
      )}

      <div className={styles.row}>
        <button className={styles.button} onClick={run} disabled={running}>
          {running ? (mode === 'update' ? t('queryEditor.updating') : t('queryEditor.aggregating')) : t('queryEditor.run')}
        </button>
        {mode === 'find' && canExplain && (
          <button className={styles.button} onClick={() => void explain()} disabled={explaining}>
            {explaining ? t('queryEditor.explaining') : t('queryEditor.explain')}
          </button>
        )}
        {mode !== 'update' && (
          <button className={styles.button} onClick={onNewDocument}>
            {t('queryEditor.newDocument')}
          </button>
        )}
        {error && <span className={styles.error}>{error}</span>}
        <div className={styles.spacer} />
        {mode === 'find' && (
          <>
            <a className={styles.exportLink} href={exportURL(db, coll, 'json', { filter: filterText })} download>
              {t('queryEditor.exportJson')}
            </a>
            <a className={styles.exportLink} href={exportURL(db, coll, 'csv', { filter: filterText })} download>
              {t('queryEditor.exportCsv')}
            </a>
          </>
        )}
      </div>
      {explainResult && (
        <>
          {explainCollscan && <div className={styles.collscanWarning}>{t('queryEditor.collscanWarning')}</div>}
          <pre className={styles.explainOutput}>{explainResult}</pre>
        </>
      )}
      {updateResult && (
        <div className={styles.explainOutput}>
          {t('queryEditor.updateResult', { matched: updateResult.matched, modified: updateResult.modified })}
        </div>
      )}
    </div>
  )
}
