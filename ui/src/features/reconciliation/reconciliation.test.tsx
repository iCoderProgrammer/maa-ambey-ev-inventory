/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { cleanup, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/reconciliation'
import type { Discrepancy, ReconcileReport } from './api/reconciliation'
import { describeReconcileError, valueLabel } from './lib/labels'

// E-06 stock reconciliation against a mocked API. Placeholder data only
// ("MAE-VEH-000001", "PRT-1", "SR1"): nothing real.

const at = '2026-11-16T09:00:00Z'
const loc = { id: 'l1', code: 'SR1', name: 'Showroom One' }

const statusDisc: Discrepancy = {
  reference: 'VEHICLE_STATUS_MISMATCH:u1',
  itemType: 'VEHICLE_UNIT',
  type: 'VEHICLE_STATUS_MISMATCH',
  severity: 'HIGH',
  vehicleUnit: { id: 'u1', unitNumber: 'MAE-VEH-000001', chassisNumber: 'CH-0001' },
  stockItem: null,
  brandId: 'b1',
  brandName: 'Brand A',
  locationId: 'l1',
  locationCode: 'SR1',
  locationName: 'Showroom One',
  expected: 'AVAILABLE',
  actual: 'DISPLAY',
  explanation: 'The unit’s status is not the status its latest stock movement left it in.',
  movementIds: ['mv9'],
  detectedAt: at,
}
const partsDisc: Discrepancy = {
  reference: 'PARTS_BROKEN_CHAIN:p1:l1:mv3',
  itemType: 'STOCK_ITEM',
  type: 'PARTS_BROKEN_CHAIN',
  severity: 'MEDIUM',
  vehicleUnit: null,
  stockItem: { id: 'p1', sku: 'PRT-1', name: 'Brake pad', uom: 'PCS' },
  brandId: null,
  brandName: null,
  locationId: 'l9',
  locationCode: null,
  locationName: null,
  expected: '10 PCS',
  actual: '9 PCS',
  explanation: 'A stock movement does not start from the quantity the previous movement left.',
  movementIds: ['mv2', 'mv3'],
  detectedAt: at,
}

function report(items: Discrepancy[], over: Partial<ReconcileReport> = {}): ReconcileReport {
  const bySeverity: Record<string, number> = {}
  const byType: Record<string, number> = {}
  for (const d of items) {
    bySeverity[d.severity] = (bySeverity[d.severity] ?? 0) + 1
    byType[d.type] = (byType[d.type] ?? 0) + 1
  }
  return {
    runId: 'run-1',
    status: 'COMPLETED',
    runAt: at,
    completedAt: at,
    durationMs: 12,
    coverage: { vehicleUnits: true, parts: true, allLocations: false, locationIds: ['l1'] },
    summary: {
      vehicleUnitsChecked: 7,
      partBalancesChecked: 3,
      recordsChecked: 10,
      movementsChecked: 41,
      discrepancies: items.length,
      byType,
      bySeverity,
    },
    items,
    page: 1,
    limit: 25,
    total: items.length,
    ...over,
  }
}

const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-9', details } }, { status })
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations: [loc] },
})
const RECON = ['stock.reconciliation.view', 'vehicle_units.view', 'stock.parts.view']

type Answer = (url: URL) => Response | Promise<Response>
function apiMock(answer: Answer = () => Response.json({ data: report([]) })) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = new URL(input as string, 'http://localhost')
    const p = url.pathname
    if (p === '/api/v1/stock-reconciliation') return Promise.resolve(answer(url))
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
const runs = (spy: Spy) =>
  spy.mock.calls
    .map(([u]) => new URL(u as string, 'http://x'))
    .filter((u) => u.pathname === '/api/v1/stock-reconciliation')
    .map((u) => Object.fromEntries(u.searchParams))
const writes = (spy: Spy) =>
  spy.mock.calls.filter(([, init]) => init?.method && init.method !== 'GET').map(([u]) => u)

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

async function startRun(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('button', { name: 'Run reconciliation' }))
}

describe('stock reconciliation API client (E-06)', () => {
  it('matches the server contract field for field (docs/openapi) and is read-only', () => {
    const specPath = path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json')
    const spec = JSON.parse(readFileSync(specPath, 'utf8')) as {
      definitions: Record<string, { properties: Record<string, { enum?: string[] }> }>
      paths: Record<string, Record<string, { parameters?: { name: string; enum?: string[] }[] }>>
    }
    const props = (name: string) => spec.definitions[name]?.properties ?? {}
    const keys = (name: string) => Object.keys(props(name)).sort()
    const r = report([statusDisc])
    expect(Object.keys(r).sort()).toEqual(keys('ReconciliationReport'))
    expect(Object.keys(r.summary).sort()).toEqual(keys('ReconciliationSummary'))
    expect(Object.keys(r.coverage).sort()).toEqual(keys('ReconciliationCoverage'))
    expect(Object.keys(statusDisc).sort()).toEqual(keys('ReconciliationDiscrepancy'))
    expect(Object.keys(statusDisc.vehicleUnit ?? {}).sort()).toEqual(
      keys('ReconciliationVehicleUnit'),
    )
    expect(Object.keys(partsDisc.stockItem ?? {}).sort()).toEqual(keys('ReconciliationStockItem'))
    expect(api.discrepancyTypeSchema.options).toEqual(props('ReconciliationDiscrepancy').type?.enum)
    expect(api.severitySchema.options).toEqual(props('ReconciliationDiscrepancy').severity?.enum)
    const params: (keyof api.ReconcileParams)[] = [
      'itemType',
      'type',
      'severity',
      'locationId',
      'page',
      'limit',
    ]
    const documented =
      spec.paths['/stock-reconciliation']?.get?.parameters?.map((p) => p.name) ?? []
    expect([...documented].sort()).toEqual([...params].sort())
    expect(Object.keys(spec.paths['/stock-reconciliation'] ?? {})).toEqual(['get'])
  })

  it('sends only the filters that are set and refuses a report that breaks the contract', async () => {
    const spy = apiMock()
    await api.runReconciliation({ severity: 'HIGH', locationId: '', page: 2, limit: 25 })
    expect(runs(spy).at(-1)).toEqual({ severity: 'HIGH', page: '2', limit: '25' })
    spy.mockResolvedValue(Response.json({ data: report([{ ...statusDisc, vehicleUnit: null }]) }))
    await expect(api.runReconciliation()).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    expect(writes(spy)).toEqual([])
  })

  it('maps failures to safe states', () => {
    expect(valueLabel('AVAILABLE')).toContain('AVAILABLE')
    expect(valueLabel('8 PCS')).toBe('8 PCS')
    expect(valueLabel(null)).toBe('—')
    expect(describeReconcileError(new Error('x')).retry).toBe(true)
  })
})

describe('stock reconciliation page (E-06)', () => {
  it('runs only when asked, shows the running state and a clean summary', async () => {
    let release: (r: Response) => void = () => undefined
    const spy = apiMock(
      () =>
        new Promise<Response>((resolve) => {
          release = resolve
        }),
    )
    const user = userEvent.setup()
    renderApp('/inventory/reconciliation', { me: me(RECON) })
    expect(await screen.findByRole('heading', { name: 'Stock Reconciliation' })).toBeInTheDocument()
    expect(screen.getByText('Reconciliation has not been run')).toBeInTheDocument()
    expect(runs(spy)).toEqual([])

    await startRun(user)
    expect(
      await screen.findByRole('status', { name: 'Running reconciliation' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Running/ })).toBeDisabled()
    expect(runs(spy)).toHaveLength(1)

    release(Response.json({ data: report([]) }))
    expect(await screen.findByText('No discrepancies')).toBeInTheDocument()
    expect(screen.getByText('Ledger agrees')).toBeInTheDocument()
    const summary = screen.getByText('Run summary').closest('section') as HTMLElement
    expect(within(summary).getByText('Vehicle units checked').nextSibling).toHaveTextContent('7')
    expect(within(summary).getByText('Movements checked').nextSibling).toHaveTextContent('41')
    expect(within(summary).getByText(/your 1 location/)).toBeInTheDocument()
    expect(writes(spy)).toEqual([])
  })

  it('shows discrepancies with expected and actual values and only authorised links', async () => {
    apiMock(() => Response.json({ data: report([statusDisc, partsDisc]) }))
    const user = userEvent.setup()
    renderApp('/inventory/reconciliation', { me: me(RECON) })
    await startRun(user)
    const table = await screen.findByRole('table', { name: 'Reconciliation discrepancies' })
    const rows = within(table).getAllByRole('row').slice(1)
    expect(rows).toHaveLength(2)
    const [vehicle, parts] = rows as [HTMLElement, HTMLElement]
    expect(within(vehicle).getByText('Vehicle status mismatch')).toBeInTheDocument()
    expect(within(vehicle).getByText('High severity')).toBeInTheDocument()
    expect(within(vehicle).getByRole('link', { name: 'MAE-VEH-000001' })).toHaveAttribute(
      'href',
      '/inventory/vehicles/u1',
    )
    expect(within(vehicle).getByText(/Available.*\(AVAILABLE\)/)).toBeInTheDocument()
    expect(within(vehicle).getByText(/\(DISPLAY\)/)).toBeInTheDocument()
    expect(within(vehicle).getByText('SR1 — Showroom One')).toBeInTheDocument()
    expect(within(vehicle).getByRole('link', { name: /stock movement 1 of 1/ })).toHaveAttribute(
      'href',
      '/inventory/movements/mv9',
    )
    // The SKU master is not opened without stock_items.view; an out-of-scope location is not named.
    expect(within(parts).queryByRole('link', { name: 'PRT-1' })).toBeNull()
    expect(within(parts).getByText('PRT-1')).toBeInTheDocument()
    expect(within(parts).getByText('A location not visible to you')).toBeInTheDocument()
    expect(within(parts).getByText('10 PCS')).toBeInTheDocument()
    expect(within(parts).getByText('9 PCS')).toBeInTheDocument()
    expect(within(parts).getAllByRole('link', { name: /stock movement/ })).toHaveLength(2)
    expect(screen.getByText('2 discrepancies')).toBeInTheDocument()
  })

  it('offers no action that changes stock', async () => {
    const spy = apiMock(() => Response.json({ data: report([statusDisc, partsDisc]) }))
    const user = userEvent.setup()
    renderApp('/inventory/reconciliation', { me: me([...RECON, 'stock_items.view']) })
    await startRun(user)
    await screen.findByRole('table', { name: 'Reconciliation discrepancies' })
    const main = screen.getByRole('main')
    const labels = within(main)
      .getAllByRole('button')
      .map((b) => b.textContent.trim())
    for (const l of labels) expect(l).not.toMatch(/repair|fix|adjust|correct|delete|apply|post/i)
    await user.click(screen.getByRole('button', { name: 'Run again' }))
    await waitFor(() => {
      expect(runs(spy)).toHaveLength(2)
    })
    expect(writes(spy)).toEqual([])
  })

  it('shows a run from earlier in the session as that run, and a new request always runs again', async () => {
    let n = 0
    const spy = apiMock(() => {
      n += 1
      return Response.json({ data: report([statusDisc], { runId: `run-${String(n)}` }) })
    })
    const user = userEvent.setup()
    const { router } = renderApp('/inventory/reconciliation', { me: me(RECON) })
    await startRun(user)
    expect(await screen.findByText('Run run-1')).toBeInTheDocument()
    await router.navigate('/inventory/movements')
    await router.navigate('/inventory/reconciliation')
    // The earlier run, with its own time — not "has not been run", and not a new run.
    expect(await screen.findByText('Run run-1')).toBeInTheDocument()
    expect(screen.queryByText('Reconciliation has not been run')).toBeNull()
    expect(screen.getByText(/^Last run /)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Run again' }))
    expect(await screen.findByText('Run run-2')).toBeInTheDocument()
    expect(runs(spy)).toHaveLength(2)
  })

  it('sends filters to the server and explains an empty filtered result', async () => {
    const spy = apiMock((url) =>
      Response.json({
        data: url.searchParams.get('severity')
          ? report([statusDisc, partsDisc], { items: [], total: 0 })
          : report([statusDisc, partsDisc]),
      }),
    )
    const user = userEvent.setup()
    renderApp('/inventory/reconciliation', { me: me(RECON) })
    await startRun(user)
    await screen.findByRole('table', { name: 'Reconciliation discrepancies' })
    await user.selectOptions(screen.getByLabelText('Severity'), 'MEDIUM')
    expect(await screen.findByText('No discrepancies match these filters')).toBeInTheDocument()
    expect(runs(spy).at(-1)).toMatchObject({ severity: 'MEDIUM', limit: '25' })
    // The whole-run summary stays.
    expect(screen.getByText('2 discrepancies')).toBeInTheDocument()
  })

  it('shows a failed run with a retry, and never partial results', async () => {
    let fail = true
    const spy = apiMock(() =>
      fail ? err(500, 'INTERNAL_ERROR') : Response.json({ data: report([statusDisc]) }),
    )
    const user = userEvent.setup()
    renderApp('/inventory/reconciliation', { me: me(RECON) })
    await startRun(user)
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('The reconciliation did not run')
    expect(alert).toHaveTextContent('Nothing was changed')
    expect(alert).toHaveTextContent('req-9')
    expect(screen.queryByRole('table')).toBeNull()
    fail = false
    await user.click(within(alert).getByRole('button', { name: 'Try again' }))
    expect(await screen.findByRole('table', { name: 'Reconciliation discrepancies' })).toBeVisible()
    expect(runs(spy)).toHaveLength(2)
  })

  it('explains a network failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('Failed to fetch'))
    const user = userEvent.setup()
    renderApp('/inventory/reconciliation', { me: me(RECON) })
    await startRun(user)
    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot reach the server')
  })

  it('handles a refused run and invalid filters', async () => {
    apiMock((url) =>
      url.searchParams.get('locationId')
        ? err(422, 'VALIDATION_FAILED', [
            { field: 'locationId', message: 'is not one of your locations' },
          ])
        : err(403, 'FORBIDDEN'),
    )
    const user = userEvent.setup()
    renderApp('/inventory/reconciliation?locationId=l1', { me: me(RECON) })
    await startRun(user)
    const invalid = (await screen.findByText('is not one of your locations')).closest(
      '[role="alert"]',
    ) as HTMLElement
    await user.click(within(invalid).getByRole('button', { name: 'Clear filters' }))
    expect(await screen.findByText(/do not have permission/i)).toBeInTheDocument()
  })

  it('is denied without the report permission or without any item permission', async () => {
    const spy = apiMock()
    renderApp('/inventory/reconciliation', { me: me(['vehicle_units.view', 'stock.parts.view']) })
    expect(await screen.findByText(/do not have permission/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Run reconciliation' })).toBeNull()
    cleanup()
    renderApp('/inventory/reconciliation', { me: me(['stock.reconciliation.view']) })
    expect(await screen.findByText(/do not have permission/i)).toBeInTheDocument()
    expect(runs(spy)).toEqual([])
  })

  it('only offers the filters and types the user can see', async () => {
    apiMock()
    renderApp('/inventory/reconciliation', {
      me: me(['stock.reconciliation.view', 'stock.parts.view']),
    })
    const type = await screen.findByLabelText('Discrepancy')
    const values = within(type)
      .getAllByRole('option')
      .map((o) => (o as HTMLOptionElement).value)
      .filter(Boolean)
    expect(values.every((v) => v.startsWith('PARTS_'))).toBe(true)
    expect(screen.queryByLabelText('Item type')).toBeNull()
    expect(screen.getByText(/every parts balance/)).toBeInTheDocument()
  })
})
