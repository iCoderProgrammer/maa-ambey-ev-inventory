/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { act, cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/openingStock'
import type { ImportBatch, ImportRow } from './api/openingStock'

// G-06 opening stock import preview against a mocked API. Placeholder values only
// ("Brand A", "CH-0001"): nothing real, nothing imported.

const t = '2026-11-16T09:00:00Z'
const loc = { id: 'l1', code: 'L1', name: 'Location One' }
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations: [loc] },
})
const PERM = ['stock.import.preview']

const spec: api.Specification = {
  templateVersion: 'OS-IMPORT-1',
  format: 'CSV, UTF-8, comma-separated. XLSX is not accepted.',
  firstLine: '#template:OS-IMPORT-1',
  maxBytes: 5 << 20,
  maxRows: 5000,
  previewTtlHours: 168,
  columns: [
    { name: 'brand', required: true, maxLength: 120, format: 'text', description: 'Brand name.' },
    {
      name: 'chassis_number',
      required: true,
      maxLength: 100,
      format: 'identifier',
      description: 'Chassis.',
    },
    { name: 'notes', required: false, maxLength: 300, format: 'text', description: 'Notes.' },
  ],
  stockStatuses: ['AVAILABLE', 'DISPLAY'],
  approvedOpeningStatuses: [],
  businessDate: '2026-11-16',
  timeZone: 'Asia/Kolkata',
  notes: ['A preview creates no stock.'],
}

const batch = (over: Partial<ImportBatch> = {}): ImportBatch => ({
  id: 'b1',
  reference: 'OSI-000001',
  status: 'PREVIEWED',
  templateVersion: 'OS-IMPORT-1',
  fileName: 'stock.csv',
  fileSha256: 'a'.repeat(64),
  fileSizeBytes: 2048,
  fileAttachmentId: 'att1',
  uploadedById: 'user-1',
  uploadedByName: 'Test User',
  uploadedAt: t,
  businessDate: '2026-11-16',
  expiresAt: '2026-11-23T09:00:00Z',
  expired: false,
  summary: {
    totalRows: 3,
    validRows: 1,
    warningRows: 0,
    errorRows: 1,
    blockedRows: 1,
    canProceedRows: 1,
    duplicateRows: 1,
    unresolvedRows: 1,
    skippedBlankRows: 0,
  },
  fileIssues: [],
  commitAvailable: false,
  approval: null,
  commit: null,
  ...over,
})

const norm = {
  brandId: null,
  modelId: null,
  variantId: null,
  colourId: null,
  locationId: null,
  supplierId: null,
  chassisNumber: null,
  chassisNormalised: null,
  motorNormalised: null,
  batteryNormalised: null,
  chargerNormalised: null,
  stockStatus: null,
  receivedDate: null,
  acquisitionCost: null,
}
const src = (chassis: string) => ({
  brand: 'Brand A',
  model_code: 'MA',
  variant_code: 'STD',
  chassis_number: chassis,
  location_code: 'L1',
  stock_status: 'AVAILABLE',
  received_date: '15/01/2026',
})
const rows: ImportRow[] = [
  {
    rowNumber: 3,
    status: 'VALID',
    canProceed: true,
    duplicate: false,
    unresolved: false,
    source: src('CH-0001'),
    normalised: { ...norm, chassisNumber: 'CH-0001', chassisNormalised: 'CH-0001' },
    resolved: { brand: { id: 'b', name: 'Brand A' } },
    issues: [],
  },
  {
    rowNumber: 4,
    status: 'ERROR',
    canProceed: false,
    duplicate: true,
    unresolved: false,
    source: src('CH-EXIST'),
    normalised: norm,
    resolved: {},
    issues: [
      {
        field: 'chassis_number',
        code: 'DUPLICATE_CHASSIS_EXISTING',
        severity: 'ERROR',
        message: 'Chassis/frame CH-EXIST is already registered to a vehicle unit.',
      },
      {
        field: 'received_date',
        code: 'INVALID_DATE',
        severity: 'ERROR',
        message: '"15/01/2026" is not a valid date.',
        suggestion: 'Write the date as YYYY-MM-DD, e.g. 2026-01-15 if this is day-month-year.',
      },
    ],
  },
  {
    rowNumber: 5,
    status: 'BLOCKED_UNVERIFIED_MAPPING',
    canProceed: false,
    duplicate: false,
    unresolved: true,
    source: { ...src('CH-0005'), brand: 'Brand P' },
    normalised: norm,
    resolved: { brand: { id: 'p', name: 'Brand P' } },
    issues: [
      {
        field: 'brand',
        code: 'BRAND_NOT_ACTIVE',
        severity: 'BLOCKED',
        message: 'Brand P exists but is not verified and activated.',
      },
    ],
  },
]

const page = <T,>(items: T[]) => ({ items, page: 1, limit: 50, total: items.length })
const err = (status: number, code: string, message = 'Refused.') =>
  Response.json({ error: { code, message, requestId: 'req-9' } }, { status })

function apiMock(state: { batch?: ImportBatch; rows?: ImportRow[]; list?: ImportBatch[] } = {}) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = new URL(input as string, 'http://localhost')
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/opening-stock-imports/specification') return json(spec)
    if (p === '/api/v1/opening-stock-imports/template')
      return Promise.resolve(
        new Response('#template:OS-IMPORT-1\nbrand\n', { headers: { 'Content-Type': 'text/csv' } }),
      )
    if (p === '/api/v1/opening-stock-imports') return json({ ...page(state.list ?? []), limit: 10 })
    if (p === '/api/v1/opening-stock-imports/b1/rows') {
      let r = state.rows ?? rows
      const status = url.searchParams.get('status')
      if (status) r = r.filter((x) => x.status === status)
      if (url.searchParams.get('duplicate') === 'true') r = r.filter((x) => x.duplicate)
      if (url.searchParams.get('unresolved') === 'true') r = r.filter((x) => x.unresolved)
      return json(page(r))
    }
    if (p === '/api/v1/opening-stock-imports/b1') return json(state.batch ?? batch())
    if (p === '/api/v1/opening-stock-imports/nope')
      return Promise.resolve(err(404, 'NOT_FOUND', 'The import was not found.'))
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}
type Spy = ReturnType<typeof apiMock>
const callsTo = (spy: Spy, pathname: string) =>
  spy.mock.calls
    .map(([u]) => new URL(u as string, 'http://x'))
    .filter((u) => u.pathname === pathname)
    .map((u) => Object.fromEntries(u.searchParams))

/** XMLHttpRequest stand-in for uploads: answers each send with the next queued response ('network' = connection error). */
class FakeXHR {
  static responses: ({ status: number; body: unknown } | 'network')[] = []
  static headers: Record<string, string>[] = []
  static urls: string[] = []
  status = 0
  responseText = ''
  withCredentials = false
  upload: { onprogress: ((e: ProgressEvent) => void) | null } = { onprogress: null }
  onload: (() => void) | null = null
  onerror: (() => void) | null = null
  onabort: (() => void) | null = null
  private reqHeaders: Record<string, string> = {}
  open(_m: string, url: string) {
    FakeXHR.urls.push(url)
  }
  setRequestHeader(k: string, v: string) {
    this.reqHeaders[k] = v
  }
  getResponseHeader(name: string) {
    return name === 'X-Request-ID' ? 'req-42' : null
  }
  abort() {
    this.onabort?.()
  }
  send() {
    FakeXHR.headers.push(this.reqHeaders)
    const next = FakeXHR.responses.shift() ?? { status: 500, body: {} }
    setTimeout(() => {
      this.upload.onprogress?.({ lengthComputable: true, loaded: 5, total: 10 } as ProgressEvent)
      if (next === 'network') {
        this.onerror?.()
        return
      }
      this.status = next.status
      this.responseText = JSON.stringify(next.body)
      this.onload?.()
    }, 0)
  }
}

beforeEach(() => {
  FakeXHR.responses = []
  FakeXHR.headers = []
  FakeXHR.urls = []
  vi.stubGlobal('XMLHttpRequest', FakeXHR)
})
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const csv = (name = 'stock.csv', content = '#template:OS-IMPORT-1\nbrand\nBrand A\n') =>
  new File([content], name, {
    type: name.endsWith('.csv') ? 'text/csv' : 'application/octet-stream',
  })

describe('opening stock import API contract (G-06)', () => {
  it('matches the server field for field (docs/openapi) and has no commit call', () => {
    const specPath = path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json')
    const doc = JSON.parse(readFileSync(specPath, 'utf8')) as {
      definitions: Record<string, { properties: Record<string, unknown> }>
      paths: Record<string, Record<string, { parameters?: { name: string }[] }>>
    }
    const keys = (name: string) => Object.keys(doc.definitions[name]?.properties ?? {}).sort()
    expect(Object.keys(batch()).sort()).toEqual(keys('OpeningStockImportBatch'))
    expect(Object.keys(batch().summary).sort()).toEqual(keys('OpeningStockImportSummary'))
    expect(Object.keys(rows[0] ?? {}).sort()).toEqual(keys('OpeningStockImportRow'))
    expect(Object.keys(norm).sort()).toEqual(keys('OpeningStockNormalisedRow'))
    expect(Object.keys(spec).sort()).toEqual(keys('OpeningStockImportSpecification'))
    const rowParams =
      doc.paths['/opening-stock-imports/{id}/rows']?.get?.parameters?.map((p) => p.name) ?? []
    expect(rowParams.sort()).toEqual(['duplicate', 'id', 'limit', 'page', 'status', 'unresolved'])
    // Writes: the upload (G-06) and, separately, approval and commit (G-07, ./api/commit.ts).
    // No edit or delete route exists, and the preview module itself has no commit call.
    const writes = [
      '/opening-stock-imports',
      '/opening-stock-imports/{id}/approval',
      '/opening-stock-imports/{id}/commit',
    ]
    const imports = Object.keys(doc.paths).filter((p) => p.startsWith('/opening-stock-imports'))
    for (const p of imports)
      for (const method of Object.keys(doc.paths[p] ?? {}))
        expect(method === 'get' || (method === 'post' && writes.includes(p))).toBe(true)
    expect(Object.keys(api).some((k) => /commit/i.test(k))).toBe(false)
  })

  it('sends the file with its idempotency key and refuses a batch that offers a commit', async () => {
    FakeXHR.responses.push({ status: 201, body: { data: batch() } })
    const progress: number[] = []
    const b = await api.uploadImport(csv(), 'osi-key-0001', { onProgress: (f) => progress.push(f) })
    expect(b.reference).toBe('OSI-000001')
    expect(FakeXHR.headers[0]?.['Idempotency-Key']).toBe('osi-key-0001')
    expect(FakeXHR.urls[0]).toMatch(/\/api\/v1\/opening-stock-imports$/)
    expect(progress).toEqual([0.5])
    FakeXHR.responses.push({ status: 201, body: { data: { ...batch(), commitAvailable: true } } })
    await expect(api.uploadImport(csv(), 'osi-key-0002')).rejects.toMatchObject({
      code: 'INVALID_RESPONSE',
    })
  })
})

describe('opening stock import wizard (G-06)', () => {
  it('shows the template rules, downloads the template and lists no previews yet', async () => {
    const spy = apiMock()
    const create = vi.fn(() => 'blob:x')
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: create, revokeObjectURL: vi.fn() }))
    const user = userEvent.setup()
    renderApp('/inventory/opening-stock', { me: me(PERM) })
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Opening Stock Import' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('never creates or changes stock')
    expect(await screen.findByText(/OS-IMPORT-1/)).toBeInTheDocument()
    expect(screen.getByRole('list', { name: 'Import steps' })).toHaveTextContent('1. Template')
    await user.click(screen.getByText('Columns (3)'))
    expect(screen.getByRole('cell', { name: 'chassis_number' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Download template/ }))
    await waitFor(() => {
      expect(create).toHaveBeenCalled()
    })
    expect(callsTo(spy, '/api/v1/opening-stock-imports/template')).toHaveLength(1)
    expect(await screen.findByText('No previews yet')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /commit|import stock/i })).toBeNull()
  })

  it('uploads a CSV with progress and opens its preview', async () => {
    apiMock()
    FakeXHR.responses.push({ status: 201, body: { data: batch() } })
    const user = userEvent.setup()
    const { router } = renderApp('/inventory/opening-stock', { me: me(PERM) })
    await screen.findByText(/OS-IMPORT-1/)
    await user.upload(screen.getByLabelText('Choose the CSV file'), csv())
    expect(screen.getByText('stock.csv')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Upload and validate' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/inventory/opening-stock/b1')
    })
    expect(FakeXHR.headers[0]?.['Idempotency-Key']).toMatch(/^osi-/)
    expect(
      await screen.findByText(
        /Preview completed\. No inventory records have been created or modified\./,
      ),
    ).toBeInTheDocument()
  })

  it('refuses a non-CSV file before sending, and explains a server refusal', async () => {
    apiMock()
    const user = userEvent.setup({ applyAccept: false })
    renderApp('/inventory/opening-stock', { me: me(PERM) })
    await screen.findByText(/OS-IMPORT-1/)
    await user.upload(screen.getByLabelText('Choose the CSV file'), csv('stock.xlsx'))
    expect(screen.getByRole('alert')).toHaveTextContent('CSV UTF-8')
    expect(screen.queryByRole('button', { name: 'Upload and validate' })).toBeNull()
    expect(FakeXHR.headers).toHaveLength(0)

    FakeXHR.responses.push({
      status: 415,
      body: {
        error: {
          code: 'UNSUPPORTED_MEDIA_TYPE',
          message: 'Only the CSV template (.csv, UTF-8 text) is accepted.',
        },
      },
    })
    await user.upload(screen.getByLabelText('Choose the CSV file'), csv())
    await user.click(screen.getByRole('button', { name: 'Upload and validate' }))
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Nothing was stored')
    expect(alert).toHaveTextContent('Only the CSV template')
    expect(screen.getByRole('button', { name: 'Upload and validate' })).toBeDisabled()
  })

  it('retries a lost upload with the same key (never a second preview)', async () => {
    apiMock()
    FakeXHR.responses.push('network', { status: 200, body: { data: batch() } })
    const user = userEvent.setup()
    const { router } = renderApp('/inventory/opening-stock', { me: me(PERM) })
    await screen.findByText(/OS-IMPORT-1/)
    await user.upload(screen.getByLabelText('Choose the CSV file'), csv())
    await user.click(screen.getByRole('button', { name: 'Upload and validate' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot reach the server')
    await user.click(screen.getByRole('button', { name: 'Try again' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/inventory/opening-stock/b1')
    })
    expect(FakeXHR.headers).toHaveLength(2)
    expect(FakeXHR.headers[1]?.['Idempotency-Key']).toBe(FakeXHR.headers[0]?.['Idempotency-Key'])
  })

  it('is denied without stock.import.preview', async () => {
    const spy = apiMock()
    renderApp('/inventory/opening-stock', { me: me([]) })
    expect(await screen.findByText(/do not have permission/i)).toBeInTheDocument()
    expect(callsTo(spy, '/api/v1/opening-stock-imports/specification')).toEqual([])
  })
})

describe('opening stock preview (G-06)', () => {
  it('shows the summary, row results, duplicate highlights and no commit action', async () => {
    apiMock()
    renderApp('/inventory/opening-stock/b1', { me: me(PERM) })
    const summary = await screen.findByRole('list', { name: 'Import steps' })
    expect(summary).toHaveTextContent('5. Summary')
    const tiles = await screen.findByLabelText('Preview summary')
    expect(within(tiles).getByText('Total rows').nextSibling).toHaveTextContent('3')
    expect(within(tiles).getByText('Duplicate identifiers').nextSibling).toHaveTextContent('1')
    expect(within(tiles).getByText('Blocked (unverified catalogue)').nextSibling).toHaveTextContent(
      '1',
    )
    expect(screen.getByText(/1 of 3 rows could proceed/)).toBeInTheDocument()
    const table = await screen.findByRole('table', { name: 'Preview rows' })
    const line = (n: number) =>
      within(
        within(table)
          .getByText(String(n), { selector: 'span.font-mono' })
          .closest('tr') as HTMLElement,
      )
    expect(line(3).getByText('Valid')).toBeInTheDocument()
    expect(line(4).getByText('Error')).toBeInTheDocument()
    expect(line(4).getByText('CH-EXIST')).toHaveClass('text-danger')
    expect(line(4).getByText(/already registered/)).toBeInTheDocument()
    expect(line(5).getByText('Blocked — unverified catalogue')).toBeInTheDocument()
    expect(line(5).getByText(/not verified and activated/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /commit|import/i })).toBeNull()
  })

  it('filters rows on the server and explains an empty filter', async () => {
    const spy = apiMock()
    const user = userEvent.setup()
    renderApp('/inventory/opening-stock/b1', { me: me(PERM) })
    await screen.findByRole('table', { name: 'Preview rows' })
    await user.selectOptions(screen.getByLabelText('Duplicates'), 'true')
    await waitFor(() => {
      expect(callsTo(spy, '/api/v1/opening-stock-imports/b1/rows').at(-1)).toMatchObject({
        duplicate: 'true',
      })
    })
    await user.selectOptions(screen.getByLabelText('Row status'), 'WARNING')
    expect(await screen.findByText('No rows match these filters')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Clear/ }))
    await user.selectOptions(screen.getByLabelText('Catalogue mapping'), 'true')
    await waitFor(() => {
      expect(callsTo(spy, '/api/v1/opening-stock-imports/b1/rows').at(-1)).toMatchObject({
        unresolved: 'true',
      })
    })
  })

  it('keeps both filter changes made in the same frame', async () => {
    const spy = apiMock()
    const { router } = renderApp('/inventory/opening-stock/b1?duplicate=true', { me: me(PERM) })
    await screen.findByRole('table', { name: 'Preview rows' })
    // Two changes before React re-renders: clearing one filter must not undo the other.
    act(() => {
      fireEvent.change(screen.getByLabelText('Duplicates'), { target: { value: '' } })
      fireEvent.change(screen.getByLabelText('Catalogue mapping'), { target: { value: 'true' } })
    })
    await waitFor(() => {
      expect(router.state.location.search).toBe('?unresolved=true')
    })
    await waitFor(() => {
      expect(callsTo(spy, '/api/v1/opening-stock-imports/b1/rows').at(-1)).toEqual({
        unresolved: 'true',
        page: '1',
        limit: '50',
      })
    })
  })

  it('shows a row’s values as uploaded and the safe suggestion', async () => {
    apiMock()
    const user = userEvent.setup()
    renderApp('/inventory/opening-stock/b1', { me: me(PERM) })
    await screen.findByRole('table', { name: 'Preview rows' })
    await user.click(screen.getAllByRole('button', { name: 'Details of line 4' })[0] as HTMLElement)
    const dialog = await screen.findByRole('dialog', { name: 'Line 4' })
    expect(within(dialog).getByText(/2026-01-15 if this is day-month-year/)).toBeInTheDocument()
    expect(within(dialog).getByRole('cell', { name: '15/01/2026' })).toBeInTheDocument()
  })

  it('shows why a file was rejected, and an expired preview as history', async () => {
    apiMock({
      batch: batch({
        status: 'REJECTED',
        summary: {
          ...batch().summary,
          totalRows: 0,
          validRows: 0,
          errorRows: 0,
          blockedRows: 0,
          canProceedRows: 0,
          duplicateRows: 0,
          unresolvedRows: 0,
        },
        fileIssues: [
          {
            code: 'MISSING_COLUMN',
            line: 2,
            column: 'chassis_number',
            message: 'Required column "chassis_number" is missing.',
          },
        ],
      }),
    })
    renderApp('/inventory/opening-stock/b1', { me: me(PERM) })
    expect(await screen.findByText('File rejected')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Line 2: Required column "chassis_number" is missing.',
    )
    expect(screen.queryByRole('table', { name: 'Preview rows' })).toBeNull()
    cleanup()
    vi.restoreAllMocks()
    apiMock({ batch: batch({ expired: true }) })
    renderApp('/inventory/opening-stock/b1', { me: me(PERM) })
    expect(await screen.findByText(/expired and is kept as history only/)).toBeInTheDocument()
  })

  it('reports a preview that is not available', async () => {
    apiMock()
    renderApp('/inventory/opening-stock/nope', { me: me(PERM) })
    expect(await screen.findByText('Could not load this preview')).toBeInTheDocument()
  })
})
