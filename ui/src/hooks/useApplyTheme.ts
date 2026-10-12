import { useEffect } from 'react'
import { useAppSelector } from '@/store'
import { THEME_STORAGE_KEY } from '@/store/uiSlice'

/** Applies the theme preference to <html> and follows OS changes in "system" mode. */
export function useApplyTheme(): void {
  const theme = useAppSelector((s) => s.ui.theme)

  useEffect(() => {
    try {
      localStorage.setItem(THEME_STORAGE_KEY, theme)
    } catch {
      // Persisting the preference is a convenience only.
    }

    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const apply = () => {
      const dark = theme === 'dark' || (theme === 'system' && media.matches)
      document.documentElement.classList.toggle('dark', dark)
    }
    apply()
    if (theme !== 'system') return
    media.addEventListener('change', apply)
    return () => {
      media.removeEventListener('change', apply)
    }
  }, [theme])
}
