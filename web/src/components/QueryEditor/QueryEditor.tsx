import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import { autocompletion } from '@codemirror/autocomplete'
import type { EditorView } from '@codemirror/view'
import styles from './QueryEditor.module.css'
import type { ExtJSONDocument, FindQuery } from '../../types'
import { startExportDownload } from '../../api/http'
import { api } from '../../api/client'
import { computeJsonFix } from '../../api/jsonFix'
import { readLocal, writeLocal } from '../../api/localCache'
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
  replayNonce: number
  onRun: (filter: string, sort: string, projection?: string) => void
  onNewDocument: () => void
  onAggregateResult: (documents: ExtJSONDocument[] | null) => void
  // Set by App.tsx when the Results table's right-click context menu picks
  // a value to filter by, or a field to hide — the only way to reach this
  // component's own draft state from a sibling (Results), since App.tsx is
  // their common parent. Mirrors replayNonce's "bump a counter to signal a
  // new external value, even if the payload could look unchanged" pattern.
  externalDraftPatch: { filterText?: string; hideField?: string } | null
  externalDraftNonce: number
}

interface DraftState {
  mode: Mode
  filterText: string
  sortText: string
  pipelineText: string
  updateText: string
  // Raw MongoDB projection, typed directly in the form or built up by the
  // Results table's "Hide field" context-menu action (which parses this
  // same text and adds "field": 0 to it — see the externalDraftPatch
  // effect below). Sent as-is when running (see run()). Only meaningful
  // in 'find' mode.
  projectionText: string
}

function defaultDraft(query: FindQuery): DraftState {
  return {
    mode: 'find',
    filterText: query.filter ?? '{}',
    sortText: query.sort ?? '',
    pipelineText: '[]',
    updateText: '{\n  "$set": {}\n}',
    projectionText: '',
  }
}

// parseProjectionFields returns the field names an exclusion projection
// (produced by "Hide field", or hand-typed the same way) hides — used to
// render the removable chip list. Returns [] for anything that isn't a
// plain flat object of 0/false values, including an inclusion-style
// projection ({field: 1}), which has no "hidden fields" to list this way.
// normalizeJsonField parses text as strict JSON; if that fails but it
// parses as a JS-object-literal (JSON5), returns the fixed strict-JSON
// string instead of throwing — used to silently accept `{cpu: 1}`-style
// input in Sort/Projection on Run, without requiring the manual "Fix
// JSON" button used by filter/pipeline/update.
function normalizeJsonField(text: string): string {
  const trimmed = text.trim()
  if (!trimmed) return ''
  try {
    JSON.parse(trimmed)
    return trimmed
  } catch {
    const fixed = computeJsonFix(trimmed)
    if (typeof fixed === 'string') return fixed
    throw new SyntaxError(trimmed)
  }
}

function parseProjectionFields(projectionText: string): string[] {
  if (!projectionText.trim()) return []
  try {
    const obj = JSON.parse(projectionText) as Record<string, unknown>
    if (!obj || typeof obj !== 'object' || Array.isArray(obj)) return []
    return Object.entries(obj)
      .filter(([, v]) => v === 0 || v === false)
      .map(([k]) => k)
  } catch {
    return []
  }
}

// draftCache persists each collection's in-progress query/pipeline/update
// text across collection switches and tab changes — this component used to
// be force-remounted via a `key` tied to the selected collection, which
// reset the editor to defaults every time and lost whatever was typed when
// navigating back to a previously-visited collection. Keyed by "db:coll",
// backed by localStorage (with this in-memory Map as a read cache in front
// of it) so the same draft also survives a full page refresh and — since
// it isn't tied to a session ID — reconnecting and revisiting the same
// collection later.
const draftCache = new Map<string, DraftState>()

function draftKey(db: string, coll: string): string {
  return `${db}:${coll}`
}

function draftStorageKey(key: string): string {
  return `gonosequel.queryDraft:${key}`
}

function loadDraft(key: string, fallback: () => DraftState): DraftState {
  const cached = draftCache.get(key)
  if (cached) return cached
  const draft = readLocal<DraftState>(draftStorageKey(key)) ?? fallback()
  draftCache.set(key, draft)
  return draft
}

function saveDraft(key: string, next: DraftState) {
  draftCache.set(key, next)
  writeLocal(draftStorageKey(key), next)
}

export function QueryEditor({
  db,
  coll,
  query,
  replayNonce,
  onRun,
  onNewDocument,
  onAggregateResult,
  externalDraftPatch,
  externalDraftNonce,
}: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const key = draftKey(db, coll)
  const [draft, setDraft] = useState<DraftState>(() => loadDraft(key, () => defaultDraft(query)))
  const { mode, filterText, sortText, pipelineText, updateText, projectionText } = draft

  function patchDraft(patch: Partial<DraftState>) {
    setDraft((prev) => {
      const next = { ...prev, ...patch }
      saveDraft(key, next)
      return next
    })
  }

  function setMode(next: Mode) {
    patchDraft({ mode: next })
  }
  function setFilterText(next: string) {
    patchDraft({ filterText: next })
  }
  function setSortText(next: string) {
    patchDraft({ sortText: next })
  }
  function setProjectionText(next: string) {
    patchDraft({ projectionText: next })
  }
  function setPipelineText(next: string) {
    patchDraft({ pipelineText: next })
  }
  function setUpdateText(next: string) {
    patchDraft({ updateText: next })
  }

  // Reload the draft cached for this collection whenever the selected
  // collection changes, and force-overwrite it with the incoming query when
  // a history entry is replayed (even if that replay targets the
  // currently-selected collection, in which case `key` alone wouldn't
  // change) — replay takes priority when both happen in the same commit.
  const prevKeyRef = useRef(key)
  const prevReplayRef = useRef(replayNonce)
  useEffect(() => {
    const keyChanged = prevKeyRef.current !== key
    const replayed = prevReplayRef.current !== replayNonce
    prevKeyRef.current = key
    prevReplayRef.current = replayNonce
    if (!keyChanged && !replayed) return

    if (replayed) {
      const existing = loadDraft(key, () => defaultDraft(query))
      const next: DraftState = {
        mode: 'find',
        filterText: query.filter ?? '{}',
        sortText: query.sort ?? '',
        pipelineText: existing.pipelineText,
        updateText: existing.updateText,
        projectionText: '',
      }
      saveDraft(key, next)
      setDraft(next)
    } else {
      setDraft(loadDraft(key, () => defaultDraft(query)))
    }
    setError(null)
    setExplainResult(null)
    setExplainCollscan(false)
    setUpdateResult(null)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, replayNonce])

  // Applies a "Filter by value" or "Hide field" pick made in the Results
  // table's context menu (see the Props doc comment for externalDraftPatch).
  useEffect(() => {
    if (externalDraftNonce === 0 || !externalDraftPatch) return
    if (externalDraftPatch.filterText !== undefined) {
      patchDraft({ mode: 'find', filterText: externalDraftPatch.filterText })
    }
    if (externalDraftPatch.hideField !== undefined) {
      const field = externalDraftPatch.hideField
      setDraft((prev) => {
        // Parse-and-patch rather than tracking a separate field list, so
        // the context menu and the form's own Projection input stay one
        // single source of truth. An unparseable or non-object existing
        // value (or an inclusion-style {field: 1}) is replaced outright —
        // "Hide field" always wins over whatever was there, rather than
        // silently failing to add the exclusion.
        let obj: Record<string, unknown> = {}
        try {
          const parsed: unknown = prev.projectionText.trim() ? JSON.parse(prev.projectionText) : {}
          if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) obj = parsed as Record<string, unknown>
        } catch {
          // fall through with obj = {}
        }
        if (obj[field] === 0 || obj[field] === false) return prev
        const nextProjection = JSON.stringify({ ...obj, [field]: 0 }, null, 2)
        const next = { ...prev, projectionText: nextProjection }
        saveDraft(key, next)
        return next
      })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [externalDraftNonce])

  const [error, setError] = useState<string | null>(null)
  const [explainResult, setExplainResult] = useState<string | null>(null)
  const [explainCollscan, setExplainCollscan] = useState(false)
  const [explaining, setExplaining] = useState(false)
  const [aggregating, setAggregating] = useState(false)
  const [updating, setUpdating] = useState(false)
  const [exporting, setExporting] = useState(false)
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

  // Same "Fix JSON" treatment as DocumentEditor: catches the common case of
  // typing a JS object literal (unquoted keys, single quotes) into a box
  // that only accepts strict JSON. Computed per box since find/update mode
  // show more than one at once, but a single button covers whichever
  // box(es) are active and fixable in the current mode.
  const filterFix = useMemo(() => computeJsonFix(filterText), [filterText])
  const pipelineFix = useMemo(() => computeJsonFix(pipelineText), [pipelineText])
  const updateFix = useMemo(() => computeJsonFix(updateText), [updateText])
  const canFix =
    mode === 'find'
      ? typeof filterFix === 'string'
      : mode === 'aggregate'
        ? typeof pipelineFix === 'string'
        : typeof filterFix === 'string' || typeof updateFix === 'string'

  function applyFix() {
    if (mode === 'find') {
      if (typeof filterFix === 'string') setFilterText(filterFix)
    } else if (mode === 'aggregate') {
      if (typeof pipelineFix === 'string') setPipelineText(pipelineFix)
    } else {
      if (typeof filterFix === 'string') setFilterText(filterFix)
      if (typeof updateFix === 'string') setUpdateText(updateFix)
    }
  }

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
    // A preset describes the contents of one specific mode's editor;
    // leaving it selected after a mode switch labels text it no longer
    // has anything to do with.
    setPresetIndex('')
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
      const sort = normalizeJsonField(sortText)
      const projection = normalizeJsonField(projectionText)
      if (sort !== sortText.trim()) setSortText(sort)
      if (projection !== projectionText.trim()) setProjectionText(projection)
      setError(null)
      setExplainResult(null)
      setExplainCollscan(false)
      onAggregateResult(null)
      onRun(filter.trim() || '{}', sort, projection)
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

  // Drops one field from the projection. The text may be JS-object-literal
  // shorthand rather than strict JSON (Run normalizes it, but the chips are
  // rendered before that happens), so fall back to computeJsonFix before
  // giving up — and if it still won't parse, leave the text alone. Clearing
  // it, as this used to, threw away every other exclusion the user had
  // typed just because one of them was quoted loosely.
  function removeHiddenField(field: string) {
    const source = (() => {
      try {
        JSON.parse(projectionText)
        return projectionText
      } catch {
        const fixed = computeJsonFix(projectionText)
        return typeof fixed === 'string' ? fixed : null
      }
    })()
    if (source === null) return

    try {
      const obj = JSON.parse(source) as Record<string, unknown>
      delete obj[field]
      setProjectionText(Object.keys(obj).length > 0 ? JSON.stringify(obj, null, 2) : '')
    } catch {
      // Parsed a moment ago; unreachable in practice.
    }
  }

  // Exports the query that actually produced the results on screen — the
  // `query` prop — not the draft in the editor. The draft may be
  // half-typed or not yet run, and exporting it would silently hand back a
  // different set of documents than the one being looked at. skip/limit
  // are deliberately absent: the server exports every matching document
  // and zeroes both anyway.
  async function runExport(format: 'json' | 'csv') {
    setError(null)
    setExporting(true)
    try {
      await startExportDownload(db, coll, format, {
        filter: query.filter || undefined,
        sort: query.sort || undefined,
        projection: query.projection || undefined,
      })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setExporting(false)
    }
  }

  async function explain() {
    const filter = textToRun(filterText)
    try {
      if (filter.trim()) JSON.parse(filter)
      // Same normalization Run does, so Explain describes exactly the
      // query Run would issue rather than only its filter.
      const sort = normalizeJsonField(sortText)
      const projection = normalizeJsonField(projectionText)
      setError(null)
      setExplaining(true)
      const result = await api.explain(db, coll, filter.trim() || '{}', sort, projection)
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

  // Whether the find-mode filter/sort/projection prepared here differ from
  // what actually produced the results currently shown (query, the last
  // value onRun was called with) — a plain trimmed-text comparison, not
  // semantic JSON equality, so a whitespace-only difference could show a
  // false positive; harmless, since this only drives a visual hint.
  const isDirty =
    mode === 'find' &&
    ((filterText.trim() || '{}') !== (query.filter?.trim() || '{}') ||
      sortText.trim() !== (query.sort ?? '').trim() ||
      projectionText.trim() !== (query.projection ?? '').trim())

  const hiddenFields = useMemo(() => parseProjectionFields(projectionText), [projectionText])

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
          basicSetup={{ lineNumbers: false, foldGutter: false, closeBrackets: false }}
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
          basicSetup={{ lineNumbers: false, foldGutter: false, closeBrackets: false }}
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
          basicSetup={{ lineNumbers: false, foldGutter: false, closeBrackets: false }}
        />
      )}

      {mode === 'find' && (
        <div className={styles.row}>
          <div className={styles.field}>
            <label className={styles.label}>{t('queryEditor.sortLabel')}</label>
            <input
              className={styles.textarea}
              style={{ minHeight: 'unset' }}
              value={sortText}
              onChange={(e) => setSortText(e.target.value)}
              placeholder={t('queryEditor.sortPlaceholder')}
            />
          </div>
        </div>
      )}
      {mode === 'find' && (
        <div className={styles.row}>
          <div className={styles.field}>
            <label className={styles.label}>{t('queryEditor.projectionLabel')}</label>
            <input
              className={styles.textarea}
              style={{ minHeight: 'unset' }}
              value={projectionText}
              onChange={(e) => setProjectionText(e.target.value)}
              placeholder={t('queryEditor.projectionPlaceholder')}
            />
          </div>
        </div>
      )}
      {mode === 'find' && hiddenFields.length > 0 && (
        <div className={styles.hiddenFieldsRow}>
          {t('queryEditor.hiddenFieldsLabel')}
          {hiddenFields.map((f) => (
            <span key={f} className={styles.hiddenFieldChip}>
              {f}
              <button
                className={styles.hiddenFieldRemove}
                onClick={() => removeHiddenField(f)}
                aria-label={t('queryEditor.removeHiddenField', { name: f })}
              >
                ✕
              </button>
            </span>
          ))}
        </div>
      )}
      {canFix && <div className={styles.hint}>{t('queryEditor.fixJsonHint')}</div>}

      <div className={styles.row}>
        {canFix && (
          <button className={styles.buttonDanger} onClick={applyFix}>
            {t('queryEditor.fixJson')}
          </button>
        )}
        <button
          className={isDirty ? styles.buttonPending : styles.button}
          onClick={run}
          disabled={running}
          title={isDirty ? t('queryEditor.pendingChangesHint') : undefined}
        >
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
            <button
              className={styles.exportLink}
              onClick={() => void runExport('json')}
              disabled={exporting}
              title={t('queryEditor.exportHint')}
            >
              {t('queryEditor.exportJson')}
            </button>
            <button
              className={styles.exportLink}
              onClick={() => void runExport('csv')}
              disabled={exporting}
              title={t('queryEditor.exportHint')}
            >
              {t('queryEditor.exportCsv')}
            </button>
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
