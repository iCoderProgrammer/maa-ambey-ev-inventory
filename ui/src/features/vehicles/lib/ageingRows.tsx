import type { ReactNode } from 'react'
import { formatDate } from '@/features/catalogue/lib/presentation'
import type { StockAgeing } from '../api/vehicleUnits'
import { AgeingBadge } from '../components/Ageing'
import { missingReasonLabels, originSourceLabels, stockAgeText } from './ageing'

/** Detail rows for the vehicle card. */
export function ageingRows(a: StockAgeing): [string, ReactNode][] {
  if (a.status === 'MISSING_ORIGIN')
    return [
      ['Stock age', <AgeingBadge ageing={a} />],
      [
        'Why no age',
        <span>
          {a.missingReason ? missingReasonLabels[a.missingReason] : ''}
          {a.missingExplanation && (
            <span className="block text-xs text-muted">{a.missingExplanation}</span>
          )}
        </span>,
      ],
    ]
  return [
    ['Stock age', `${stockAgeText(a)} (as of ${formatDate(a.asOf)})`],
    ['Ageing band', <AgeingBadge ageing={a} />],
    [
      'Origin date',
      a.originDate && a.originSource
        ? `${formatDate(a.originDate)} (${originSourceLabels[a.originSource]})`
        : '—',
    ],
  ]
}
