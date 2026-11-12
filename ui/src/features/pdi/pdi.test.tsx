/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/pdi'
import type {
  ChecklistItem,
  Inspection,
  InspectionDetail,
  PdiOverview,
  RecordResult,
} from './api/pdi'

// Placeholder data only ("MAE-VEH-000001", "BRAKES", "Brand A"): nothing real.
// API answers are mocked; lifecycle, permissions, scope, evidence privacy and
// idempotency under concurrency are proven by the Go/PostgreSQL tests.

const t = '2026-11-11T09:00:00Z'
const loc = { id: 'l1', code: 'SR1', name: 'Showroom' }
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations: [loc] },
})
const VIEW = ['vehicle_units.view']
const INSPECTOR = [...VIEW, 'vehicle_units.pdi.record', 'attachments.upload', 'attachments.view']
const MANAGER = [...VIEW, 'vehicle_units.pdi.checklist.manage']

const item = (code: string, over: Partial<ChecklistItem> = {}): ChecklistItem => ({
  id: `ci-${code}`,
  code,
  label: code === 'BRAKES' ? 'Brakes' : 'Lights',
  description: null,
  brandId: null,
  brandName: null,
  sortOrder: 1,
  status: 'ACTIVE',
  version: 1,
  createdBy: 'u-m',
  createdAt: t,
  retiredBy: null,
  retiredAt: null,
  retireReason: null,
  ...over,
})
const checklist = [item('BRAKES'), item('LIGHTS', { sortOrder: 2 })]

const unitState = (status: PdiOverview['unit']['status'], version = 2): PdiOverview['unit'] => ({
  id: 'v1',
  unitNumber: 'MAE-VEH-000001',
  status,
  brandId: 'b1',
  locationId: 'l1',
  version,
  saleable: status === 'AVAILABLE',
})
const inspection = (over: Partial<Inspection> = {}): Inspection => ({
  id: 'p1',
  inspectionNumber: 'PDI-000001',
  vehicleUnitId: 'v1',
  unitNumber: 'MAE-VEH-000001',
  disposition: 'PASS',
  result: 'ALL_PASSED',
  fromStatus: 'RECEIVED_PENDING_PDI',
  toStatus: 'AVAILABLE',
  locationId: 'l1',
  notes: null,
  movementId: 'm1',
  inspectedBy: 'u-i',
  inspectedByName: 'Inspector',
  inspectedAt: t,
  itemCount: 2,
  failedCount: 0,
  evidenceCount: 0,
  ...over,
})
const detail = (over: Partial<InspectionDetail> = {}): InspectionDetail => ({
  ...inspection(),
  items: [
    {
      lineNo: 1,
      checklistItemId: 'ci-BRAKES',
      code: 'BRAKES',
      label: 'Brakes',
      outcome: 'PASS',
      note: null,
    },
  ],
  evidence: [],
  evidenceHidden: false,
  ...over,
})
const overview = (over: Partial<PdiOverview> = {}): PdiOverview => ({
  unit: unitState('RECEIVED_PENDING_PDI'),
  allowedDispositions: ['PASS', 'HOLD'],
  canRecord: true,
  checklist,
  inspections: [],
  ...over,
})
const vehicle = {
  id: 'v1',
  unitNumber: 'MAE-VEH-000001',
  status: 'RECEIVED_PENDING_PDI',
  brandId: 'b1',
  brandName: 'Brand A',
  modelId: 'm1',
  modelName: 'Model 1',
  variantId: 'va1',
  variantName: 'Standard',
  chassisNumber: 'CH-0001',
  chassisNormalised: 'CH-0001',
  motorSerial: null,
  batterySerial: null,
  chargerSerial: null,
  locationId: 'l1',
  locationCode: 'SR1',
  locationName: 'Showroom',
  sourceType: 'GOODS_RECEIPT',
  supplierId: 's1',
  supplierName: 'Supplier One',
  receivedDate: '2026-11-10',
  unitCost: null,
  costCurrency: null,
  costHidden: true,
  version: 2,
  createdAt: t,
  updatedAt: t,
  source: {
    type: 'GOODS_RECEIPT',
    supplierId: 's1',
    supplierName: 'Supplier One',
    purchaseOrderId: 'po1',
    purchaseOrderNumber: 'PO-000001',
    purchaseOrderLineId: 'pol1',
    goodsReceiptId: 'g1',
    goodsReceiptNumber: 'GRN-000001',
    goodsReceiptLineId: 'gl1',
    goodsReceiptUnitId: 'gu1',
    supplierReference: 'DC-1',
    receivedDate: '2026-11-10',
  },
  events: [],
}
const attachment = (id: string, locationId: string | null = 'l1') => ({
  id,
  fileName: `${id}.jpg`,
  contentType: 'image/jpeg',
  sizeBytes: 1024,
  sha256: 'ab',
  locationId,
  linked: false,
  uploadedBy: 'user-1',
  createdAt: t,
  previewable: true,
})
const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json(
    { error: { code, message: 'Refused by the server.', requestId: 'req-9', details } },
    { status },
  )

type Handler = (url: URL, init: RequestInit | undefined) => Response | Error | undefined

/** A fake API: GETs answer from `state`; handler overrides first. */
function apiMock(state: { overview: PdiOverview }, handler: Handler = () => undefined) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom instanceof Error) return Promise.reject(custom)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/vehicle-units/v1/pdi') return json(state.overview)
    if (p === '/api/v1/vehicle-units/v1')
      return json({ ...vehicle, status: state.overview.unit.status })
    if (p === '/api/v1/vehicle-units') return json(page([vehicle]))
    if (p.startsWith('/api/v1/pdi-inspections/')) return json(detail())
    if (p === '/api/v1/pdi-checklist-items') return json({ ...page(checklist), limit: 50 })
    if (p === '/api/v1/attachments/limits')
      return json({
        maxBytes: 10_485_760,
        allowedTypes: [{ contentType: 'image/jpeg', extensions: ['.jpg'], previewable: true }],
      })
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}
type Spy = ReturnType<typeof apiMock>
const bodyOf = (init: RequestInit | undefined): unknown =>
  typeof init?.body === 'string' ? JSON.parse(init.body) : undefined
const posts = (spy: Spy, p: string) =>
  spy.mock.calls.filter(
    ([u, init]) =>
      init?.method === 'POST' && new URL(u as string, 'http://x').pathname === `/api/v1${p}`,
  ) as [string, RequestInit][]
const keyOf = (c: [string, RequestInit]) => new Headers(c[1].headers).get('Idempotency-Key')

/** XMLHttpRequest stand-in for uploads: records the URL, answers with the next queued response. */
class FakeXHR {
  static responses: { status: number; body: unknown }[] = []
  static urls: string[] = []
  status = 0
  responseText = ''
  withCredentials = false
  upload: { onprogress: ((e: ProgressEvent) => void) | null } = { onprogress: null }
  onload: (() => void) | null = null
  onerror: (() => void) | null = null
  onabort: (() => void) | null = null
  open(_m: string, url: string) {
    FakeXHR.urls.push(url)
  }
  setRequestHeader() {}
  getResponseHeader() {
    return 'req-x'
  }
  abort() {
    this.onabort?.()
  }
  send() {
    const next = FakeXHR.responses.shift() ?? { status: 500, body: {} }
    setTimeout(() => {
      this.status = next.status
      this.responseText = JSON.stringify(next.body)
      this.onload?.()
    }, 0)
  }
}

beforeEach(() => {
  FakeXHR.responses = []
  FakeXHR.urls = []
  vi.stubGlobal('XMLHttpRequest', FakeXHR)
})
afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const recorded = (o: PdiOverview, d: Partial<InspectionDetail>): RecordResult => {
  const insp = detail(d)
  return { inspection: insp, unit: unitState(insp.toStatus, o.unit.version + 1) }
}

async function answer(
  user: ReturnType<typeof userEvent.setup>,
  label: string,
  outcome: 'Pass' | 'Fail',
) {
  const group = await screen.findByRole('radiogroup', { name: `${label} outcome` })
  await user.click(within(group).getByRole('radio', { name: outcome }))
}

async function choose(
  user: ReturnType<typeof userEvent.setup>,
  disposition: 'Pass' | 'Hold' | 'Reject',
) {
  const group = screen.getByRole('group', { name: 'Disposition' })
  await user.click(within(group).getByRole('radio', { name: new RegExp(`^${disposition}`) }))
}

describe('PDI API client', () => {
  it('matches the server contract field for field (docs/openapi)', () => {
    const spec = JSON.parse(
      readFileSync(
        path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json'),
        'utf8',
      ),
    ) as {
      definitions: Record<string, { properties: Record<string, { enum?: string[] }> }>
      paths: Record<
        string,
        Record<string, { parameters?: { name: string; in: string; required?: boolean }[] }>
      >
    }
    const keys = (n: string) => Object.keys(spec.definitions[n]?.properties ?? {}).sort()
    expect(Object.keys(overview()).sort()).toEqual(keys('PDIOverview'))
    expect(Object.keys(unitState('AVAILABLE')).sort()).toEqual(keys('PDIUnitState'))
    expect(Object.keys(item('BRAKES')).sort()).toEqual(keys('PDIChecklistItem'))
    expect(Object.keys(inspection()).sort()).toEqual(keys('PDIInspection'))
    expect(Object.keys(detail()).sort()).toEqual(keys('PDIInspectionDetail'))
    expect(Object.keys(detail().items[0] ?? {}).sort()).toEqual(keys('PDIItemOutcome'))
    expect(keys('PDIEvidence')).toEqual([
      'attachmentId',
      'contentType',
      'originalName',
      'sizeBytes',
    ])
    expect(keys('PDIRecordResult')).toEqual(['inspection', 'unit'])
    expect(keys('PDIInspectionCreate')).toEqual([
      'attachmentIds',
      'disposition',
      'items',
      'notes',
      'version',
    ])
    expect(keys('PDIItemOutcomeInput')).toEqual(['checklistItemId', 'note', 'outcome'])
    expect(keys('PDIChecklistItemCreate')).toEqual([
      'brandId',
      'code',
      'description',
      'label',
      'sortOrder',
    ])
    expect(keys('PDIChecklistItemRetire')).toEqual(['reason', 'version'])
    expect(spec.definitions.PDIInspection?.properties.disposition?.enum).toEqual(
      api.dispositionSchema.options,
    )
    expect(spec.definitions.PDIInspection?.properties.result?.enum).toEqual(
      api.resultSchema.options,
    )
    expect(spec.definitions.PDIItemOutcome?.properties.outcome?.enum).toEqual(
      api.outcomeSchema.options,
    )
    expect(spec.paths['/vehicle-units/{id}/pdi']?.post?.parameters).toContainEqual(
      expect.objectContaining({ name: 'Idempotency-Key', in: 'header', required: true }),
    )
    expect(spec.paths['/vehicle-units']?.get?.parameters).toContainEqual(
      expect.objectContaining({ name: 'saleable', in: 'query' }),
    )
  })

  it('refuses an answer for another unit or disposition', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      Response.json({
        data: recorded(overview(), { disposition: 'HOLD', toStatus: 'QUALITY_HOLD' }),
      }),
    )
    const a = api.newInspectionAction('v1', {
      version: 2,
      disposition: 'PASS',
      items: [],
      attachmentIds: [],
    })
    await expect(api.sendInspection(a)).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    expect(Object.isFrozen(a) && Object.isFrozen(a.body)).toBe(true)
  })
})

describe('PDI pages', () => {
  it('lists vehicles awaiting PDI from the vehicle unit API', async () => {
    const spy = apiMock({ overview: overview() })
    renderApp('/procurement/pdi', { me: me(VIEW) })
    expect((await screen.findAllByRole('link', { name: 'MAE-VEH-000001' })).length).toBeGreaterThan(
      0,
    )
    const listCall = spy.mock.calls.find(
      ([u]) => new URL(u as string, 'http://x').pathname === '/api/v1/vehicle-units',
    )
    expect(new URL(listCall?.[0] as string, 'http://x').searchParams.get('status')).toBe(
      'RECEIVED_PENDING_PDI',
    )
    expect(screen.getByRole('link', { name: 'Vehicle Inspections (PDI)' })).toBeInTheDocument()
  })

  it('shows access denied and calls no PDI endpoint without permission', async () => {
    const spy = apiMock({ overview: overview() })
    renderApp('/procurement/pdi/v1', { me: me([]) })
    expect(
      await screen.findByText(/You do not have permission to view vehicle inspections/),
    ).toBeInTheDocument()
    // The denied page keeps exactly one page heading (regression: browser crawl 2026-11-11).
    expect(screen.getAllByRole('heading', { level: 1 }).map((h) => h.textContent)).toEqual([
      'Vehicle inspection',
    ])
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/pdi'))).toBe(false)
  })

  it('renders the checklist and blocks an incomplete inspection before sending', async () => {
    const user = userEvent.setup()
    const spy = apiMock({ overview: overview() })
    renderApp('/procurement/pdi/v1', { me: me(INSPECTOR) })
    expect(
      await screen.findByRole('heading', { level: 1, name: 'MAE-VEH-000001' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/Not saleable\./)).toBeInTheDocument()
    expect(screen.getByText('Brand A · Model 1 · Standard')).toBeInTheDocument()
    const groups = screen
      .getAllByRole('radiogroup')
      .map((g) => g.getAttribute('aria-label'))
      .filter((n) => n?.endsWith(' outcome'))
    expect(groups).toEqual(['Brakes outcome', 'Lights outcome'])
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    expect(await screen.findByText('Brakes: Choose Pass or Fail.')).toBeInTheDocument()
    expect(screen.getByText('Choose a disposition.')).toBeInTheDocument()
    // A failed item needs a note, and a failed item cannot pass.
    await answer(user, 'Brakes', 'Fail')
    await answer(user, 'Lights', 'Pass')
    await choose(user, 'Pass')
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    expect(await screen.findByText('Describe what failed.')).toBeInTheDocument()
    expect(screen.getByText(/A vehicle with a failed item cannot pass/)).toBeInTheDocument()
    // A hold needs notes and evidence.
    await choose(user, 'Hold')
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    expect(await screen.findByText(/notes are required for a hold/)).toBeInTheDocument()
    expect(screen.getByText(/Add at least one photo or document/)).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(posts(spy, '/vehicle-units/v1/pdi')).toHaveLength(0)
  })

  it('records a pass after confirmation and shows the server result only then', async () => {
    const user = userEvent.setup()
    const state = { overview: overview() }
    let release: (() => void) | undefined
    const spy = apiMock(state)
    const original = spy.getMockImplementation()
    spy.mockImplementation((input, init) => {
      const url = new URL(input as string, 'http://localhost')
      if (init?.method === 'POST' && url.pathname === '/api/v1/vehicle-units/v1/pdi') {
        return new Promise((resolve) => {
          release = () => {
            const r = recorded(state.overview, {})
            state.overview = overview({
              unit: r.unit,
              allowedDispositions: [],
              inspections: [r.inspection],
            })
            resolve(Response.json({ data: r }, { status: 201 }))
          }
        })
      }
      return original?.(input, init) as Promise<Response>
    })
    renderApp('/procurement/pdi/v1', { me: me(INSPECTOR) })
    await answer(user, 'Brakes', 'Pass')
    await answer(user, 'Lights', 'Pass')
    await choose(user, 'Pass')
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    const dialog = await screen.findByRole('dialog', { name: 'Pass MAE-VEH-000001?' })
    expect(within(dialog).getByText(/becomes available \(saleable\) stock/)).toBeInTheDocument()
    expect(within(dialog).getByText('2 items, 0 failed')).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Record pass' }))
    expect(within(dialog).getByRole('button', { name: 'Recording…' })).toBeDisabled()
    // Not shown as available before the server confirms.
    expect(screen.queryByText(/Saleable\./)).not.toBeInTheDocument()
    const [call] = posts(spy, '/vehicle-units/v1/pdi')
    expect(call && bodyOf(call[1])).toEqual({
      version: 2,
      disposition: 'PASS',
      items: [
        { checklistItemId: 'ci-BRAKES', outcome: 'PASS' },
        { checklistItemId: 'ci-LIGHTS', outcome: 'PASS' },
      ],
      attachmentIds: [],
    })
    expect(call && keyOf(call)).toMatch(/^[0-9a-f-]{36}$/)
    release?.()
    expect(await screen.findByText(/PDI-000001 recorded\. The vehicle is now/)).toBeInTheDocument()
    expect(await screen.findByText(/Saleable\./)).toBeInTheDocument()
    expect(screen.getByText(/No inspection can be recorded/)).toBeInTheDocument()
    expect(
      within(screen.getByRole('list', { name: 'Inspections' })).getByText('PDI-000001'),
    ).toBeInTheDocument()
  })

  it('holds a vehicle with a defect note and evidence uploaded for its location', async () => {
    const user = userEvent.setup()
    const state = { overview: overview() }
    const spy = apiMock(state, (url, init) => {
      if (init?.method === 'POST' && url.pathname === '/api/v1/vehicle-units/v1/pdi') {
        const r = recorded(state.overview, {
          disposition: 'HOLD',
          result: 'HAS_FAILURES',
          toStatus: 'QUALITY_HOLD',
          notes: 'Scratched panel',
          failedCount: 1,
          evidenceCount: 1,
        })
        state.overview = overview({
          unit: r.unit,
          allowedDispositions: ['REJECT'],
          inspections: [r.inspection],
        })
        return Response.json({ data: r }, { status: 201 })
      }
      return undefined
    })
    FakeXHR.responses.push({ status: 201, body: { data: attachment('a1') } })
    renderApp('/procurement/pdi/v1', { me: me(INSPECTOR) })
    await answer(user, 'Brakes', 'Fail')
    await user.type(screen.getByLabelText(/Defect note — Brakes/), 'Brake lever loose')
    await answer(user, 'Lights', 'Pass')
    await user.type(screen.getByLabelText(/^Defect notes/), 'Scratched panel')
    await user.upload(
      await screen.findByLabelText('Choose files to upload'),
      new File(['x'], 'photo.jpg', { type: 'image/jpeg' }),
    )
    expect(
      await screen.findByRole('list', { name: 'Evidence for this inspection' }),
    ).toHaveTextContent('a1.jpg')
    expect(FakeXHR.urls[0]).toContain('/attachments?locationId=l1')
    await choose(user, 'Hold')
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    const dialog = await screen.findByRole('dialog', { name: 'Hold MAE-VEH-000001?' })
    expect(
      within(dialog).getByText(/not saleable and cannot be reserved or sold/),
    ).toBeInTheDocument()
    expect(within(dialog).getByText(/Brakes: Brake lever loose/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Put on hold' }))
    expect(await screen.findByText(/PDI-000001 recorded\. The vehicle is now/)).toBeInTheDocument()
    const [call] = posts(spy, '/vehicle-units/v1/pdi')
    expect(call && bodyOf(call[1])).toEqual({
      version: 2,
      disposition: 'HOLD',
      items: [
        { checklistItemId: 'ci-BRAKES', outcome: 'FAIL', note: 'Brake lever loose' },
        { checklistItemId: 'ci-LIGHTS', outcome: 'PASS' },
      ],
      notes: 'Scratched panel',
      attachmentIds: ['a1'],
    })
    expect(await screen.findByText(/on quality hold after a failed inspection/)).toBeInTheDocument()
  })

  it('offers only Reject for a held vehicle and keeps the upload failure visible', async () => {
    const user = userEvent.setup()
    const state = {
      overview: overview({
        unit: unitState('QUALITY_HOLD', 3),
        allowedDispositions: ['REJECT'],
        inspections: [
          inspection({ disposition: 'HOLD', toStatus: 'QUALITY_HOLD', result: 'HAS_FAILURES' }),
        ],
      }),
    }
    const spy = apiMock(state, (url, init) => {
      if (init?.method === 'POST' && url.pathname === '/api/v1/vehicle-units/v1/pdi') {
        const r = recorded(state.overview, {
          disposition: 'REJECT',
          fromStatus: 'QUALITY_HOLD',
          toStatus: 'REJECTED',
          result: 'HAS_FAILURES',
        })
        return Response.json({ data: r }, { status: 201 })
      }
      return undefined
    })
    FakeXHR.responses.push(
      {
        status: 413,
        body: {
          error: {
            code: 'FILE_TOO_LARGE',
            message: 'The file is larger than 10 MB.',
            requestId: 'r1',
          },
        },
      },
      { status: 201, body: { data: attachment('a2') } },
    )
    renderApp('/procurement/pdi/v1', { me: me(INSPECTOR) })
    const group = await screen.findByRole('group', { name: 'Disposition' })
    expect(within(group).getAllByRole('radio')).toHaveLength(1)
    expect(within(group).getByRole('radio', { name: /^Reject/ })).toBeChecked()
    expect(screen.getByText(/can only be rejected here/)).toBeInTheDocument()
    const input = await screen.findByLabelText('Choose files to upload')
    await user.upload(input, new File(['x'], 'huge.jpg', { type: 'image/jpeg' }))
    expect(await screen.findByText('The file is larger than 10 MB.')).toBeInTheDocument()
    expect(
      screen.queryByRole('list', { name: 'Evidence for this inspection' }),
    ).not.toBeInTheDocument()
    await answer(user, 'Brakes', 'Fail')
    await user.type(screen.getByLabelText(/Defect note — Brakes/), 'Cracked')
    await answer(user, 'Lights', 'Pass')
    await user.type(screen.getByLabelText(/^Defect notes/), 'Frame cracked')
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    expect(await screen.findByText(/Add at least one photo or document/)).toBeInTheDocument()
    await user.upload(input, new File(['x'], 'ok.jpg', { type: 'image/jpeg' }))
    expect(
      await screen.findByRole('list', { name: 'Evidence for this inspection' }),
    ).toHaveTextContent('a2.jpg')
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    const dialog = await screen.findByRole('dialog', { name: 'Reject MAE-VEH-000001?' })
    await user.click(within(dialog).getByRole('button', { name: 'Reject vehicle' }))
    await waitFor(() => {
      expect(posts(spy, '/vehicle-units/v1/pdi')).toHaveLength(1)
    })
    expect(bodyOf(posts(spy, '/vehicle-units/v1/pdi')[0]?.[1])).toMatchObject({
      version: 3,
      disposition: 'REJECT',
      attachmentIds: ['a2'],
    })
  })

  it('maps server validation errors to the form and refuses conflicts without success', async () => {
    const user = userEvent.setup()
    let answerWith = err(422, 'VALIDATION_FAILED', [
      { field: 'items[1].note', message: 'is too long' },
    ])
    const spy = apiMock({ overview: overview() }, (url, init) =>
      init?.method === 'POST' && url.pathname === '/api/v1/vehicle-units/v1/pdi'
        ? answerWith
        : undefined,
    )
    renderApp('/procurement/pdi/v1', { me: me(INSPECTOR) })
    await answer(user, 'Brakes', 'Pass')
    await answer(user, 'Lights', 'Pass')
    await choose(user, 'Pass')
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    let dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Record pass' }))
    expect(await within(dialog).findByText('Lights — note: is too long')).toBeInTheDocument()
    expect(within(dialog).getByText(/Reference: req-9/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Back to the form' }))
    expect(await screen.findByText('is too long')).toBeInTheDocument()
    // Stale version: explained, the overview is reloaded, nothing shown as recorded.
    answerWith = err(409, 'VERSION_CONFLICT')
    const before = spy.mock.calls.filter(
      ([u, i]) => !i?.method && (u as string).endsWith('/vehicle-units/v1/pdi'),
    ).length
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Record pass' }))
    expect(await within(dialog).findByText(/was changed by someone else/)).toBeInTheDocument()
    await waitFor(() => {
      expect(
        spy.mock.calls.filter(
          ([u, i]) =>
            (i?.method ?? 'GET') === 'GET' && (u as string).endsWith('/vehicle-units/v1/pdi'),
        ).length,
      ).toBeGreaterThan(before)
    })
    expect(screen.queryByText(/recorded\. The vehicle is now/)).not.toBeInTheDocument()
    // No checklist configured on the server.
    answerWith = err(409, 'PDI_CHECKLIST_NOT_CONFIGURED')
    await user.click(within(dialog).getByRole('button', { name: 'Back to the form' }))
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Record pass' }))
    expect(await within(dialog).findByText(/cannot pass PDI/)).toBeInTheDocument()
  })

  it('retries an unknown outcome with the same key and body', async () => {
    const user = userEvent.setup()
    let fail = true
    const state = { overview: overview() }
    const spy = apiMock(state, (url, init) => {
      if (init?.method === 'POST' && url.pathname === '/api/v1/vehicle-units/v1/pdi') {
        if (fail) {
          fail = false
          return new TypeError('Failed to fetch')
        }
        return Response.json({ data: recorded(state.overview, {}) })
      }
      return undefined
    })
    renderApp('/procurement/pdi/v1', { me: me(INSPECTOR) })
    await answer(user, 'Brakes', 'Pass')
    await answer(user, 'Lights', 'Pass')
    await choose(user, 'Pass')
    await user.click(screen.getByRole('button', { name: 'Review inspection' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Record pass' }))
    expect(await within(dialog).findByText(/may or may not have been recorded/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Send again unchanged' }))
    expect(await screen.findByText(/PDI-000001 recorded/)).toBeInTheDocument()
    const calls = posts(spy, '/vehicle-units/v1/pdi')
    expect(calls).toHaveLength(2)
    expect(keyOf(calls[0] as [string, RequestInit])).toBe(keyOf(calls[1] as [string, RequestInit]))
    expect(calls[0]?.[1].body).toBe(calls[1]?.[1].body)
  })

  it('is read-only without PDI permission and explains an unconfigured checklist', async () => {
    apiMock({ overview: overview({ canRecord: false, checklist: [] }) })
    renderApp('/procurement/pdi/v1', { me: me(VIEW) })
    expect(
      await screen.findByText(/Recording an inspection needs PDI permission/),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Review inspection' })).not.toBeInTheDocument()
  })

  it('disables Pass when no checklist applies but allows a hold', async () => {
    apiMock({ overview: overview({ checklist: [] }) })
    renderApp('/procurement/pdi/v1', { me: me(INSPECTOR) })
    expect(
      await screen.findByText(/checklist is not configured, so this vehicle cannot pass/),
    ).toBeInTheDocument()
    const group = screen.getByRole('group', { name: 'Disposition' })
    expect(within(group).getByRole('radio', { name: /^Pass/ })).toBeDisabled()
    expect(within(group).getByRole('radio', { name: /^Hold/ })).toBeEnabled()
  })

  it('shows not found when the server answers 404 (unknown or out-of-scope unit)', async () => {
    apiMock({ overview: overview() }, (url) =>
      url.pathname === '/api/v1/vehicle-units/v1/pdi' ? err(404, 'NOT_FOUND') : undefined,
    )
    renderApp('/procurement/pdi/v1', { me: me(INSPECTOR) })
    expect(await screen.findByText('Vehicle not found')).toBeInTheDocument()
  })

  it('shows inspection history with outcomes and keeps hidden evidence hidden', async () => {
    const user = userEvent.setup()
    apiMock(
      {
        overview: overview({
          unit: unitState('QUALITY_HOLD'),
          allowedDispositions: ['REJECT'],
          canRecord: false,
          inspections: [
            inspection({
              disposition: 'HOLD',
              toStatus: 'QUALITY_HOLD',
              result: 'HAS_FAILURES',
              notes: 'Scratch on panel',
              evidenceCount: 2,
            }),
          ],
        }),
      },
      (url) =>
        url.pathname === '/api/v1/pdi-inspections/p1'
          ? Response.json({
              data: detail({ evidenceHidden: true, evidenceCount: 2, evidence: [] }),
            })
          : undefined,
    )
    renderApp('/procurement/pdi/v1', { me: me(VIEW) })
    const list = await screen.findByRole('list', { name: 'Inspections' })
    expect(within(list).getByText('Scratch on panel')).toBeInTheDocument()
    expect(within(list).getByText('Hold')).toBeInTheDocument()
    await user.click(within(list).getByRole('button', { name: 'Show details' }))
    expect(
      await within(list).findByText(/2 files — viewing evidence needs attachment permission/),
    ).toBeInTheDocument()
    expect(within(list).getByRole('button', { name: 'Hide details' })).toHaveAttribute(
      'aria-expanded',
      'true',
    )
  })

  it('lets a checklist manager add and retire items', async () => {
    const user = userEvent.setup()
    const spy = apiMock({ overview: overview() }, (url, init) => {
      if (init?.method === 'POST' && url.pathname === '/api/v1/pdi-checklist-items')
        return Response.json({ data: item('TYRES') }, { status: 201 })
      if (
        init?.method === 'POST' &&
        url.pathname === '/api/v1/pdi-checklist-items/ci-BRAKES/retire'
      )
        return Response.json({
          data: item('BRAKES', {
            status: 'RETIRED',
            version: 2,
            retiredBy: 'u',
            retiredAt: t,
            retireReason: 'x',
          }),
        })
      return undefined
    })
    renderApp('/procurement/pdi/checklist', { me: me(MANAGER) })
    expect(await screen.findByRole('table', { name: 'PDI checklist items' })).toBeInTheDocument()
    await user.click(screen.getAllByRole('button', { name: 'Add item' })[0] as HTMLElement)
    const dialog = await screen.findByRole('dialog', { name: 'Add checklist item' })
    await user.click(within(dialog).getByRole('button', { name: 'Add item' }))
    expect(await within(dialog).findByText(/2–40 characters/)).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText(/^Code/), 'tyres')
    await user.type(within(dialog).getByLabelText(/^Label/), 'Tyres')
    await user.click(within(dialog).getByRole('button', { name: 'Add item' }))
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
    expect(bodyOf(posts(spy, '/pdi-checklist-items')[0]?.[1])).toEqual({
      code: 'TYRES',
      label: 'Tyres',
    })
    await user.click(screen.getAllByRole('button', { name: 'Retire BRAKES' })[0] as HTMLElement)
    const retire = await screen.findByRole('dialog', { name: 'Retire BRAKES?' })
    await user.type(within(retire).getByLabelText(/^Reason/), 'Merged')
    await user.click(within(retire).getByRole('button', { name: 'Retire item' }))
    await waitFor(() => {
      expect(bodyOf(posts(spy, '/pdi-checklist-items/ci-BRAKES/retire')[0]?.[1])).toEqual({
        version: 1,
        reason: 'Merged',
      })
    })
  })
})
