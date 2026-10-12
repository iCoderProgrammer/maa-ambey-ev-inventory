import type { ReactNode } from 'react'
import { cn } from '@/lib/utils/cn'

export interface Column<T> {
  key: string
  header: string
  render: (row: T) => ReactNode
  className?: string
}

export interface TableProps<T> {
  caption: string
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string
  /** Shown instead of an empty body. */
  empty?: ReactNode
}

/** Accessible data table with sticky header; scrolls horizontally on narrow screens. */
export function Table<T>({ caption, columns, rows, rowKey, empty }: TableProps<T>) {
  return (
    <div className="overflow-x-auto rounded-card border border-border/80 bg-surface shadow-xs">
      <table className="w-full border-collapse text-left text-sm">
        <caption className="sr-only">{caption}</caption>
        <thead className="sticky top-0 z-10 border-b border-border bg-surface-muted/90 backdrop-blur-xs text-[11px] font-semibold tracking-wider text-muted uppercase">
          <tr>
            {columns.map((c) => (
              <th key={c.key} scope="col" className={cn('px-4 py-3', c.className)}>
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-border/80">
          {rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-10 text-center text-sm text-muted">
                {empty ?? 'No records.'}
              </td>
            </tr>
          ) : (
            rows.map((row) => (
              <tr
                key={rowKey(row)}
                className="bg-surface transition-colors duration-150 hover:bg-surface-muted/40"
              >
                {columns.map((c) => (
                  <td key={c.key} className={cn('px-4 py-3 text-ink align-middle', c.className)}>
                    {c.render(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}
