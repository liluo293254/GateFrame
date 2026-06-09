import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, ChevronDown, Languages } from 'lucide-react'
import { setLocale } from '@/i18n'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const LOCALES = [
  { value: 'en' as const, labelKey: 'common.english' },
  { value: 'zh-CN' as const, labelKey: 'common.chinese' },
] as const

type LocaleValue = (typeof LOCALES)[number]['value']

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation()
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const current: LocaleValue = i18n.language.startsWith('zh') ? 'zh-CN' : 'en'

  useEffect(() => {
    if (!open) return

    function onPointerDown(event: PointerEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setOpen(false)
      }
    }

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }

    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  function selectLocale(locale: LocaleValue) {
    setLocale(locale)
    setOpen(false)
  }

  return (
    <div ref={containerRef} className="relative">
      <Button
        type="button"
        variant="outline"
        size="sm"
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-label={t('common.language')}
        className="h-9 gap-1.5 px-2.5 font-normal shadow-none"
        onClick={() => setOpen((value) => !value)}
      >
        <Languages className="size-4 shrink-0 text-muted-foreground" />
        <span className="max-w-[5.5rem] truncate">
          {t(LOCALES.find((item) => item.value === current)!.labelKey)}
        </span>
        <ChevronDown
          className={cn(
            'size-4 shrink-0 text-muted-foreground transition-transform duration-200',
            open && 'rotate-180',
          )}
        />
      </Button>

      {open ? (
        <ul
          role="listbox"
          aria-label={t('common.language')}
          className="absolute right-0 z-50 mt-1.5 min-w-[9.5rem] overflow-hidden rounded-md border bg-background p-1 shadow-lg"
        >
          {LOCALES.map(({ value, labelKey }) => {
            const selected = value === current
            return (
              <li key={value} role="presentation">
                <button
                  type="button"
                  role="option"
                  aria-selected={selected}
                  className={cn(
                    'flex w-full items-center gap-2 rounded-sm px-2.5 py-2 text-left text-sm transition-colors',
                    'hover:bg-muted focus-visible:bg-muted focus-visible:outline-none',
                    selected && 'bg-muted/70 font-medium',
                  )}
                  onClick={() => selectLocale(value)}
                >
                  <Check
                    className={cn(
                      'size-4 shrink-0 text-primary',
                      !selected && 'invisible',
                    )}
                  />
                  <span>{t(labelKey)}</span>
                </button>
              </li>
            )
          })}
        </ul>
      ) : null}
    </div>
  )
}
