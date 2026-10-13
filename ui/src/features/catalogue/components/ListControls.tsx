import { Search, X } from 'lucide-react'
import { useEffect, useId, useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select, type SelectOption } from '@/components/ui/Select'

/** Search box that updates the list after a short pause or on Enter. */
export function SearchBox({
  value,
  onSearch,
  label,
}: {
  value: string
  onSearch: (q: string) => void
  label: string
}) {
  const id = useId()
  const [draft, setDraft] = useState(value)
  // Follow outside changes (clear filters, back/forward) without remounting, so focus stays.
  const [synced, setSynced] = useState(value)
  if (value !== synced) {
    setSynced(value)
    setDraft(value)
  }
  useEffect(() => {
    if (draft === value) return
    const t = setTimeout(() => {
      onSearch(draft.trim())
    }, 350)
    return () => {
      clearTimeout(t)
    }
  }, [draft, value, onSearch])
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1.5 sm:min-w-56">
      <label htmlFor={id} className="text-xs font-medium text-muted">
        {label}
      </label>
      <div className="relative">
        <Search
          className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted"
          aria-hidden="true"
        />
        <Input
          id={id}
          type="search"
          value={draft}
          maxLength={100}
          className="pl-9"
          onChange={(e) => {
            setDraft(e.target.value)
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') onSearch(draft.trim())
          }}
        />
      </div>
    </div>
  )
}

export function FilterSelect({
  label,
  value,
  options,
  onChange,
  allLabel = 'All',
  disabled,
}: {
  label: string
  value: string
  options: SelectOption[]
  onChange: (v: string) => void
  allLabel?: string
  disabled?: boolean
}) {
  const id = useId()
  return (
    <div className="flex min-w-0 flex-col gap-1.5 sm:w-48">
      <label htmlFor={id} className="text-xs font-medium text-muted">
        {label}
      </label>
      <Select
        id={id}
        value={value}
        options={options}
        placeholder={allLabel}
        disabled={disabled}
        onChange={(e) => {
          onChange(e.target.value)
        }}
      />
    </div>
  )
}

export function Toolbar({
  children,
  onClear,
  active,
}: {
  children: ReactNode
  onClear: () => void
  active: boolean
}) {
  return (
    <div
      role="search"
      className="mb-4 flex flex-col gap-3 rounded-card border border-border/80 bg-surface p-4 shadow-xs sm:flex-row sm:flex-wrap sm:items-end"
    >
      {children}
      {active && (
        <Button variant="ghost" size="sm" onClick={onClear} className="sm:mb-1">
          <X className="size-3.5" aria-hidden="true" />
          Clear filters
        </Button>
      )}
    </div>
  )
}
