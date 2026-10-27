import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from './Button'

export interface PaginationProps {
  page: number
  limit: number
  total: number
  onPage: (page: number) => void
  /** Singular/plural noun for the record count, e.g. ["brand", "brands"]. */
  noun?: [string, string]
}

/** Server pagination: record range, total and previous/next. */
export function Pagination({
  page,
  limit,
  total,
  onPage,
  noun = ['record', 'records'],
}: PaginationProps) {
  const pages = Math.max(1, Math.ceil(total / limit))
  const from = total === 0 ? 0 : (page - 1) * limit + 1
  const to = Math.min(total, page * limit)
  // A stale link or a shrunk result can ask for a page past the end: say so and lead back.
  const beyond = total > 0 && page > pages
  return (
    <nav
      aria-label="Pagination"
      className="flex flex-wrap items-center justify-between gap-3 pt-3 text-sm text-muted"
    >
      <p aria-live="polite">
        {total === 0
          ? `0 ${noun[1]}`
          : beyond
            ? `No ${noun[1]} on this page (${String(total)} in total)`
            : `${String(from)}–${String(to)} of ${String(total)} ${total === 1 ? noun[0] : noun[1]}`}
      </p>
      <div className="flex items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => {
            onPage(beyond ? pages : page - 1)
          }}
          disabled={page <= 1}
          aria-label="Previous page"
        >
          <ChevronLeft className="size-4" aria-hidden="true" />
          Previous
        </Button>
        <span className="tabular-nums">
          Page {page} of {pages}
        </span>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => {
            onPage(page + 1)
          }}
          disabled={page >= pages}
          aria-label="Next page"
        >
          Next
          <ChevronRight className="size-4" aria-hidden="true" />
        </Button>
      </div>
    </nav>
  )
}
