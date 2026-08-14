import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import en from './locales/en.json'
import pt from './locales/pt.json'

// English is the default; Portuguese is used only when the browser asks
// for it (navigator.language). No manual language switcher in the UI, so
// there is nothing to persist — the language always follows the browser
// on every load.
void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: { en: { translation: en }, pt: { translation: pt } },
    fallbackLng: 'en',
    supportedLngs: ['en', 'pt'],
    detection: { order: ['navigator'], caches: [] },
    interpolation: { escapeValue: false },
  })

export default i18n
