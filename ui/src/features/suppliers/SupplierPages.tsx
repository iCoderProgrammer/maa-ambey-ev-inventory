import { Download, Pencil, Plus, ShieldCheck, Truck, UserMinus } from 'lucide-react'
import { useCallback, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Pagination } from '@/components/ui/Pagination'
import { AccessDeniedState, EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { fetchAttachment } from '@/features/attachments/api/attachments'
import { ArchivedBanner, DefinitionList } from '@/features/catalogue/components/DetailParts'
import { FilterSelect, SearchBox, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { useBrands } from '@/features/catalogue/hooks/useCatalogue'
import { useListParams } from '@/features/catalogue/lib/listParams'
import {
  describeError,
  formatDate,
  formatDateTime,
  personLabel,
} from '@/features/catalogue/lib/presentation'
import { ApiError } from '@/lib/api/client'
import {
  removeContact,
  type BrandLink,
  type Contact,
  type Supplier,
  type SupplierDetail,
  type SupplierDocument,
  type SupplierStatus,
} from './api/suppliers'
import {
  AddBrandLinkDialog,
  ArchiveSupplierDialog,
  ContactFormDialog,
  DocumentDialog,
  EndBrandLinkDialog,
  SupplierFormDialog,
  VerifyBrandLinkDialog,
  VerifySupplierDialog,
} from './components/SupplierDialogs'
import { LinkStatusBadge, SupplierStatusBadge } from './components/SupplierBadges'
import { useSupplier, useSupplierList, useSupplierMutation } from './hooks/useSuppliers'
import { documentTypeLabels, supplierStatusLabels } from './lib/labels'

const crumbs = [
  { label: 'Home', to: '/' },
  { label: 'Suppliers', to: '/procurement/suppliers' },
]
const linkClass = 'font-medium text-primary hover:underline'

function usePerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return {
    me,
    meId: me.data?.user.id,
    view: p.includes(Permission.SupplierView),
    manage: p.includes(Permission.SupplierManage),
    verify: p.includes(Permission.SupplierVerify),
    archive: p.includes(Permission.SupplierArchive),
    upload: p.includes(Permission.AttachmentsUpload),
    openFiles: p.includes(Permission.AttachmentsView),
  }
}

/** Gate for both pages: suppliers.view, checked again by the API.
 * `header` keeps a detail page's heading and breadcrumbs while checking or denied. */
function Gate({ children, header }: { children: ReactNode; header?: ReactNode }) {
  const { me, view } = usePerms()
  if (me.isPending)
    return (
      <>
        {header}
        <LoadingState label="Checking your permissions…" />
      </>
    )
  if (me.isError && !me.data) {
    return (
      <>
        {header}
        <ErrorState
          error={me.error}
          title="Could not check your permissions"
          onRetry={() => void me.refetch()}
        />
      </>
    )
  }
  if (!view)
    return (
      <>
        {header}
        <AccessDeniedState what="suppliers" />
      </>
    )
  return <>{children}</>
}

function NotProofNotice() {
  return (
    <p
      role="note"
      className="mb-4 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm text-ink"
    >
      A supplier record is <strong>not proof of dealership authorisation</strong>. Each brand
      relationship is verified separately with the authorisation document; only verified, current
      relationships count.
    </p>
  )
}

// ── List ────────────────────────────────────────────────────────────────────

export function SuppliersPage() {
  const { manage } = usePerms()
  const [adding, setAdding] = useState(false)
  const navigate = useNavigate()
  return (
    <>
      <SuppliersHeader
        action={
          manage ? (
            <Button
              onClick={() => {
                setAdding(true)
              }}
            >
              <Plus className="size-4" aria-hidden="true" />
              Add supplier
            </Button>
          ) : undefined
        }
      />
      <Gate>
        <SuppliersList
          canAdd={manage}
          onAdd={() => {
            setAdding(true)
          }}
        />
      </Gate>
      <SupplierFormDialog
        open={adding}
        onClose={() => {
          setAdding(false)
        }}
        onSaved={(s) => void navigate(`/procurement/suppliers/${s.id}`)}
      />
    </>
  )
}

function SuppliersHeader({ action }: { action?: ReactNode }) {
  return (
    <PageHeader
      title="Suppliers"
      breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Suppliers' }]}
      description="Supplier master: legal identity, contacts, private documents and verified relationships with brands."
      actions={action}
    />
  )
}

function SuppliersList({ canAdd, onAdd }: { canAdd: boolean; onAdd: () => void }) {
  const { values, page, set } = useListParams(['q', 'status', 'brandId'] as const)
  const brands = useBrands({ limit: 100 })
  const query = useSupplierList({
    q: values.q || undefined,
    status: (values.status || undefined) as SupplierStatus | undefined,
    brandId: values.brandId || undefined,
    page,
    limit: 25,
  })
  const onSearch = useCallback((q: string) => {
    set({ q })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [])
  const filtered = Boolean(values.q || values.status || values.brandId)
  const cols: Column<Supplier>[] = [
    {
      key: 'name',
      header: 'Supplier',
      render: (s) => (
        <div className="flex flex-col">
          <Link to={`/procurement/suppliers/${s.id}`} className={linkClass}>
            {s.legalName}
          </Link>
          {s.tradeName && <span className="text-xs text-muted">{s.tradeName}</span>}
        </div>
      ),
    },
    { key: 'status', header: 'Status', render: (s) => <SupplierStatusBadge status={s.status} /> },
    {
      key: 'gstin',
      header: 'GSTIN',
      render: (s) => <span className="font-mono text-xs">{s.gstin ?? '—'}</span>,
    },
    {
      key: 'place',
      header: 'Location',
      render: (s) => [s.city, s.state].filter(Boolean).join(', ') || '—',
    },
    { key: 'brands', header: 'Brands', render: (s) => String(s.brandCount) },
  ]
  const add = canAdd ? (
    <Button onClick={onAdd}>
      <Plus className="size-4" aria-hidden="true" />
      Add supplier
    </Button>
  ) : null

  let body: ReactNode
  if (query.isPending) body = <LoadingState label="Loading suppliers…" />
  else if (query.isError && !query.data) {
    body = (
      <ErrorState
        error={query.error}
        title="Could not load suppliers"
        onRetry={() => void query.refetch()}
      />
    )
  } else if (query.data.total === 0) {
    body = filtered ? (
      <EmptyState title="No suppliers match these filters">
        Change the search or clear the filters.
      </EmptyState>
    ) : (
      <EmptyState
        title="No suppliers yet"
        icon={<Truck className="size-6 text-muted/80" />}
        action={add}
      >
        Add suppliers from their registration documents. Nothing is filled in for you.
      </EmptyState>
    )
  } else {
    body = (
      <div className={query.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="Suppliers"
          columns={cols}
          rows={query.data.items}
          rowKey={(s) => s.id}
        />
        <Pagination
          page={query.data.page}
          limit={query.data.limit}
          total={query.data.total}
          noun={['supplier', 'suppliers']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  }

  return (
    <>
      <NotProofNotice />
      <Toolbar
        active={filtered}
        onClear={() => {
          set({ q: '', status: '', brandId: '' })
        }}
      >
        <SearchBox label="Search name or GSTIN" value={values.q} onSearch={onSearch} />
        <FilterSelect
          label="Status"
          value={values.status}
          options={Object.entries(supplierStatusLabels).map(([value, label]) => ({ value, label }))}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
        <FilterSelect
          label="Brand"
          value={values.brandId}
          options={(brands.data?.items ?? []).map((b) => ({ value: b.id, label: b.name }))}
          allLabel="Any brand"
          onChange={(v) => {
            set({ brandId: v })
          }}
        />
      </Toolbar>
      {body}
    </>
  )
}

// ── Detail ──────────────────────────────────────────────────────────────────

export function SupplierDetailPage() {
  const { supplierId = '' } = useParams()
  return (
    <Gate header={<PageHeader title="Supplier" breadcrumbs={[...crumbs, { label: 'Supplier' }]} />}>
      <SupplierDetailBody id={supplierId} />
    </Gate>
  )
}

type Open =
  | { kind: 'edit' | 'verify' | 'archive' | 'contact' | 'document' | 'link' }
  | { kind: 'editContact'; contact: Contact }
  | { kind: 'verifyLink' | 'endLink'; link: BrandLink }
  | null

function SupplierDetailBody({ id }: { id: string }) {
  const q = useSupplier(id)
  const perms = usePerms()
  const [open, setOpen] = useState<Open>(null)
  const close = () => {
    setOpen(null)
  }

  if (!q.data) {
    const notFound = q.error instanceof ApiError && q.error.status === 404
    return (
      <>
        <PageHeader title="Supplier" breadcrumbs={[...crumbs, { label: 'Supplier' }]} />
        {q.isPending ? (
          <LoadingState label="Loading supplier…" />
        ) : notFound ? (
          <EmptyState title="This supplier does not exist">
            <Link to="/procurement/suppliers" className={linkClass}>
              Back to suppliers
            </Link>
          </EmptyState>
        ) : (
          <ErrorState
            error={q.error}
            title="Could not load the supplier"
            onRetry={() => void q.refetch()}
          />
        )}
      </>
    )
  }
  const s = q.data
  const archived = s.status === 'ARCHIVED'
  const editable = !archived && perms.manage

  return (
    <>
      <PageHeader
        title={s.legalName}
        breadcrumbs={[...crumbs, { label: s.legalName }]}
        status={<SupplierStatusBadge status={s.status} />}
        actions={
          !archived && (
            <>
              {perms.manage && (
                <Button
                  variant="secondary"
                  onClick={() => {
                    setOpen({ kind: 'edit' })
                  }}
                >
                  <Pencil className="size-4" aria-hidden="true" />
                  Edit
                </Button>
              )}
              {perms.archive && (
                <Button
                  variant="ghost"
                  onClick={() => {
                    setOpen({ kind: 'archive' })
                  }}
                >
                  Archive
                </Button>
              )}
            </>
          )
        }
      />
      {archived && <ArchivedBanner what="supplier" />}
      <NotProofNotice />

      <div className="grid items-start gap-6 lg:grid-cols-2">
        <Card title="Details">
          <DefinitionList
            items={[
              ['Legal name', s.legalName],
              ['Trade name', s.tradeName ?? '—'],
              [
                'GSTIN',
                s.gstin ? (
                  <span key="g" className="font-mono">
                    {s.gstin}
                  </span>
                ) : (
                  '—'
                ),
              ],
              [
                'Address',
                [s.addressLine1, s.addressLine2, s.city, s.state, s.postalCode]
                  .filter(Boolean)
                  .join(', ') || '—',
              ],
              ['Notes', s.notes ?? '—'],
              ['Last updated', formatDateTime(s.updatedAt)],
            ]}
          />
        </Card>
        <Card
          title="Identity verification"
          description="Checked against one of the supplier’s documents."
          actions={
            s.status === 'PENDING_VERIFICATION' &&
            perms.verify && (
              <Button
                size="sm"
                variant="secondary"
                onClick={() => {
                  setOpen({ kind: 'verify' })
                }}
              >
                <ShieldCheck className="size-3.5" aria-hidden="true" />
                Verify
              </Button>
            )
          }
        >
          {s.status === 'PENDING_VERIFICATION' ? (
            <p role="note" className="rounded-control bg-warning-subtle px-3 py-2 text-sm text-ink">
              <strong>Not verified.</strong> Attach a registration document, then record the
              verification.
            </p>
          ) : (
            <DefinitionList
              items={[
                ['Reference', s.verificationReference ?? '—'],
                [
                  'Document',
                  s.documents.find((d) => d.attachmentId === s.verificationDocumentId)?.fileName ??
                    '—',
                ],
                ['Verified by', personLabel(s.verifiedBy, perms.meId)],
                ['Verified at', formatDateTime(s.verifiedAt)],
              ]}
            />
          )}
        </Card>
      </div>

      <div className="mt-6 flex flex-col gap-6">
        <BrandLinksCard
          supplier={s}
          canAdd={editable}
          canVerify={perms.verify && !archived}
          canEnd={perms.archive}
          onOpen={setOpen}
        />
        <ContactsCard supplier={s} editable={editable} onOpen={setOpen} />
        <DocumentsCard
          supplier={s}
          canAdd={editable && perms.upload}
          canOpen={perms.openFiles}
          onAdd={() => {
            setOpen({ kind: 'document' })
          }}
        />
      </div>

      <SupplierFormDialog open={open?.kind === 'edit'} onClose={close} supplier={s} />
      <VerifySupplierDialog open={open?.kind === 'verify'} onClose={close} supplier={s} />
      <ArchiveSupplierDialog open={open?.kind === 'archive'} onClose={close} supplier={s} />
      <ContactFormDialog open={open?.kind === 'contact'} onClose={close} supplierId={s.id} />
      {open?.kind === 'editContact' && (
        <ContactFormDialog open onClose={close} supplierId={s.id} contact={open.contact} />
      )}
      <DocumentDialog open={open?.kind === 'document'} onClose={close} supplierId={s.id} />
      <AddBrandLinkDialog open={open?.kind === 'link'} onClose={close} supplier={s} />
      {open?.kind === 'verifyLink' && (
        <VerifyBrandLinkDialog open onClose={close} supplier={s} link={open.link} />
      )}
      {open?.kind === 'endLink' && <EndBrandLinkDialog open onClose={close} link={open.link} />}
    </>
  )
}

function BrandLinksCard({
  supplier,
  canAdd,
  canVerify,
  canEnd,
  onOpen,
}: {
  supplier: SupplierDetail
  canAdd: boolean
  canVerify: boolean
  canEnd: boolean
  onOpen: (o: Open) => void
}) {
  const cols: Column<BrandLink>[] = [
    {
      key: 'brand',
      header: 'Brand',
      render: (l) => (
        <Link to={`/admin/catalogue/brands/${l.brandId}`} className={linkClass}>
          {l.brandName}
        </Link>
      ),
    },
    { key: 'status', header: 'Status', render: (l) => <LinkStatusBadge link={l} /> },
    {
      key: 'evidence',
      header: 'Authorisation',
      render: (l) =>
        l.status === 'UNVERIFIED' ? (
          <span className="text-xs text-muted">No evidence recorded</span>
        ) : (
          <span className="text-xs">
            {l.authorisationReference ?? '—'} · {formatDate(l.effectiveFrom)} →{' '}
            {l.effectiveTo ? formatDate(l.effectiveTo) : 'open'}
            {l.endedReason && <span className="block text-muted">Ended: {l.endedReason}</span>}
          </span>
        ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (l) => {
        const b: ReactNode[] = []
        if (l.status === 'UNVERIFIED' && canVerify) {
          b.push(
            <Button
              key="v"
              size="sm"
              variant="secondary"
              onClick={() => {
                onOpen({ kind: 'verifyLink', link: l })
              }}
            >
              Verify
            </Button>,
          )
        }
        if (l.status !== 'ENDED' && canEnd) {
          b.push(
            <Button
              key="e"
              size="sm"
              variant="ghost"
              onClick={() => {
                onOpen({ kind: 'endLink', link: l })
              }}
            >
              End
            </Button>,
          )
        }
        return b.length ? (
          <div className="flex flex-wrap gap-1.5 [&>button]:whitespace-nowrap">{b}</div>
        ) : null
      },
    },
  ]
  return (
    <Card
      title="Brand relationships"
      description="Brands this supplier may supply. Only verified, current relationships count for a brand."
      actions={
        canAdd && (
          <Button
            size="sm"
            onClick={() => {
              onOpen({ kind: 'link' })
            }}
          >
            <Plus className="size-3.5" aria-hidden="true" />
            Add brand
          </Button>
        )
      }
    >
      {supplier.brandLinks.length === 0 ? (
        <EmptyState title="No brand relationships">
          Add a brand this supplier supplies, then verify it with the authorisation document.
        </EmptyState>
      ) : (
        <ResponsiveTable
          caption="Brand relationships"
          columns={cols}
          rows={supplier.brandLinks}
          rowKey={(l) => l.id}
        />
      )}
    </Card>
  )
}

function ContactsCard({
  supplier,
  editable,
  onOpen,
}: {
  supplier: SupplierDetail
  editable: boolean
  onOpen: (o: Open) => void
}) {
  const remove = useSupplierMutation((c: Contact) => removeContact(supplier.id, c.id, c.version))
  const [error, setError] = useState<string | null>(null)
  const cols: Column<Contact>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (c) => (
        <div className="flex flex-col">
          <span className="font-medium">
            {c.name}
            {c.isPrimary && <span className="ml-2 text-xs text-primary">Primary</span>}
            {!c.active && <span className="ml-2 text-xs text-muted">Removed</span>}
          </span>
          {c.role && <span className="text-xs text-muted">{c.role}</span>}
        </div>
      ),
    },
    { key: 'phone', header: 'Phone', render: (c) => c.phone ?? '—' },
    { key: 'email', header: 'Email', render: (c) => c.email ?? '—' },
    {
      key: 'actions',
      header: 'Actions',
      render: (c) =>
        editable && c.active ? (
          <div className="flex flex-wrap gap-1.5">
            <Button
              size="sm"
              variant="secondary"
              onClick={() => {
                onOpen({ kind: 'editContact', contact: c })
              }}
            >
              Edit
            </Button>
            <Button
              size="sm"
              variant="ghost"
              loading={remove.isPending && remove.variables.id === c.id}
              onClick={() => {
                setError(null)
                remove.mutate(c, {
                  onError: (e) => {
                    setError(describeError(e).message)
                  },
                })
              }}
            >
              <UserMinus className="size-3.5" aria-hidden="true" />
              Remove
            </Button>
          </div>
        ) : null,
    },
  ]
  return (
    <Card
      title="Contacts"
      description="Business contacts. Removed contacts are kept for history."
      actions={
        editable && (
          <Button
            size="sm"
            onClick={() => {
              onOpen({ kind: 'contact' })
            }}
          >
            <Plus className="size-3.5" aria-hidden="true" />
            Add contact
          </Button>
        )
      }
    >
      {error && (
        <p role="alert" className="mb-3 text-sm text-danger">
          {error}
        </p>
      )}
      {supplier.contacts.length === 0 ? (
        <EmptyState title="No contacts">No contact has been recorded.</EmptyState>
      ) : (
        <ResponsiveTable
          caption="Contacts"
          columns={cols}
          rows={supplier.contacts}
          rowKey={(c) => c.id}
        />
      )}
    </Card>
  )
}

function DocumentsCard({
  supplier,
  canAdd,
  canOpen,
  onAdd,
}: {
  supplier: SupplierDetail
  canAdd: boolean
  canOpen: boolean
  onAdd: () => void
}) {
  const [error, setError] = useState<string | null>(null)
  const download = async (d: SupplierDocument) => {
    setError(null)
    try {
      const blob = await fetchAttachment(d.attachmentId, 'attachment')
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = d.fileName
      a.rel = 'noopener'
      document.body.append(a)
      a.click()
      a.remove()
      setTimeout(() => {
        URL.revokeObjectURL(url)
      }, 10_000)
    } catch (e) {
      setError(describeError(e).message)
    }
  }
  const cols: Column<SupplierDocument>[] = [
    {
      key: 'file',
      header: 'Document',
      render: (d) => <span className="font-medium">{d.fileName}</span>,
    },
    { key: 'type', header: 'Type', render: (d) => documentTypeLabels[d.documentType] },
    { key: 'added', header: 'Added', render: (d) => formatDateTime(d.createdAt) },
    {
      key: 'open',
      header: 'Open',
      render: (d) =>
        canOpen ? (
          <Button
            size="sm"
            variant="secondary"
            onClick={() => void download(d)}
            aria-label={`Download ${d.fileName}`}
          >
            <Download className="size-3.5" aria-hidden="true" />
            Download
          </Button>
        ) : null,
    },
  ]
  return (
    <Card
      title="Documents"
      description="Private files (GST registration, authorisation letters, agreements). Opened through short-lived links."
      actions={
        canAdd && (
          <Button size="sm" onClick={onAdd}>
            <Plus className="size-3.5" aria-hidden="true" />
            Attach document
          </Button>
        )
      }
    >
      {error && (
        <p role="alert" className="mb-3 text-sm text-danger">
          {error}
        </p>
      )}
      {!canOpen && supplier.documents.length > 0 && (
        <p className="mb-3 text-xs text-muted">
          Opening documents needs permission to view documents.
        </p>
      )}
      {supplier.documents.length === 0 ? (
        <EmptyState title="No documents">No document has been attached.</EmptyState>
      ) : (
        <ResponsiveTable
          caption="Documents"
          columns={cols}
          rows={supplier.documents}
          rowKey={(d) => d.id}
        />
      )}
    </Card>
  )
}
