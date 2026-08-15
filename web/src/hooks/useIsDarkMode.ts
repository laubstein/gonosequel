import { useEffect, useState } from 'react'

// computeIsDark mirrors useTheme.ts's own resolution: an explicit
// data-theme attribute on <html> wins, otherwise fall back to the OS
// preference — same rule the app's own dark-mode CSS variables use (see
// index.css's `@media (prefers-color-scheme: dark)` block).
function computeIsDark(): boolean {
  const attr = document.documentElement.getAttribute('data-theme')
  if (attr === 'dark') return true
  if (attr === 'light') return false
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

// useIsDarkMode reports whether the app is currently in dark mode, for
// components (CodeMirror) that need a real light/dark theme prop instead
// of CSS variables. It observes the DOM directly rather than duplicating
// useTheme's own React state, since multiple independent useState('system')
// instances would drift out of sync within the same tab — this hook stays
// correct regardless of which component's toggle button changed the theme.
export function useIsDarkMode(): boolean {
  const [isDark, setIsDark] = useState(computeIsDark)

  useEffect(() => {
    const update = () => setIsDark(computeIsDark())
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    media.addEventListener('change', update)
    const observer = new MutationObserver(update)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
    return () => {
      media.removeEventListener('change', update)
      observer.disconnect()
    }
  }, [])

  return isDark
}
