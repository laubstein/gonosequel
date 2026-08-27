import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import CodeMirror from '@uiw/react-codemirror'
import { autocompletion } from '@codemirror/autocomplete'
import { keymap, type EditorView } from '@codemirror/view'
import { Prec } from '@codemirror/state'
import styles from './QueryEditor.module.css'
import { api } from '../../api/client'
import { readLocal, writeLocal } from '../../api/localCache'
import { useIsDarkMode } from '../../hooks/useIsDarkMode'
import { redisCommandCompletionSource } from './redisCommandCompletion'
import { formatRedisResult } from './redisResultFormat'
import type { CommandResult } from '../../types'

interface Props {
  db: string
  coll: string
}

// scriptCache persists each collection's in-progress command script across
// collection switches — keyed per collection so switching away and back
// restores the right script instead of leaking the previous collection's
// content or wiping it back to empty. Backed by localStorage (with this
// in-memory Map as a read cache in front of it) so it also survives a page
// refresh and — since it isn't tied to a session ID — reconnecting and
// revisiting the same collection later.
const scriptCache = new Map<string, string>()

function scriptKey(db: string, coll: string): string {
  return `${db}:${coll}`
}

function scriptStorageKey(key: string): string {
  return `gonosequel.commandDraft:${key}`
}

function loadScript(key: string): string {
  const cached = scriptCache.get(key)
  if (cached !== undefined) return cached
  const script = readLocal<string>(scriptStorageKey(key)) ?? ''
  scriptCache.set(key, script)
  return script
}

function saveScript(key: string, next: string) {
  scriptCache.set(key, next)
  writeLocal(scriptStorageKey(key), next)
}

// PRESETS inserts a ready-to-run command line into the textarea — the
// Redis/Valkey equivalent of QueryEditor's Mongo-shaped presets, using
// {coll} as a placeholder for the currently selected collection's key
// prefix.
const PRESETS: { labelKey: string; command: (coll: string) => string }[] = [
  { labelKey: 'redisRunner.presetScan', command: (coll) => `SCAN 0 MATCH ${coll}:* COUNT 100` },
  { labelKey: 'redisRunner.presetType', command: () => 'TYPE key' },
  { labelKey: 'redisRunner.presetTtl', command: () => 'TTL key' },
  { labelKey: 'redisRunner.presetGet', command: () => 'GET key' },
  { labelKey: 'redisRunner.presetDel', command: () => 'DEL key' },
]

// RedisCommandRunner replaces QueryEditor+Results for Redis/Valkey
// connections: a multi-line textarea where each line is a raw backend
// command, run in sequence (redis-cli's own pipe/batch behavior — see
// pkg/api/handlers_command.go), with results shown as redis-cli-style
// plain text rather than the Mongo-shaped document table. The sidebar's
// key browsing table and the RedisValueEditor structured editor (opened
// by clicking a key, or "+ New key") are unrelated to this and unaffected.
export function RedisCommandRunner({ db, coll }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const key = scriptKey(db, coll)
  const [script, setScriptState] = useState(() => loadScript(key))
  const [presetIndex, setPresetIndex] = useState('')
  const [results, setResults] = useState<CommandResult[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const viewRef = useRef<EditorView | null>(null)
  const isDark = useIsDarkMode()

  function setScript(next: string) {
    setScriptState(next)
    saveScript(key, next)
  }

  const prevKeyRef = useRef(key)
  useEffect(() => {
    if (prevKeyRef.current === key) return
    prevKeyRef.current = key
    setScriptState(loadScript(key))
    setPresetIndex('')
    setResults(null)
    setError(null)
  }, [key])

  const run = useMutation({
    mutationFn: (text: string) => api.runCommand(db, text),
    onSuccess: (res) => {
      setResults(res)
      setError(null)
      // A command typed here can write (SET), delete (DEL), or wipe the
      // database (FLUSHDB) — but this runner is outside the query/mutation
      // flow those views use, so nothing else knows the data moved. Without
      // this, the key table and the sidebar's collection list keep showing
      // what was true before the command ran, indefinitely. The command
      // text isn't parsed to decide whether it wrote: a read-only command
      // just makes this a no-op refetch.
      void queryClient.invalidateQueries({ queryKey: ['documents', db] })
      void queryClient.invalidateQueries({ queryKey: ['collections', db] })
      void queryClient.invalidateQueries({ queryKey: ['collectionStats', db] })
    },
    onError: (e) => {
      setError(e instanceof Error ? e.message : String(e))
      // Drop the previous run's output: leaving it under the new error
      // message reads as though it were this command's result.
      setResults(null)
    },
  })

  // Read via the live EditorView rather than the `script` state: the
  // keymap command below is captured once by the memoized extensions array,
  // so closing over `script`/`run.isPending` directly would see stale
  // values from whichever render first built the extension. The view's own
  // state (passed fresh on every keypress) and this ref are always current.
  const pendingRef = useRef(false)
  pendingRef.current = run.isPending

  // If the user has a non-empty text selection, running only executes the
  // selected text — matches the "run selection" convention of most script
  // runners (mongosh's own shell, SQL clients). Otherwise the whole script
  // runs, as before.
  function textToRun(view: EditorView | null): string {
    if (!view) return script
    const sel = view.state.selection.main
    if (!sel.empty) return view.state.sliceDoc(sel.from, sel.to)
    return view.state.doc.toString()
  }

  function runNow(view: EditorView | null) {
    if (pendingRef.current) return
    const text = textToRun(view)
    if (!text.trim()) return
    run.mutate(text)
  }

  const extensions = useMemo(
    () => [
      autocompletion({ override: [redisCommandCompletionSource] }),
      Prec.highest(
        keymap.of([
          {
            key: 'Mod-Enter',
            run: (view) => {
              runNow(view)
              return true
            },
          },
        ]),
      ),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  )

  function applyPreset(index: string) {
    setPresetIndex(index)
    if (index === '') return
    const preset = PRESETS[Number(index)]
    if (!preset) return
    const line = preset.command(coll)
    setScript(script.trim() ? `${script}\n${line}` : line)
  }

  return (
    <>
      <div className={styles.editor}>
        <div className={styles.row}>
          <select className={styles.presetSelect} value={presetIndex} onChange={(e) => applyPreset(e.target.value)}>
            <option value="">{t('queryEditor.presetPlaceholder')}</option>
            {PRESETS.map((preset, i) => (
              <option key={i} value={i}>
                {t(preset.labelKey)}
              </option>
            ))}
          </select>
        </div>

        <CodeMirror
          value={script}
          height="100px"
          extensions={extensions}
          theme={isDark ? 'dark' : 'light'}
          onChange={(value) => setScript(value)}
          onCreateEditor={(view) => {
            viewRef.current = view
          }}
          placeholder={t('redisRunner.placeholder')}
          basicSetup={{ lineNumbers: true, foldGutter: false, closeBrackets: false }}
        />

        <div className={styles.row}>
          <button className={styles.button} onClick={() => runNow(viewRef.current)} disabled={run.isPending || !script.trim()}>
            {run.isPending ? t('redisRunner.running') : t('redisRunner.run')}
          </button>
          {error && <span className={styles.error}>{error}</span>}
        </div>
      </div>

      {results && (
        <div className={styles.explainOutput}>
          {results.map((r, i) => (
            <div key={i}>
              <div>{`> ${r.command}`}</div>
              <div style={r.error ? { color: 'var(--color-danger)' } : undefined}>
                {r.error ? `(error) ${r.error}` : formatRedisResult(r.result)}
              </div>
              {i < results.length - 1 && <br />}
            </div>
          ))}
        </div>
      )}
    </>
  )
}
