import type { InputHTMLAttributes, Ref } from 'react'
import { cn } from '@/lib/utils/cn'
import { controlClass } from './controlClass'

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  ref?: Ref<HTMLInputElement>
}

export function Input({ className, ...props }: InputProps) {
  return <input className={cn(controlClass, className)} {...props} />
}
