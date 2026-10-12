import { clsx, type ClassValue } from 'clsx'
import { extendTailwindMerge } from 'tailwind-merge'

// Teach tailwind-merge the project's custom radius tokens (src/styles/index.css @theme).
const twMerge = extendTailwindMerge({
  extend: { theme: { radius: ['card', 'control'] } },
})

/** Joins class names and resolves Tailwind conflicts (the last class wins). */
export function cn(...classes: ClassValue[]): string {
  return twMerge(clsx(classes))
}
