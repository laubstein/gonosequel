import type { CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import type { SchemaField } from '../../types'

// fieldCompletionSource offers the collection's known field paths (from
// pkg/client's $sample-based schema inference) whenever the cursor sits
// inside or right after a JSON key's opening quote.
export function fieldCompletionSource(fields: SchemaField[]) {
  return (context: CompletionContext): CompletionResult | null => {
    const word = context.matchBefore(/"[\w.]*/)
    if (!word) return null
    if (word.from === word.to && !context.explicit) return null

    return {
      from: word.from + 1, // skip the opening quote
      options: fields.map((f) => ({
        label: f.path,
        type: 'property',
        detail: f.types.map((t) => t.type).join(' | '),
      })),
      validFor: /^[\w.]*$/,
    }
  }
}
