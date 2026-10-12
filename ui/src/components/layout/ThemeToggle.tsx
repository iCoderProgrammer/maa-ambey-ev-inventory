import { Monitor, Moon, Sun } from 'lucide-react'
import { useAppDispatch, useAppSelector } from '@/store'
import { setTheme, type ThemePreference } from '@/store/uiSlice'
import { cn } from '@/lib/utils/cn'

const options: { value: ThemePreference; label: string; Icon: typeof Sun }[] = [
  { value: 'light', label: 'Light theme', Icon: Sun },
  { value: 'dark', label: 'Dark theme', Icon: Moon },
  { value: 'system', label: 'Use system theme', Icon: Monitor },
]

export function ThemeToggle() {
  const theme = useAppSelector((s) => s.ui.theme)
  const dispatch = useAppDispatch()

  return (
    <div
      role="radiogroup"
      aria-label="Colour theme"
      className="flex rounded-control border border-border p-0.5"
    >
      {options.map(({ value, label, Icon }) => (
        <button
          key={value}
          type="button"
          role="radio"
          aria-checked={theme === value}
          aria-label={label}
          title={label}
          onClick={() => dispatch(setTheme(value))}
          className={cn(
            'grid size-7 cursor-pointer place-items-center rounded-[6px] text-muted transition-colors duration-150',
            theme === value ? 'bg-surface-muted text-ink shadow-xs' : 'hover:text-ink',
          )}
        >
          <Icon className="size-4" aria-hidden="true" />
        </button>
      ))}
    </div>
  )
}
