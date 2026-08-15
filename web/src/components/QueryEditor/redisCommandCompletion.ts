import type { CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import { REDIS_COMMANDS } from './redisCommands'

// redisCommandCompletionSource offers Redis command names whenever the
// cursor is at the start of a line (the command-name position) — the same
// scope redis-cli's own tab-completion covers, not per-argument hints.
export function redisCommandCompletionSource(context: CompletionContext): CompletionResult | null {
  const line = context.state.doc.lineAt(context.pos)
  const textBeforeCursor = context.state.sliceDoc(line.from, context.pos)
  // Only complete while the line's only content so far is the command name
  // itself — once there's a space, the user has moved on to arguments.
  if (/\s/.test(textBeforeCursor)) return null

  const word = context.matchBefore(/\w*/)
  if (!word) return null
  if (word.from === word.to && !context.explicit) return null

  return {
    from: word.from,
    options: REDIS_COMMANDS.map((cmd) => ({
      label: cmd.name,
      type: 'keyword',
      detail: cmd.syntax,
    })),
    validFor: /^\w*$/,
  }
}
