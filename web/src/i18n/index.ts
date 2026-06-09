import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import zhCN from './locales/zh-CN.json'

const STORAGE_KEY = 'web_locale'

function savedLocale(): string {
  return localStorage.getItem(STORAGE_KEY) || 'en'
}

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    'zh-CN': { translation: zhCN },
  },
  lng: savedLocale(),
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
})

export function setLocale(locale: 'en' | 'zh-CN') {
  localStorage.setItem(STORAGE_KEY, locale)
  void i18n.changeLanguage(locale)
}

export default i18n
