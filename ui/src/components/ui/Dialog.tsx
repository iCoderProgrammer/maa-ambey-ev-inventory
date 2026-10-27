import { X } from 'lucide-react'
import { Dialog as DialogPrimitive } from 'radix-ui'
import { useRef, type ReactNode } from 'react'
import { Button } from './Button'

export interface DialogProps {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  children?: ReactNode
  footer?: ReactNode
}

/**
 * Modal dialog on Radix Dialog: focus trap, Escape/overlay close, inert background,
 * scroll lock and focus return to the opener. For irreversible actions put a
 * descriptive confirm button in `footer` ("Post receipt", not "OK").
 */
export function Dialog({ open, onClose, title, description, children, footer }: DialogProps) {
  // Radix returns focus only to its own Trigger; these dialogs are opened from ordinary buttons.
  const openerRef = useRef<HTMLElement | null>(null)
  return (
    <DialogPrimitive.Root
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose()
      }}
    >
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-ink/40 backdrop-blur-xs data-[state=closed]:animate-fade-out data-[state=open]:animate-fade-in dark:bg-black/70" />
        <DialogPrimitive.Content
          className="fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-2rem)] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-card border border-border/80 bg-surface text-ink shadow-2xl data-[state=closed]:animate-pop-out data-[state=open]:animate-pop-in"
          {...(description ? {} : { 'aria-describedby': undefined })}
          onOpenAutoFocus={() => {
            openerRef.current =
              document.activeElement instanceof HTMLElement ? document.activeElement : null
          }}
          onCloseAutoFocus={(event) => {
            // An opener removed meanwhile (e.g. after navigating) leaves focus to the new page.
            if (openerRef.current?.isConnected) {
              event.preventDefault()
              openerRef.current.focus()
            }
          }}
        >
          <div className="flex items-start justify-between gap-4 border-b border-border px-5 py-4">
            <div>
              <DialogPrimitive.Title className="text-lg font-semibold">
                {title}
              </DialogPrimitive.Title>
              {description && (
                <DialogPrimitive.Description className="mt-1 text-sm text-muted">
                  {description}
                </DialogPrimitive.Description>
              )}
            </div>
            <DialogPrimitive.Close asChild>
              <Button variant="ghost" size="sm" aria-label="Close dialog" className="-mr-2">
                <X className="size-4" aria-hidden="true" />
              </Button>
            </DialogPrimitive.Close>
          </div>
          {children && <div className="px-5 py-4">{children}</div>}
          {footer && (
            <div className="flex flex-col-reverse gap-2 border-t border-border px-5 py-3 sm:flex-row sm:justify-end">
              {footer}
            </div>
          )}
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
