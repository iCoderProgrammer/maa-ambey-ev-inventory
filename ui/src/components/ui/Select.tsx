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

/** Native select: keyboard and screen-reader support come for free. */
export function Select({ options, placeholder, className, ...props }: SelectProps) {
  return (
    <select className={cn(controlClass, 'pr-8', className)} {...props}>
      {placeholder !== undefined && <option value="">{placeholder}</option>}
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  )
}
