import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import type { Brand } from '@/features/catalogue/api/contracts'
import { renderApp, testMe } from '@/test/render'
import type { BrandLink, Supplier, SupplierDetail } from './api/suppliers'

// Placeholder data only ("Example Supplier", synthetic GSTIN): nothing real.

const stamps = { version: 2, createdAt: '2026-10-13T09:00:00Z', updatedAt: '2026-10-13T09:00:00Z' }
const supplier: Supplier = {
  id: 's1',
  legalName: 'Example Supplier Private Limited',
  tradeName: null,
  gstin: '27ABCDE1234F1Z5',
  addressLine1: null,
  addressLine2: null,
  city: 'Pune',
  state: 'Maharashtra',
  postalCode: null,
  status: 'PENDING_VERIFICATION',
  verificationReference: null,
  verificationDocumentId: null,
  verifiedBy: null,
  verifiedAt: null,
  notes: null,
  createdBy: 'u-x',
  brandCount: 1,
  ...stamps,
}
const brand = (
  id: string,
  name: string,
  lifecycleStatus: Brand['lifecycleStatus'] = 'PENDING_VERIFICATION',
): Brand => ({
  id,
  name,
  slug: id,
  lifecycleStatus,
  authorisationStatus: 'UNVERIFIED',
  manufacturerLegalName: null,
  authorisationReference: null,
  authorisationEvidenceId: null,
  verifiedBy: null,
  verifiedAt: null,
  operational: false,
  ...stamps,
})
const link: BrandLink = {
  id: 'l1',
  supplierId: 's1',
  supplierName: supplier.legalName,
  supplierStatus: 'PENDING_VERIFICATION',
  brandId: 'b1',
  brandName: 'Brand One',
  status: 'UNVERIFIED',
  authorisationReference: null,
  evidenceDocumentId: null,
  effectiveFrom: null,
  effectiveTo: null,
  verifiedBy: null,
  verifiedAt: null,
  note: null,
  endedReason: null,
  current: false,
  ...stamps,
}
const detail: SupplierDetail = {
  ...supplier,
  contacts: [
    {
      id: 'c1',
      supplierId: 's1',
      name: 'Contact One',
      role: null,
      phone: '9000000000',
      email: null,
      isPrimary: true,
      active: true,
      ...stamps,
    },
  ],
  documents: [
    {
      id: 'd1',
      supplierId: 's1',
      attachmentId: 'a1',
      documentType: 'AUTHORISATION',
      note: null,
      fileName: 'letter.pdf',
      contentType: 'application/pdf',
      addedBy: 'u-x',
      createdAt: '2026-10-13T09:00:00Z',
    },
  ],
  brandLinks: [link],
}
const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const me = (permissions: string[]): Me => ({ ...testMe, permissions })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-5', details } }, { status })

type Handler = (url: URL, init: RequestInit | undefined) => Response | undefined

function api(handler: Handler = () => undefined) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/suppliers') return json(page([supplier]))
    if (p === '/api/v1/suppliers/s1') return json(detail)
    if (p === '/api/v1/brands')
      return json(
        page([
          brand('b1', 'Brand One'),
          brand('b2', 'Brand Two'),
          brand('b3', 'Old Brand', 'ARCHIVED'),
        ]),
      )
    if (p === '/api/v1/brands/b1') return json(brand('b1', 'Brand One'))
    if (p === '/api/v1/brands/b1/activation-readiness')
      return json({ brandId: 'b1', ready: false, checks: [] })
    if (p === '/api/v1/models') return json(page([]))
    if (p === '/api/v1/supplier-brand-links') return json([link])
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}

function bodyOf(init: RequestInit | undefined): unknown {
  return typeof init?.body === 'string' ? JSON.parse(init.body) : undefined
}

const ALL = ['suppliers.view', 'suppliers.manage', 'suppliers.verify', 'suppliers.archive']

afterEach(() => {
  vi.restoreAllMocks()
})

describe('supplier list', () => {
  it('is denied without suppliers.view and never calls the API', async () => {
    const spy = api()
    renderApp('/procurement/suppliers', { me: me([]) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/suppliers'))).toBe(false)
  })

  it('keeps the heading and breadcrumbs when a detail link is denied without suppliers.view', async () => {
    const spy = api()
    renderApp('/procurement/suppliers/s1', { me: me([]) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: 'Supplier' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Suppliers' })).toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/suppliers'))).toBe(false)
  })

  it('lists suppliers with status, says a record is not authorisation, and sends filters', async () => {
    const user = userEvent.setup()
    const spy = api()
    renderApp('/procurement/suppliers', { me: me(['suppliers.view']) })
    const table = await screen.findByRole('table', { name: 'Suppliers' })
    expect(within(table).getByText('Pending verification')).toBeInTheDocument()
    expect(within(table).getByText('27ABCDE1234F1Z5')).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent(/not proof of dealership authorisation/)
    expect(screen.queryByRole('button', { name: 'Add supplier' })).not.toBeInTheDocument()
    await user.selectOptions(screen.getByLabelText('Status'), 'VERIFIED')
    await waitFor(() => {
      expect(
        spy.mock.calls.some(
          ([u]) => new URL(u as string, 'http://x').searchParams.get('status') === 'VERIFIED',
        ),
      ).toBe(true)
    })
  })

  it('validates a new supplier, shows a server duplicate on the field and opens it after the server confirms', async () => {
    const user = userEvent.setup()
    let n = 0
    let sent: unknown
    api((u, init) => {
      if (u.pathname === '/api/v1/suppliers' && init?.method === 'POST') {
        n++
        sent = bodyOf(init)
        return n === 1
          ? err(409, 'DUPLICATE', [{ field: 'legalName', message: 'is already used' }])
          : Response.json({ data: supplier }, { status: 201 })
      }
      return undefined
    })
    renderApp('/procurement/suppliers', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Add supplier' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add supplier' })
    await user.type(within(dialog).getByLabelText(/Legal name/), 'Example Supplier Private Limited')
    await user.type(within(dialog).getByLabelText(/GSTIN/), 'NOT-A-GSTIN')
    await user.click(within(dialog).getByRole('button', { name: 'Add supplier' }))
    expect(await within(dialog).findByText(/Enter a 15-character GSTIN/)).toBeInTheDocument()
    await user.clear(within(dialog).getByLabelText(/GSTIN/))
    await user.type(within(dialog).getByLabelText(/GSTIN/), '27abcde1234f1z5')
    await user.click(within(dialog).getByRole('button', { name: 'Add supplier' }))
    expect(await within(dialog).findByText('Already used — choose another')).toBeInTheDocument()
    expect(sent).toMatchObject({
      legalName: 'Example Supplier Private Limited',
      gstin: '27ABCDE1234F1Z5',
      city: null,
    })
    await user.click(within(dialog).getByRole('button', { name: 'Add supplier' }))
    expect(
      await screen.findByRole('heading', { level: 1, name: supplier.legalName }),
    ).toBeInTheDocument()
  })
})

describe('supplier detail', () => {
  it('shows contacts, documents and brand relationships with permission-aware actions', async () => {
    api()
    renderApp('/procurement/suppliers/s1', { me: me(['suppliers.view']) })
    expect(
      await screen.findByRole('heading', { level: 1, name: supplier.legalName }),
    ).toBeInTheDocument()
    expect(screen.getAllByText('Contact One').length).toBeGreaterThan(0)
    expect(screen.getAllByText('letter.pdf').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Unverified').length).toBeGreaterThan(0)
    expect(screen.getByText(/Not verified\./)).toBeInTheDocument()
    for (const name of [
      'Edit',
      'Archive',
      'Verify',
      'Add contact',
      'Attach document',
      'Add brand',
      'End',
    ]) {
      expect(screen.queryByRole('button', { name })).not.toBeInTheDocument()
    }
    // Opening documents needs attachments.view as well.
    expect(screen.queryByRole('button', { name: /Download/ })).not.toBeInTheDocument()
    expect(
      screen.getByText('Opening documents needs permission to view documents.'),
    ).toBeInTheDocument()
  })

  it('offers only brands without an open relationship and never archived ones', async () => {
    const user = userEvent.setup()
    api()
    renderApp('/procurement/suppliers/s1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Add brand' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add brand relationship' })
    const select = within(dialog).getByLabelText(/Brand/)
    await waitFor(() => {
      expect(
        within(select)
          .getAllByRole('option')
          .map((o) => o.textContent),
      ).toEqual(['Choose a brand', 'Brand Two'])
    })
  })

  it('verifies a relationship with the supplier’s own document and shows the server’s refusal', async () => {
    const user = userEvent.setup()
    let sent: unknown
    api((u, init) => {
      if (u.pathname === '/api/v1/supplier-brand-links/l1/verify') {
        sent = bodyOf(init)
        return err(409, 'INVALID_STATE')
      }
      return undefined
    })
    renderApp('/procurement/suppliers/s1', { me: me(ALL) })
    const table = await screen.findByRole('table', { name: 'Brand relationships' })
    await user.click(within(table).getByRole('button', { name: 'Verify' }))
    const dialog = await screen.findByRole('dialog', { name: 'Verify relationship with Brand One' })
    expect(within(dialog).getByText(/Verify the supplier itself first/)).toBeInTheDocument()
    await user.selectOptions(within(dialog).getByLabelText(/Authorisation document/), 'a1')
    await user.type(within(dialog).getByLabelText(/Authorisation reference/), 'DLR-1')
    await user.type(within(dialog).getByLabelText(/Valid from/), '2026-10-01')
    await user.click(within(dialog).getByRole('button', { name: 'Verify relationship' }))
    expect(await within(dialog).findByText('Refused.')).toBeInTheDocument()
    expect(sent).toEqual({
      version: 2,
      authorisationReference: 'DLR-1',
      evidenceDocumentId: 'a1',
      effectiveFrom: '2026-10-01',
      effectiveTo: null,
    })
  })

  it('requires a phone or email for a contact', async () => {
    const user = userEvent.setup()
    api()
    renderApp('/procurement/suppliers/s1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Add contact' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add contact' })
    await user.type(within(dialog).getByLabelText(/Name/), 'Contact Two')
    await user.click(within(dialog).getByRole('button', { name: 'Add contact' }))
    expect(await within(dialog).findByText('Give a phone number or an email')).toBeInTheDocument()
  })

  it('reports an unknown supplier as not found', async () => {
    api()
    renderApp('/procurement/suppliers/nope', { me: me(['suppliers.view']) })
    expect(await screen.findByText('This supplier does not exist')).toBeInTheDocument()
  })
})

describe('brand page supplier relationships', () => {
  it('lists real relationships for supplier viewers', async () => {
    api()
    renderApp('/admin/catalogue/brands/b1', { me: me(['catalogue.view', 'suppliers.view']) })
    const list = await screen.findByRole('list', { name: 'Supplier relationships' })
    expect(within(list).getByRole('link', { name: supplier.legalName })).toBeInTheDocument()
    expect(within(list).getByText('Unverified')).toBeInTheDocument()
  })
})
