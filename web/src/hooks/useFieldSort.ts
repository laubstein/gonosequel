import { useEffect, useState } from 'react'
import { readLocal, writeLocal } from '../api/localCache'

const STORAGE_KEY = 'gonosequel.sortFields'

// Whether field names are shown in alphabetical order, everywhere they
// are shown: the results table's columns, the results JSON view, and the
// document editor.
//
// One preference rather than three toggles — wanting fields sorted is a
// property of how you read documents, not of which panel you happen to be
// looking at. The subscriber list keeps every mounted consumer in step,
// since the toggle appears in more than one place at once and React state
// alone would leave each copy with its own value.
let sortFields = readLocal<boolean>(STORAGE_KEY) ?? false
const subscribers = new Set<(value: boolean) => void>()

export function useFieldSort() {
  const [value, setValue] = useState(sortFields)

  useEffect(() => {
    subscribers.add(setValue)
    // Adopt whatever the shared value is now, in case it changed between
    // this component's first render and this effect running.
    setValue(sortFields)
    return () => {
      subscribers.delete(setValue)
    }
  }, [])

  function toggle() {
    sortFields = !sortFields
    writeLocal(STORAGE_KEY, sortFields)
    for (const notify of subscribers) notify(sortFields)
  }

  return { sortFields: value, toggleFieldSort: toggle }
}
