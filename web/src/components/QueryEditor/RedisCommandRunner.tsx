import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation } from '@tanstack/react-query'
import CodeMirror from '@uiw/react-codemirror'
import { autocompletion } from '@codemirror/autocomplete'
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

  const extensions = [autocompletion({ override: [redisCommandCompletionSource] })]

  const run = useMutation({
    mutationFn: () => api.runCommand(db, script),
    onSuccess: (res) => {
      setResults(res)
      setError(null)
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

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
          placeholder={t('redisRunner.placeholder')}
          basicSetup={{ lineNumbers: true, foldGutter: false }}
        />

        <div className={styles.row}>
          <button className={styles.button} onClick={() => run.mutate()} disabled={run.isPending || !script.trim()}>
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
