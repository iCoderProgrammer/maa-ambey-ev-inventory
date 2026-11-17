/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import type { CommitResult } from './api/commit'
import type { ImportApproval, ImportBatch, Specification } from './api/openingStock'

// G-07 review → approval → commit against a mocked server. Placeholder values
// only; the "server" here decides every answer, exactly as the real one does.

const t = '2026-11-16T09:00:00Z'
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: true, locations: [] },
})
const ALL = ['stock.import.preview', 'stock.import.approve', 'stock.import.commit']

const spec = (approvedOpeningStatuses: string[] = ['AVAILABLE']): Specification => ({
  templateVersion: 'OS-IMPORT-1',
  format: 'CSV',
  firstLine: '#template:OS-IMPORT-1',
  maxBytes: 5 << 20,
  maxRows: 5000,
  previewTtlHours: 168,
  columns: [],
  stockStatuses: ['AVAILABLE'],
  approvedOpeningStatuses,
  businessDate: '2026-11-16',
  timeZone: 'Asia/Kolkata',
  notes: [],
})

const approval: ImportApproval = {
  id: 'ap1',
  approvedById: 'u-approver',
  approvedByName: 'Approver Test',
  approvedAt: t,
  physicalCountReference: 'PC-TEST-1',
  physicalCountDate: '2026-11-15',
  mappingsReviewed: true,
  duplicatesReviewed: true,
  receivedDatesApproved: true,
  note: null,
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
  uploadedByName: 'Uploader Test',
  uploadedAt: t,
  businessDate: '2026-11-16',
  expiresAt: '2026-11-23T09:00:00Z',
  expired: false,
  summary: {
    totalRows: 2,
    validRows: 2,
    warningRows: 0,
    errorRows: 0,
    blockedRows: 0,
    canProceedRows: 2,
    duplicateRows: 0,
    unresolvedRows: 0,
    skippedBlankRows: 0,
  },
  fileIssues: [],
  commitAvailable: false,
  approval: null,
  commit: null,
  ...over,
})

const result = (over: Partial<CommitResult> = {}): CommitResult => ({
  commitId: 'c1',
  reference: 'OSC-000001',
  batchId: 'b1',
  batchReference: 'OSI-000001',
  batchStatus: 'COMMITTED',
  unitsCreated: 2,
  committedById: 'u-committer',
  committedByName: 'Committer Test',
  committedAt: t,
  approvalId: 'ap1',
  fileSha256: 'a'.repeat(64),
  templateVersion: 'OS-IMPORT-1',
  sourceType: 'OPENING_STOCK',
  movementType: 'OPENING_BALANCE',
  units: [
    {
      rowNumber: 3,
      vehicleUnitId: 'v1',
      unitNumber: 'MAE-VEH-000101',
      status: 'AVAILABLE',
      movementId: 'm1',
    },
    {
      rowNumber: 4,
      vehicleUnitId: 'v2',
      unitNumber: 'MAE-VEH-000102',
      status: 'AVAILABLE',
      movementId: 'm2',
    },
  ],
  replayed: false,
  ...over,
})

/** 'lost': the server commits, but the answer never arrives. */
type Answer = Response | 'network' | 'lost'
const err = (status: number, code: string, message = 'Refused.', details?: unknown) =>
  Response.json({ error: { code, message, requestId: 'req-1', details } }, { status })
const data = (d: unknown, status = 200) => Response.json({ data: d }, { status })

interface Server {
  batch: ImportBatch
  commit: CommitResult | null
  spec: Specification
  approvalAnswers: Answer[]
  commitAnswers: Answer[]
  calls: { method: string; path: string; headers: Record<string, string>; body: unknown }[]
}

function serve(over: Partial<Server> = {}): Server {
  const s: Server = {
    batch: batch(),
    commit: null,
    spec: spec(),
    approvalAnswers: [],
    commitAnswers: [],
    calls: [],
    ...over,
  }
  vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const p = url.pathname
    const method = init?.method ?? 'GET'
    const headers = (init?.headers ?? {}) as Record<string, string>
    s.calls.push({
      method,
      path: p,
      headers,
      body: typeof init?.body === 'string' ? JSON.parse(init.body) : undefined,
    })
    const answer = (a: Answer | undefined, fallback: () => Response) => {
      if (a === 'lost') s.commit = result()
      return a === 'network' || a === 'lost'
        ? Promise.reject(new TypeError('Failed to fetch'))
        : Promise.resolve(a ?? fallback())
    }
    if (p === '/api/v1/opening-stock-imports/specification') return Promise.resolve(data(s.spec))
    if (p === '/api/v1/opening-stock-imports/b1') return Promise.resolve(data(s.batch))
    if (p === '/api/v1/opening-stock-imports/b1/rows')
      return Promise.resolve(data({ items: [], page: 1, limit: 200, total: 0 }))
    if (p === '/api/v1/opening-stock-imports/b1/approval' && method === 'POST')
      return answer(s.approvalAnswers.shift(), () => data(approval, 201))
    if (p === '/api/v1/opening-stock-imports/b1/commit' && method === 'POST')
      return answer(s.commitAnswers.shift(), () => {
        s.commit = result()
        return data(result(), 201)
      })
    if (p === '/api/v1/opening-stock-imports/b1/commit')
      return Promise.resolve(s.commit ? data(s.commit) : err(404, 'NOT_FOUND', 'Not committed.'))
    if (p.endsWith('/health'))
      return Promise.resolve(data({ status: 'ok', service: 'api', version: 'v', time: 'x' }))
    if (p.endsWith('/ready'))
      return Promise.resolve(
        data({ status: 'ready', checks: { database: 'up', migrations: 'up' } }),
      )
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
  return s
}
const posts = (s: Server, suffix: string) =>
  s.calls.filter((c) => c.method === 'POST' && c.path.endsWith(suffix))
const gets = (s: Server, p: string) => s.calls.filter((c) => c.method === 'GET' && c.path === p)

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

const open = (perms = ALL) => renderApp('/inventory/opening-stock/b1/commit', { me: me(perms) })

async function fillApproval(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/physical count reference/i), 'PC-REAL-7')
  await user.type(screen.getByLabelText(/physical count date/i), '2026-11-15')
  await user.click(screen.getByRole('checkbox', { name: /reviewed the brand, model/i }))
  await user.click(screen.getByRole('checkbox', { name: /reviewed the duplicate/i }))
  await user.click(screen.getByRole('radio', { name: /^no/i }))
}

describe('opening stock commit API contract (G-07)', () => {
  it('matches the server field for field (docs/openapi)', () => {
    const doc = JSON.parse(
      readFileSync(
        path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json'),
        'utf8',
      ),
    ) as {
      definitions: Record<string, { properties: Record<string, unknown> }>
      paths: Record<string, Record<string, { parameters?: { name: string; in: string }[] }>>
    }
    const keys = (name: string) => Object.keys(doc.definitions[name]?.properties ?? {}).sort()
    expect(Object.keys(approval).sort()).toEqual(keys('OpeningStockImportApproval'))
    expect(Object.keys(result()).sort()).toEqual(keys('OpeningStockImportCommit'))
    expect(Object.keys(result().units[0] ?? {}).sort()).toEqual(keys('OpeningStockCommittedUnit'))
    expect(
      [
        'physicalCountReference',
        'physicalCountDate',
        'mappingsReviewed',
        'duplicatesReviewed',
        'receivedDatesApproved',
        'note',
      ].sort(),
    ).toEqual(keys('OpeningStockImportApprovalRequest'))
    const commitPost = doc.paths['/opening-stock-imports/{id}/commit']?.post?.parameters ?? []
    expect(commitPost.some((p) => p.name === 'Idempotency-Key' && p.in === 'header')).toBe(true)
    expect(doc.paths['/opening-stock-imports/{id}/commit']?.get).toBeDefined()
    expect(doc.paths['/opening-stock-imports/{id}/approval']?.post).toBeDefined()
  })
})

describe('approval (G-07)', () => {
  it('starts with nothing selected, validates, confirms and records exactly what was entered', async () => {
    const s = serve()
    const user = userEvent.setup()
    open()
    expect(await screen.findByRole('heading', { name: /2\. approval/i })).toBeInTheDocument()
    expect(screen.getByText('b1')).toBeInTheDocument()
    for (const name of [/reviewed the brand, model/i, /reviewed the duplicate/i])
      expect(screen.getByRole('checkbox', { name })).not.toBeChecked()
    for (const name of [/^yes/i, /^no/i])
      expect(screen.getByRole('radio', { name })).not.toBeChecked()
    expect(posts(s, '/approval')).toHaveLength(0) // never sent on opening

    await user.click(screen.getByRole('button', { name: /review the approval/i }))
    expect(
      await screen.findByText(/enter the reference of the signed physical count/i),
    ).toBeInTheDocument()
    expect(screen.getByText(/confirm the mappings were reviewed/i)).toBeInTheDocument()
    expect(screen.getByText(/choose whether the received dates/i)).toBeInTheDocument()
    expect(posts(s, '/approval')).toHaveLength(0)

    await fillApproval(user)
    await user.click(screen.getByRole('button', { name: /review the approval/i }))
    const dialog = await screen.findByRole('dialog', { name: /record this approval/i })
    expect(within(dialog).getByText('PC-REAL-7')).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: /cancel/i }))
    expect(posts(s, '/approval')).toHaveLength(0)

    await user.click(screen.getByRole('button', { name: /review the approval/i }))
    const before = gets(s, '/api/v1/opening-stock-imports/b1').length
    s.batch = batch({
      approval: { ...approval, physicalCountReference: 'PC-REAL-7', receivedDatesApproved: false },
    })
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: /record approval/i }),
    )
    await waitFor(() => {
      expect(posts(s, '/approval')).toHaveLength(1)
    })
    expect(posts(s, '/approval')[0]?.body).toEqual({
      physicalCountReference: 'PC-REAL-7',
      physicalCountDate: '2026-11-15',
      mappingsReviewed: true,
      duplicatesReviewed: true,
      receivedDatesApproved: false,
      note: null,
    })
    // The batch is re-read; the server's approval is shown.
    expect(await screen.findByText(/Approver Test/)).toBeInTheDocument()
    await waitFor(() => {
      expect(gets(s, '/api/v1/opening-stock-imports/b1').length).toBeGreaterThan(before)
    })
    expect(
      await screen.findByText(/no — stock age will show as not available/i),
    ).toBeInTheDocument()
  })

  it('shows the server’s field errors and refuses without permission', async () => {
    serve({
      approvalAnswers: [
        err(422, 'VALIDATION_FAILED', 'Check the highlighted fields.', [
          { field: 'physicalCountDate', message: 'cannot be after today' },
        ]),
        err(403, 'FORBIDDEN', 'You do not have permission.'),
      ],
    })
    const user = userEvent.setup()
    open()
    await screen.findByRole('heading', { name: /2\. approval/i })
    await fillApproval(user)
    await user.click(screen.getByRole('button', { name: /review the approval/i }))
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: /record approval/i }),
    )
    expect(await screen.findByText('cannot be after today')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /review the approval/i }))
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', { name: /record approval/i }),
    )
    expect((await screen.findAllByText(/permission/i)).length).toBeGreaterThan(0)
  })

  it('offers no approval form without stock.import.approve', async () => {
    serve()
    open(['stock.import.preview', 'stock.import.commit'])
    expect(await screen.findByText(/you do not have it/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /review the approval/i })).not.toBeInTheDocument()
  })
})

describe('commit (G-07)', () => {
  const approved = () => batch({ approval })

  it.each([
    [
      'fatal errors',
      batch({
        approval,
        summary: { ...batch().summary, validRows: 1, errorRows: 1, canProceedRows: 1 },
      }),
      spec(),
      /1 line\(s\) have errors/i,
    ],
    ['no approval', batch(), spec(), /must be approved before/i],
    ['no approved opening status', approved(), spec([]), /no stock status is approved/i],
    ['expired preview', batch({ approval, expired: true }), spec(), /preview has expired/i],
    [
      'rejected file',
      batch({
        approval,
        status: 'REJECTED',
        summary: { ...batch().summary, totalRows: 0, validRows: 0, canProceedRows: 0 },
      }),
      spec(),
      /could not be read as the template/i,
    ],
  ])('is not offered with %s', async (_name, b, sp, reason) => {
    const s = serve({ batch: b, spec: sp })
    open()
    expect((await screen.findAllByText(reason)).length).toBeGreaterThan(0)
    expect(screen.getByRole('button', { name: /commit this batch/i })).toBeDisabled()
    expect(posts(s, '/commit')).toHaveLength(0)
  })

  it('confirms with the server’s data, sends once with a key and shows the server’s result', async () => {
    const s = serve({ batch: approved() })
    const user = userEvent.setup()
    open()
    await user.click(await screen.findByRole('button', { name: /commit this batch/i }))
    const dialog = await screen.findByRole('dialog', { name: /commit this batch into stock/i })
    expect(within(dialog).getByText('OSI-000001 (b1)')).toBeInTheDocument()
    expect(within(dialog).getByText('stock.csv')).toBeInTheDocument()
    expect(within(dialog).getByText(/PC-TEST-1/)).toBeInTheDocument()
    expect(
      within(dialog).getByText(/creates inventory records and stock movements/i),
    ).toBeInTheDocument()
    const go = within(dialog).getByRole('button', { name: /commit 2 line/i })
    expect(go).toBeDisabled() // explicit confirmation first
    await user.click(within(dialog).getByRole('checkbox'))
    // Two clicks in the same frame, before React can disable the button.
    fireEvent.click(go)
    fireEvent.click(go)
    expect(
      await screen.findByRole('heading', { name: /committed · OSC-000001/i }),
    ).toBeInTheDocument()
    expect(posts(s, '/commit')).toHaveLength(1)
    expect(posts(s, '/commit')[0]?.headers['Idempotency-Key']).toMatch(/^osc-[0-9a-f-]{36}$/)
    expect(screen.getByRole('link', { name: 'MAE-VEH-000101' })).toHaveAttribute(
      'href',
      '/inventory/vehicles/v1',
    )
    expect(screen.getByText(/2 vehicle unit\(s\) were created/i)).toBeInTheDocument()
    expect(screen.queryByText(/repeats the earlier/i)).not.toBeInTheDocument()
  })

  it('shows a replayed answer as the earlier commit', async () => {
    serve({ batch: approved(), commitAnswers: [data(result({ replayed: true }), 200)] })
    const user = userEvent.setup()
    open()
    await user.click(await screen.findByRole('button', { name: /commit this batch/i }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('checkbox'))
    await user.click(within(dialog).getByRole('button', { name: /commit 2 line/i }))
    expect(await screen.findByText(/repeats the earlier successful commit/i)).toBeInTheDocument()
  })

  it('after a lost response asks the server instead of committing again', async () => {
    const s = serve({ batch: approved(), commitAnswers: ['lost'] })
    const user = userEvent.setup()
    open()
    await user.click(await screen.findByRole('button', { name: /commit this batch/i }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('checkbox'))
    const statusChecks = gets(s, '/api/v1/opening-stock-imports/b1/commit').length
    await user.click(within(dialog).getByRole('button', { name: /commit 2 line/i }))
    expect(
      await screen.findByRole('heading', { name: /committed · OSC-000001/i }),
    ).toBeInTheDocument()
    expect(posts(s, '/commit')).toHaveLength(1)
    expect(gets(s, '/api/v1/opening-stock-imports/b1/commit').length).toBeGreaterThan(statusChecks)
  })

  it('when nothing was committed, retries with the same key', async () => {
    const s = serve({
      batch: approved(),
      commitAnswers: ['network', err(502, 'BAD_GATEWAY', 'Bad gateway.')],
    })
    const user = userEvent.setup()
    open()
    await user.click(await screen.findByRole('button', { name: /commit this batch/i }))
    let dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('checkbox'))
    await user.click(within(dialog).getByRole('button', { name: /commit 2 line/i }))
    expect(
      await screen.findByText(/server has no commit for this batch: nothing was created/i),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /try the commit again/i }))
    expect(await screen.findByText(/server has no commit for this batch/i)).toBeInTheDocument()
    // A fresh attempt through the dialog keeps the key as well.
    await user.click(screen.getByRole('button', { name: /commit this batch/i }))
    dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('checkbox'))
    await user.click(within(dialog).getByRole('button', { name: /commit 2 line/i }))
    await waitFor(() => {
      expect(posts(s, '/commit')).toHaveLength(3)
    })
    const keys = new Set(posts(s, '/commit').map((c) => c.headers['Idempotency-Key']))
    expect(keys.size).toBe(1)
  })

  it('sends nothing when the status check before the commit fails', async () => {
    const s = serve({ batch: approved() })
    const user = userEvent.setup()
    open()
    await user.click(await screen.findByRole('button', { name: /commit this batch/i }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('checkbox'))
    // The session ends before the click: GET …/commit answers 401.
    vi.mocked(globalThis.fetch).mockImplementation(() =>
      Promise.resolve(err(401, 'UNAUTHENTICATED', 'Sign in.')),
    )
    await user.click(within(dialog).getByRole('button', { name: /commit 2 line/i }))
    expect(await screen.findByText(/your session has ended/i)).toBeInTheDocument()
    const sent = vi
      .mocked(globalThis.fetch)
      .mock.calls.filter(
        ([u, init]) => (u as string).endsWith('/b1/commit') && init?.method === 'POST',
      )
    expect(sent).toHaveLength(0)
    expect(posts(s, '/commit')).toHaveLength(0)
  })

  it('shows an existing commit after a reload and offers nothing else', async () => {
    const s = serve({
      batch: batch({
        approval,
        commit: {
          id: 'c1',
          reference: 'OSC-000001',
          committedById: 'u',
          committedByName: 'C',
          committedAt: t,
          unitsCreated: 2,
        },
      }),
      commit: result(),
    })
    open()
    expect(
      await screen.findByRole('heading', { name: /committed · OSC-000001/i }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /commit this batch/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /review the approval/i })).not.toBeInTheDocument()
    expect(posts(s, '/commit')).toHaveLength(0)
  })

  it('on 409 already committed shows that commit', async () => {
    const s = serve({
      batch: approved(),
      commitAnswers: [err(409, 'IMPORT_ALREADY_COMMITTED', 'Already committed as OSC-000001.')],
    })
    const user = userEvent.setup()
    open()
    await user.click(await screen.findByRole('button', { name: /commit this batch/i }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('checkbox'))
    s.commit = result()
    await user.click(within(dialog).getByRole('button', { name: /commit 2 line/i }))
    expect(
      await screen.findByRole('heading', { name: /committed · OSC-000001/i }),
    ).toBeInTheDocument()
  })

  it.each([
    [
      err(409, 'OPENING_STATUSES_NOT_APPROVED', 'No stock status is approved.'),
      /has not approved any opening-stock status/i,
    ],
    [
      err(409, 'IMPORT_PREVIEW_EXPIRED', 'This preview has expired.'),
      /upload the file again; the new preview needs its own approval/i,
    ],
    [
      err(409, 'IMPORT_APPROVAL_REQUIRED', 'No approval.'),
      /record the physical-count approval first/i,
    ],
    [
      err(409, 'INVALID_STATE', 'Not committable.'),
      /cannot be committed\. upload a corrected file/i,
    ],
    [
      err(422, 'IMPORT_ROWS_BLOCKED', '1 line(s) cannot be committed.', [
        {
          field: 'rows[4].chassis_number',
          message: 'DUPLICATE_CHASSIS_EXISTING (ERROR): Chassis CH-2 is already registered.',
        },
      ]),
      /chassis CH-2 is already registered/i,
    ],
    [err(403, 'FORBIDDEN', 'Forbidden.'), /permission/i],
  ])('explains a refusal (%#) and creates nothing', async (answer, text) => {
    const s = serve({ batch: approved(), commitAnswers: [answer] })
    const user = userEvent.setup()
    open()
    await user.click(await screen.findByRole('button', { name: /commit this batch/i }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('checkbox'))
    await user.click(within(dialog).getByRole('button', { name: /commit 2 line/i }))
    expect((await screen.findAllByText(text)).length).toBeGreaterThan(0)
    expect(screen.queryByRole('heading', { name: /committed ·/i })).not.toBeInTheDocument()
    expect(posts(s, '/commit')).toHaveLength(1)
  })

  it('never shows a browser-made count as the result', async () => {
    // The server says 2 units were created although the batch had 3 lines: the server's number is shown.
    serve({
      batch: batch({
        approval,
        summary: { ...batch().summary, totalRows: 3, validRows: 3, canProceedRows: 3 },
      }),
      commit: result(),
    })
    open()
    expect(await screen.findByText(/2 vehicle unit\(s\) were created/i)).toBeInTheDocument()
  })

  it('refuses the page without any import permission', async () => {
    const s = serve()
    open(['vehicle_units.view'])
    expect(await screen.findByText(/do not have permission/i)).toBeInTheDocument()
    expect(gets(s, '/api/v1/opening-stock-imports/b1')).toHaveLength(0)
  })
})

describe('G-06 preview stays preview-only', () => {
  it('keeps its statement, has no commit button and links to the separate review page', async () => {
    serve()
    renderApp('/inventory/opening-stock/b1', { me: me(ALL) })
    expect(
      await screen.findByText(
        'Preview completed. No inventory records have been created or modified.',
      ),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /commit/i })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: /review, approve and commit/i })).toHaveAttribute(
      'href',
      '/inventory/opening-stock/b1/commit',
    )
  })

  it('shows no review link to a preview-only user', async () => {
    serve()
    renderApp('/inventory/opening-stock/b1', { me: me(['stock.import.preview']) })
    await screen.findByText(
      'Preview completed. No inventory records have been created or modified.',
    )
    expect(
      screen.queryByRole('link', { name: /review, approve and commit/i }),
    ).not.toBeInTheDocument()
  })
})
