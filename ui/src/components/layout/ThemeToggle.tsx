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
      className="flex items-center rounded-control border border-border/80 bg-surface-muted/60 p-0.5 shadow-xs"
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
            'grid size-7 cursor-pointer place-items-center rounded-[6px] text-muted transition-[background-color,color,box-shadow,transform] duration-150',
            theme === value
              ? 'bg-surface text-ink shadow-xs ring-1 ring-border/60'
              : 'hover:text-ink hover:bg-surface-muted/50 active:scale-95',
          )}
        >
          <Icon className="size-3.5" aria-hidden="true" />
        </button>
      ))}
    </div>
  )
}
