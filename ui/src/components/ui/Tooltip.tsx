import { Tooltip as TooltipPrimitive } from 'radix-ui'
import type { ReactNode } from 'react'

export const TooltipProvider = TooltipPrimitive.Provider

export interface TooltipProps {
  content: ReactNode
  side?: 'top' | 'right' | 'bottom' | 'left'
  /** Show the tooltip only when true (e.g. only for the collapsed sidebar). */
  enabled?: boolean
  /** A single focusable element; it receives the trigger props. */
  children: ReactNode
}

/**
 * Short supplementary label on hover and keyboard focus. Never put essential
 * information only in a tooltip: the trigger needs its own accessible name.
 */
export function Tooltip({ content, side = 'top', enabled = true, children }: TooltipProps) {
  if (!enabled) return children
  return (
    <TooltipPrimitive.Root>
      <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          side={side}
          sideOffset={8}
          className="z-50 rounded-control bg-ink px-2.5 py-1.5 text-xs font-medium text-surface shadow-md data-[state=closed]:animate-fade-out data-[state=delayed-open]:animate-fade-in data-[state=instant-open]:animate-fade-in"
        >
          {content}
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  )
}
