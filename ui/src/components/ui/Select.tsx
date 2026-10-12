import { ChevronDown } from 'lucide-react'
import type { Ref, SelectHTMLAttributes } from 'react'
import { cn } from '@/lib/utils/cn'
import { controlClass } from './controlClass'

export interface SelectOption {
  value: string
  label: string
}

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  options: SelectOption[]
  placeholder?: string
  ref?: Ref<HTMLSelectElement>
}

/** Native select with styled chevron: keyboard and screen-reader support come for free. */
export function Select({ options, placeholder, className, ...props }: SelectProps) {
  return (
    <div className="relative w-full">
      <select
        className={cn(controlClass, 'appearance-none pr-10 cursor-pointer', className)}
        {...props}
      >
        {placeholder !== undefined && <option value="">{placeholder}</option>}
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
      <ChevronDown
        className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 size-4 text-muted transition-transform duration-150"
        aria-hidden="true"
      />
    </div>
  )
}
