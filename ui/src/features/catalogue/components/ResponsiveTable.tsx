import { Table, type Column } from '@/components/ui/Table'

/**
 * Table on wider screens; on phones a stacked list where every column is a
 * labelled line, so names and statuses are visible without sideways scrolling.
 * The first column is the record's title.
 */
export function ResponsiveTable<T>({
  caption,
  columns,
  rows,
  rowKey,
}: {
  caption: string
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string
}) {
  const [title, ...rest] = columns
  return (
    <>
      <div className="hidden sm:block">
        <Table caption={caption} columns={columns} rows={rows} rowKey={rowKey} />
      </div>
      <ul aria-label={caption} className="flex flex-col gap-3 sm:hidden">
        {rows.map((row) => (
          <li
            key={rowKey(row)}
            className="rounded-card border border-border/80 bg-surface p-4 shadow-xs"
          >
            {title && <div className="mb-3 text-sm">{title.render(row)}</div>}
            <dl className="flex flex-col gap-2 text-sm">
              {rest.map((c) => (
                <div
                  key={c.key}
                  className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1"
                >
                  <dt className="text-xs font-medium text-muted">{c.header}</dt>
                  <dd className="text-ink">{c.render(row)}</dd>
                </div>
              ))}
            </dl>
          </li>
        ))}
      </ul>
    </>
  )
}
