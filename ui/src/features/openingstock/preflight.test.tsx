/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { cleanup, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/preflight'
import type { Preflight } from './api/preflight'

// H-08 reconciliation preflight against a mocked API. Placeholder values only
// ("Brand A", "CH-0001"): nothing real, nothing imported, nothing counted.

const t = '2026-11-17T09:00:00Z'
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations: [{ id: 'l1', code: 'L1', name: 'Location One' }] },
})
const PERM = ['stock.import.preview']

const src = (chassis: string) => ({
  brand: 'Brand A',
  model_code: 'MA',
  variant_code: 'STD',
  chassis_number: chassis,
  location_code: 'L1',
  stock_status: 'AVAILABLE',
})

const preflight = (over: Partial<Preflight> = {}): Preflight => ({
  batchId: 'b1',
  batchReference: 'OSI-000001',
  batchStatus: 'PREVIEWED',
  fileName: 'stock.csv',
  fileSha256: 'a'.repeat(64),
  fileSizeBytes: 2048,
  templateVersion: 'OS-IMPORT-1',
  uploadedByName: 'Test User',
  uploadedAt: t,
  expiresAt: '2026-11-24T09:00:00Z',
  expired: false,
  checkedAt: t,
  businessDate: '2026-11-17',
  nonMutating: true,
  rowsChecked: true,
  source: {
    status: 'VERIFIED',
    fileAvailable: true,
    checksumMatches: true,
    sizeMatches: true,
    templateCompatible: true,
    matchesStagedRows: true,
    message: 'The stored file matches the upload (SHA-256, size, template and lines).',
  },
  summary: {
    totalRows: 4,
    validRows: 1,
    warningRows: 1,
    blockingErrorRows: 1,
    unverifiedRows: 0,
    manualReconciliationRows: 1,
    duplicateRows: 1,
    missingChassisRows: 0,
    changedSincePreviewRows: 1,
  },
  previewSummary: {
    totalRows: 4,
    validRows: 2,
    warningRows: 1,
    errorRows: 1,
    blockedRows: 0,
    canProceedRows: 3,
    duplicateRows: 1,
    unresolvedRows: 0,
    skippedBlankRows: 0,
  },
  issueCounts: [{ code: 'DUPLICATE_CHASSIS_EXISTING', severity: 'ERROR', rows: 1 }],
  quantities: [
    {
      locationCode: 'L1',
      brand: 'Brand A',
      modelCode: 'MA',
      variantCode: 'STD',
      stockStatus: 'AVAILABLE',
      fileRows: 4,
      validRows: 1,
      physicalCount: null,
      difference: null,
    },
  ],
  physicalCount: {
    status: 'NOT_RECORDED',
    reference: null,
    countDate: null,
    message: 'No signed physical count is linked to this batch.',
  },
  approvedOpeningStatuses: [],
  approval: null,
  commit: null,
  blockers: [
    { code: 'ROWS_NOT_VALID', message: 'Some rows are not valid today.', decision: 'OQ-49' },
    {
      code: 'OPENING_STATUSES_NOT_APPROVED',
      message: 'No opening-stock status is approved.',
      decision: 'OQ-47',
    },
    {
      code: 'PHYSICAL_COUNT_QUANTITIES_NOT_RECORDED',
      message: 'The system holds no counted quantities.',
      decision: 'H-08',
    },
  ],
  exceptions: [
    {
      rowNumber: 4,
      category: 'WARNING',
      previewStatus: 'WARNING',
      currentStatus: 'WARNING',
      changedSincePreview: false,
      source: src('CH-0002'),
      issues: [
        {
          field: 'acquisition_cost',
          code: 'COST_MISSING',
          severity: 'WARNING',
          message: 'No acquisition cost.',
        },
        {
          field: '',
          code: 'WARNING_NOT_ACCEPTED',
          severity: 'ERROR',
          message: 'No approved rule accepts warnings at commit (OQ-49).',
        },
      ],
    },
    {
      rowNumber: 5,
      category: 'BLOCKING_ERROR',
      previewStatus: 'ERROR',
      currentStatus: 'ERROR',
      changedSincePreview: false,
      source: src('=CH-EXIST'),
      issues: [
        {
          field: 'chassis_number',
          code: 'DUPLICATE_CHASSIS_EXISTING',
          severity: 'ERROR',
          message: 'Chassis/frame CH-EXIST is already registered, "really".',
        },
      ],
    },
    {
      rowNumber: 6,
      category: 'MANUAL_RECONCILIATION',
      previewStatus: 'VALID',
      currentStatus: 'VALID',
      changedSincePreview: true,
      source: src('CH-0006'),
      issues: [
        {
          field: '',
          code: 'CHANGED_SINCE_PREVIEW',
          severity: 'ERROR',
          message: 'The catalogue changed since the preview.',
        },
      ],
    },
  ],
  ...over,
})

const err = (status: number, code: string, message = 'Refused.') =>
  Response.json({ error: { code, message, requestId: 'req-9' } }, { status })

function apiMock(answer: () => Response = () => Response.json({ data: preflight() })) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const p = url.pathname
    if ((init?.method ?? 'GET') !== 'GET') return Promise.resolve(err(405, 'METHOD_NOT_ALLOWED'))
    if (p === '/api/v1/opening-stock-imports/b1/preflight') return Promise.resolve(answer())
    if (p.endsWith('/health'))
      return Promise.resolve(
        Response.json({ data: { status: 'ok', service: 'api', version: 'v', time: 'x' } }),
      )
    if (p.endsWith('/ready'))
      return Promise.resolve(
        Response.json({ data: { status: 'ready', checks: { database: 'up', migrations: 'up' } } }),
      )
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}
type Spy = ReturnType<typeof apiMock>
const preflightCalls = (spy: Spy) =>
  spy.mock.calls.filter(([u]) => (u as string).includes('/preflight')).map(([, init]) => init)

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('opening stock preflight API contract (H-08)', () => {
  it('matches the server field for field (docs/openapi) and is a GET only', () => {
    const specPath = path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json')
    const doc = JSON.parse(readFileSync(specPath, 'utf8')) as {
      definitions: Record<string, { properties: Record<string, unknown> }>
      paths: Record<string, Record<string, unknown>>
    }
    const keys = (name: string) => Object.keys(doc.definitions[name]?.properties ?? {}).sort()
    const p = preflight()
    expect(Object.keys(p).sort()).toEqual(keys('OpeningStockPreflight'))
    expect(Object.keys(p.source).sort()).toEqual(keys('OpeningStockPreflightSource'))
    expect(Object.keys(p.summary).sort()).toEqual(keys('OpeningStockPreflightSummary'))
    expect(Object.keys(p.exceptions[0] ?? {}).sort()).toEqual(keys('OpeningStockPreflightRow'))
    expect(Object.keys(p.quantities[0] ?? {}).sort()).toEqual(keys('OpeningStockPreflightQuantity'))
    expect(Object.keys(p.blockers[0] ?? {}).sort()).toEqual(keys('OpeningStockPreflightBlocker'))
    expect(Object.keys(p.physicalCount).sort()).toEqual(keys('OpeningStockPreflightPhysicalCount'))
    expect(Object.keys(p.issueCounts[0] ?? {}).sort()).toEqual(
      keys('OpeningStockPreflightIssueCount'),
    )
    expect(Object.keys(doc.paths['/opening-stock-imports/{id}/preflight'] ?? {})).toEqual(['get'])
    // The module reads only: no approve/commit/upload call exists here.
    expect(Object.keys(api).some((k) => /commit|approve|upload|post/i.test(k))).toBe(false)
  })

  it('refuses an answer that does not state it changed nothing, or whose counts disagree', () => {
    expect(api.preflightSchema.safeParse(preflight()).success).toBe(true)
    expect(api.preflightSchema.safeParse({ ...preflight(), nonMutating: false }).success).toBe(
      false,
    )
    const p = preflight()
    expect(
      api.preflightSchema.safeParse({ ...p, summary: { ...p.summary, validRows: 2 } }).success,
    ).toBe(false)
  })

  it('builds the exceptions CSV from the server answer and neutralises formulas', () => {
    const csv = api.exceptionsCsv(preflight())
    const lines = csv.trim().split('\r\n')
    expect(lines[0]).toBe(
      'batch_reference,line,category,preview_status,current_status,changed_since_preview,chassis_number,field,code,severity,message',
    )
    expect(lines).toHaveLength(5) // header + 2 issues of line 4 + 1 of line 5 + 1 of line 6
    expect(csv).toContain(`'=CH-EXIST`)
    expect(csv).not.toMatch(/,=CH-EXIST/)
    expect(csv).toContain('"Chassis/frame CH-EXIST is already registered, ""really""."')
    expect(api.csvCell('+1')).toBe(`'+1`)
    expect(api.csvCell('@x')).toBe(`'@x`)
  })
})

describe('opening stock preflight screen (H-08)', () => {
  it('shows integrity, categories, quantities beside the physical count and blockers — read only', async () => {
    const spy = apiMock()
    const user = userEvent.setup()
    renderApp('/inventory/opening-stock/b1/preflight', { me: me(PERM) })
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Reconciliation preflight' }),
    ).toBeInTheDocument()
    expect(
      await screen.findByText(
        /No inventory records were created or changed, and nothing was approved or committed\./,
      ),
    ).toBeInTheDocument()
    const checks = screen.getByRole('list', { name: 'Source file checks' })
    expect(within(checks).getAllByRole('listitem')).toHaveLength(5)
    expect(checks).toHaveTextContent('SHA-256 matches the upload: passed')

    const summary = screen.getByLabelText('Preflight summary')
    expect(within(summary).getByText('Blocking errors').nextSibling).toHaveTextContent('1')
    expect(within(summary).getByText('Manual reconciliation').nextSibling).toHaveTextContent('1')

    const blockers = screen.getByRole('list', { name: 'Blockers' })
    expect(blockers).toHaveTextContent('Pending: OQ-47')
    expect(blockers).toHaveTextContent('Pending: H-08')

    // Imported and physical quantities are separate; the physical count is never invented.
    const qTable = screen.getAllByRole('table')[0]
    if (!qTable) throw new Error('no quantities table')
    expect(
      within(qTable).getByRole('columnheader', { name: 'Imported (file)' }),
    ).toBeInTheDocument()
    expect(within(qTable).getByRole('columnheader', { name: 'Physical count' })).toBeInTheDocument()
    expect(within(qTable).getAllByText('Not recorded').length).toBeGreaterThan(0)

    // Exceptions with categories; filter by category.
    expect(screen.getAllByText('Warning — not accepted').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Needs manual reconciliation').length).toBeGreaterThan(0)
    expect(screen.getAllByText(/changed since the preview/).length).toBeGreaterThan(0)
    await user.selectOptions(screen.getByLabelText('Category'), 'BLOCKING_ERROR')
    const exTable = screen.getByRole('table', { name: 'Records needing attention' })
    expect(within(exTable).getAllByRole('row')).toHaveLength(2) // header + line 5
    expect(exTable).toHaveTextContent('Line 5')

    // No approval or commit action anywhere on this page; only GET requests were made.
    expect(screen.queryByRole('button', { name: /approve|commit|import stock/i })).toBeNull()
    expect(screen.queryByRole('link', { name: /approve|commit/i })).toBeNull()
    for (const init of preflightCalls(spy)) expect(init?.method ?? 'GET').toBe('GET')
  })

  it('downloads the exceptions report and runs again only when asked', async () => {
    const spy = apiMock()
    const create = vi.fn(() => 'blob:x')
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: create, revokeObjectURL: vi.fn() }))
    const user = userEvent.setup()
    renderApp('/inventory/opening-stock/b1/preflight', { me: me(PERM) })
    await user.click(await screen.findByRole('button', { name: /Download exceptions/ }))
    expect(create).toHaveBeenCalledTimes(1)
    expect(preflightCalls(spy)).toHaveLength(1)
    await user.click(screen.getByRole('button', { name: /Run again/ }))
    await waitFor(() => {
      expect(preflightCalls(spy)).toHaveLength(2)
    })
  })

  it('shows the empty state when every record passes, and the signed count reference after approval', async () => {
    const p = preflight()
    apiMock(() =>
      Response.json({
        data: {
          ...p,
          summary: {
            ...p.summary,
            totalRows: 1,
            validRows: 1,
            warningRows: 0,
            blockingErrorRows: 0,
            manualReconciliationRows: 0,
            changedSincePreviewRows: 0,
            duplicateRows: 0,
          },
          exceptions: [],
          issueCounts: [],
          quantities: [{ ...p.quantities[0], fileRows: 1, validRows: 1 }],
          physicalCount: {
            status: 'REFERENCE_ONLY',
            reference: 'PC-TEST-1',
            countDate: '2026-11-15',
            message:
              'The approval names a signed physical count, but its quantities are not recorded.',
          },
          blockers: [p.blockers[2]],
        },
      }),
    )
    renderApp('/inventory/opening-stock/b1/preflight', { me: me(PERM) })
    expect(await screen.findByText('No exceptions')).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('signed count PC-TEST-1 of 2026-11-15')
    expect(screen.queryByRole('button', { name: /Download exceptions/ })).toBeNull()
  })

  it('reports a failed file integrity check without row results', async () => {
    const p = preflight()
    apiMock(() =>
      Response.json({
        data: {
          ...p,
          rowsChecked: false,
          source: {
            ...p.source,
            status: 'INVALID',
            checksumMatches: false,
            templateCompatible: false,
            matchesStagedRows: false,
            message:
              'The stored file no longer matches the uploaded file (SHA-256 or size differs).',
          },
          summary: {
            totalRows: 0,
            validRows: 0,
            warningRows: 0,
            blockingErrorRows: 0,
            unverifiedRows: 0,
            manualReconciliationRows: 0,
            duplicateRows: 0,
            missingChassisRows: 0,
            changedSincePreviewRows: 0,
          },
          exceptions: [],
          quantities: [],
          issueCounts: [],
          blockers: [
            { code: 'SOURCE_INVALID', message: 'The stored file no longer matches.', decision: '' },
          ],
        },
      }),
    )
    renderApp('/inventory/opening-stock/b1/preflight', { me: me(PERM) })
    expect(await screen.findByText('Failed')).toBeInTheDocument()
    expect(screen.getByRole('list', { name: 'Source file checks' })).toHaveTextContent(
      'SHA-256 matches the upload: failed',
    )
    expect(screen.queryByLabelText('Preflight summary')).toBeNull()
    expect(screen.queryByText('Records needing attention')).toBeNull()
  })

  it('denies without an import permission and makes no preflight request', async () => {
    const spy = apiMock()
    renderApp('/inventory/opening-stock/b1/preflight', { me: me([]) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(preflightCalls(spy)).toHaveLength(0)
  })

  it('shows the API error with a retry (not found, forbidden)', async () => {
    let answer = err(404, 'NOT_FOUND', 'The import was not found.')
    apiMock(() => answer)
    const user = userEvent.setup()
    renderApp('/inventory/opening-stock/b1/preflight', { me: me(['stock.import.approve']) })
    expect(await screen.findByText('Could not run the preflight')).toBeInTheDocument()
    answer = Response.json({ data: preflight() })
    await user.click(screen.getByRole('button', { name: /try again|retry/i }))
    expect(await screen.findByText(/Read-only check/)).toBeInTheDocument()
  })
})
