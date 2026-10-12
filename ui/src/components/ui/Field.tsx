import { useId, type ReactElement, type ReactNode } from 'react'

export interface FieldControlProps {
  id: string
  'aria-invalid'?: boolean
  'aria-describedby'?: string
}

export interface FieldProps {
  label: string
  hint?: string
  error?: string
  required?: boolean
  /** Render prop receiving the ids that wire the control to its label/hint/error. */
  children: (control: FieldControlProps) => ReactElement
}

/** Label + hint + error wrapper that keeps form controls accessible. */
export function Field({ label, hint, error, required, children }: FieldProps): ReactNode {
  const id = useId()
  const hintId = hint ? `${id}-hint` : undefined
  const errorId = error ? `${id}-error` : undefined
  const describedBy = [hintId, errorId].filter(Boolean).join(' ') || undefined

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-medium text-ink">
        {label}
        {required && (
          <span className="ml-0.5 text-danger" aria-hidden="true">
            *
          </span>
        )}
      </label>
      {children({ id, 'aria-invalid': error ? true : undefined, 'aria-describedby': describedBy })}
      {hint && (
        <p id={hintId} className="text-xs text-muted">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} role="alert" className="text-xs text-danger">
          {error}
        </p>
      )}
    </div>
  )
}
