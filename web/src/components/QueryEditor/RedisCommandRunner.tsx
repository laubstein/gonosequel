import { useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation } from '@tanstack/react-query'
import CodeMirror from '@uiw/react-codemirror'
import { autocompletion } from '@codemirror/autocomplete'
import { keymap, type EditorView } from '@codemirror/view'
import { Prec } from '@codemirror/state'
import styles from './QueryEditor.module.css'
import { api } from '../../api/client'
import { redisCommandCompletionSource } from './redisCommandCompletion'
import { formatRedisResult } from './redisResultFormat'
import type { CommandResult } from '../../types'

interface Props {
  db: string
  coll: string
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
  const [script, setScript] = useState('')
  const [presetIndex, setPresetIndex] = useState('')
  const [results, setResults] = useState<CommandResult[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const viewRef = useRef<EditorView | null>(null)

  const run = useMutation({
    mutationFn: (text: string) => api.runCommand(db, text),
    onSuccess: (res) => {
      setResults(res)
      setError(null)
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
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
    setScript((s) => (s.trim() ? `${s}\n${line}` : line))
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
          onChange={(value) => setScript(value)}
          onCreateEditor={(view) => {
            viewRef.current = view
          }}
          placeholder={t('redisRunner.placeholder')}
          basicSetup={{ lineNumbers: true, foldGutter: false }}
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
