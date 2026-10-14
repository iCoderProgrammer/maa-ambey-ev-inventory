import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { useMe } from '@/features/access/hooks/useAccess'
import { Permission } from '@/features/access/api/access'
import { AttachmentUploader } from '@/features/attachments/components/AttachmentUploader'
import { useAttachmentLimits, useMyAttachments } from '@/features/attachments/hooks/useAttachments'
import { ActionErrorBox } from '@/features/catalogue/components/ActionErrorBox'
import { useBrands } from '@/features/catalogue/hooks/useCatalogue'
import { describeError, todayISO, type ActionError } from '@/features/catalogue/lib/presentation'
import {
  addContact,
  addDocument,
  archiveSupplier,
  createBrandLink,
  createSupplier,
  documentTypes,
  endBrandLink,
  updateContact,
  updateSupplier,
  verifyBrandLink,
  verifySupplier,
  type BrandLink,
  type Contact,
  type Supplier,
  type SupplierDetail,
} from '../api/suppliers'
import { useRefreshSuppliers, useSupplierMutation } from '../hooks/useSuppliers'
import { documentTypeLabels } from '../lib/labels'

const opt = (max: number, label: string) =>
  z
    .string()
    .trim()
    .max(max, `${label}: at most ${String(max)} characters`)
const orNull = (v: string) => (v.trim() === '' ? null : v.trim())

function useFailure() {
  const [failure, setFailure] = useState<ActionError | null>(null)
  const refresh = useRefreshSuppliers()
  return { failure, setFailure, refresh }
}

function FormDialog({
  title,
  description,
  onClose,
  formId,
  pending,
  submitLabel,
  danger,
  children,
  failure,
  onReload,
}: {
  title: string
  description?: string
  onClose: () => void
  formId: string
  pending: boolean
  submitLabel: string
  danger?: boolean
  children: ReactNode
  failure: ActionError | null
  onReload: () => void
}) {
  return (
    <Dialog
      open
      onClose={onClose}
      title={title}
      description={description}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={pending}>
            Cancel
          </Button>
          <Button
            type="submit"
            form={formId}
            loading={pending}
            variant={danger ? 'danger' : 'primary'}
          >
            {submitLabel}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {children}
        {failure && <ActionErrorBox error={failure} onReload={onReload} />}
      </div>
    </Dialog>
  )
}

// ── Supplier create / edit ──────────────────────────────────────────────────

const supplierSchema = z.object({
  legalName: z.string().trim().min(1, 'Legal name is required').max(200, 'At most 200 characters'),
  tradeName: opt(200, 'Trade name'),
  gstin: z
    .string()
    .trim()
    .transform((v) => v.replace(/\s+/g, '').toUpperCase())
    .refine((v) => v === '' || /^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$/.test(v), {
      message: 'Enter a 15-character GSTIN, or leave it empty if not applicable',
    }),
  addressLine1: opt(200, 'Address'),
  addressLine2: opt(200, 'Address'),
  city: opt(100, 'City'),
  state: opt(100, 'State'),
  postalCode: z
    .string()
    .trim()
    .refine((v) => v === '' || /^[1-9][0-9]{5}$/.test(v), { message: 'Enter a 6-digit PIN code' }),
  notes: opt(1000, 'Notes'),
})
type SupplierValues = z.input<typeof supplierSchema>
const supplierFields = [
  'legalName',
  'tradeName',
  'gstin',
  'addressLine1',
  'addressLine2',
  'city',
  'state',
  'postalCode',
  'notes',
] as const

function SupplierFormBody({
  onClose,
  supplier,
  onSaved,
}: {
  open: boolean
  onClose: () => void
  supplier?: Supplier
  onSaved?: (s: Supplier) => void
}) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const save = useSupplierMutation((v: SupplierValues) => {
    const body = Object.fromEntries(
      supplierFields.map((k) => [k, k === 'legalName' ? v[k].trim() : orNull(v[k])]),
    )
    return supplier
      ? updateSupplier(supplier.id, { version: supplier.version, ...body })
      : createSupplier(body as { legalName: string })
  })
  const form = useForm<SupplierValues, unknown, SupplierValues>({
    resolver: zodResolver(supplierSchema),
    defaultValues: Object.fromEntries(
      supplierFields.map((k) => [k, supplier?.[k] ?? '']),
    ) as SupplierValues,
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      const s = await save.mutateAsync(v)
      onSaved?.(s)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: supplierFields }))
    }
  })
  const field = (
    name: (typeof supplierFields)[number],
    label: string,
    hint?: string,
    required?: boolean,
  ) => (
    <Field label={label} hint={hint} required={required} error={errors[name]?.message}>
      {(p) => <Input {...p} autoComplete="off" {...form.register(name)} />}
    </Field>
  )
  return (
    <FormDialog
      title={supplier ? 'Edit supplier' : 'Add supplier'}
      description={
        supplier
          ? 'Changing the legal name or GSTIN of a verified supplier returns it to pending verification.'
          : 'New suppliers start pending verification. A supplier record does not prove dealership authorisation.'
      }
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel={supplier ? 'Save supplier' : 'Add supplier'}
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <p className="rounded-control bg-warning-subtle px-3 py-2 text-xs text-ink">
          Enter details exactly as on the supplier’s registration documents. Do not guess legal
          names or GSTINs.
        </p>
        {field('legalName', 'Legal name', 'As registered, e.g. on the GST certificate.', true)}
        <div className="grid gap-4 sm:grid-cols-2">
          {field('tradeName', 'Trade name')}
          {field('gstin', 'GSTIN', 'If applicable.')}
        </div>
        {field('addressLine1', 'Address line 1')}
        {field('addressLine2', 'Address line 2')}
        <div className="grid gap-4 sm:grid-cols-3">
          {field('city', 'City')}
          {field('state', 'State')}
          {field('postalCode', 'PIN code')}
        </div>
        {field('notes', 'Notes')}
      </form>
    </FormDialog>
  )
}

export function SupplierFormDialog(props: {
  open: boolean
  onClose: () => void
  supplier?: Supplier
  onSaved?: (s: Supplier) => void
}) {
  return props.open ? <SupplierFormBody {...props} /> : null
}

// ── Contacts ────────────────────────────────────────────────────────────────

const contactSchema = z
  .object({
    name: z.string().trim().min(1, 'Name is required').max(120, 'At most 120 characters'),
    role: opt(80, 'Role'),
    phone: z
      .string()
      .trim()
      .refine((v) => v === '' || /^\+?[0-9][0-9 -]{5,18}[0-9]$/.test(v), {
        message: 'Enter a phone number',
      }),
    email: z
      .string()
      .trim()
      .refine((v) => v === '' || z.email().safeParse(v).success, {
        message: 'Enter an email address',
      }),
    isPrimary: z.boolean(),
  })
  .refine((v) => v.phone !== '' || v.email !== '', {
    message: 'Give a phone number or an email',
    path: ['phone'],
  })
type ContactValues = z.input<typeof contactSchema>

function ContactFormBody({
  onClose,
  supplierId,
  contact,
}: {
  open: boolean
  onClose: () => void
  supplierId: string
  contact?: Contact
}) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const save = useSupplierMutation((v: ContactValues) => {
    const body = {
      name: v.name.trim(),
      role: orNull(v.role),
      phone: orNull(v.phone),
      email: orNull(v.email),
      isPrimary: v.isPrimary,
    }
    return contact
      ? updateContact(supplierId, contact.id, { version: contact.version, ...body })
      : addContact(supplierId, body)
  })
  const form = useForm<ContactValues, unknown, ContactValues>({
    resolver: zodResolver(contactSchema),
    defaultValues: {
      name: contact?.name ?? '',
      role: contact?.role ?? '',
      phone: contact?.phone ?? '',
      email: contact?.email ?? '',
      isPrimary: contact?.isPrimary ?? false,
    },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, { setError: form.setError, names: ['name', 'role', 'phone', 'email'] }),
      )
    }
  })
  return (
    <FormDialog
      title={contact ? 'Edit contact' : 'Add contact'}
      description="A business contact at the supplier. Only what is needed to place orders and receive documents."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel={contact ? 'Save contact' : 'Add contact'}
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <Field label="Name" required error={errors.name?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('name')} />}
        </Field>
        <Field label="Role" hint="e.g. Sales manager" error={errors.role?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('role')} />}
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Phone" error={errors.phone?.message}>
            {(p) => <Input {...p} type="tel" autoComplete="off" {...form.register('phone')} />}
          </Field>
          <Field label="Email" error={errors.email?.message}>
            {(p) => <Input {...p} type="email" autoComplete="off" {...form.register('email')} />}
          </Field>
        </div>
        <label className="flex items-center gap-2 text-sm text-ink">
          <input type="checkbox" className="size-4" {...form.register('isPrimary')} />
          Primary contact (replaces the current primary)
        </label>
      </form>
    </FormDialog>
  )
}

export function ContactFormDialog(props: {
  open: boolean
  onClose: () => void
  supplierId: string
  contact?: Contact
}) {
  return props.open ? <ContactFormBody {...props} /> : null
}

// ── Documents ───────────────────────────────────────────────────────────────

const docSchema = z.object({
  attachmentId: z.string().min(1, 'Choose or upload a document'),
  documentType: z.enum(documentTypes),
  note: opt(300, 'Note'),
})
type DocValues = z.input<typeof docSchema>

function DocumentBody({
  onClose,
  supplierId,
}: {
  open: boolean
  onClose: () => void
  supplierId: string
}) {
  const formId = useId()
  const me = useMe()
  const canView = me.data?.permissions.includes(Permission.AttachmentsView) ?? false
  const mine = useMyAttachments(canView)
  const limits = useAttachmentLimits(true)
  const { failure, setFailure, refresh } = useFailure()
  const save = useSupplierMutation((v: DocValues) =>
    addDocument(supplierId, {
      attachmentId: v.attachmentId,
      documentType: v.documentType,
      note: v.note.trim() || undefined,
    }),
  )
  const form = useForm<DocValues, unknown, DocValues>({
    resolver: zodResolver(docSchema),
    defaultValues: { attachmentId: '', documentType: 'OTHER', note: '' },
  })
  const attachmentId = useWatch({ control: form.control, name: 'attachmentId' })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, {
          setError: form.setError,
          names: ['attachmentId', 'documentType', 'note'],
        }),
      )
    }
  })
  return (
    <FormDialog
      title="Attach document"
      description="Attach one of your private uploads. It stays private: only people allowed to view suppliers and documents can open it."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="Attach document"
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <Field label="Document type" required error={errors.documentType?.message}>
          {(p) => (
            <Select
              {...p}
              options={documentTypes.map((t) => ({ value: t, label: documentTypeLabels[t] }))}
              {...form.register('documentType')}
            />
          )}
        </Field>
        {canView && (
          <Field
            label="Your uploads"
            hint={mine.isPending ? 'Loading your uploads…' : undefined}
            error={errors.attachmentId?.message}
          >
            {(p) => (
              <Select
                {...p}
                options={(mine.data ?? []).map((a) => ({ value: a.id, label: a.fileName }))}
                placeholder="Choose a document"
                value={attachmentId}
                onChange={(e) => {
                  form.setValue('attachmentId', e.target.value, {
                    shouldValidate: form.formState.isSubmitted,
                  })
                }}
              />
            )}
          </Field>
        )}
        {limits.data && (
          <div className="rounded-control border border-border p-3">
            <p className="mb-2 text-sm font-medium text-ink">Upload a new document</p>
            <AttachmentUploader
              limits={limits.data}
              onUploaded={(a) => {
                form.setValue('attachmentId', a.id, { shouldValidate: true })
              }}
            />
          </div>
        )}
        {!canView && errors.attachmentId && (
          <p role="alert" className="text-xs font-medium text-danger">
            {errors.attachmentId.message}
          </p>
        )}
        <Field label="Note" error={errors.note?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('note')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function DocumentDialog(props: { open: boolean; onClose: () => void; supplierId: string }) {
  return props.open ? <DocumentBody {...props} /> : null
}

// ── Verify supplier ─────────────────────────────────────────────────────────

const verifySchema = z.object({
  reference: z.string().trim().min(1, 'Reference is required').max(200, 'At most 200 characters'),
  documentId: z.string().min(1, 'Choose the document you checked'),
})
type VerifyValues = z.input<typeof verifySchema>

function docOptions(s: SupplierDetail) {
  return s.documents.map((d) => ({
    value: d.attachmentId,
    label: `${documentTypeLabels[d.documentType]} — ${d.fileName}`,
  }))
}

function VerifyBody({
  onClose,
  supplier,
}: {
  open: boolean
  onClose: () => void
  supplier: SupplierDetail
}) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const save = useSupplierMutation((v: VerifyValues) =>
    verifySupplier(supplier.id, {
      version: supplier.version,
      reference: v.reference.trim(),
      documentId: v.documentId,
    }),
  )
  const form = useForm<VerifyValues, unknown, VerifyValues>({
    resolver: zodResolver(verifySchema),
    defaultValues: { reference: '', documentId: '' },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['reference', 'documentId'] }))
    }
  })
  return (
    <FormDialog
      title="Verify supplier"
      description={`Record that the identity of “${supplier.legalName}” was checked against a document. You are recorded as the verifier.`}
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="Verify supplier"
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <p className="rounded-control bg-info-subtle px-3 py-2 text-xs text-ink">
          This verifies the supplier’s identity only. Authorisation to supply a brand is verified
          per brand relationship.
        </p>
        <Field
          label="Evidence document"
          required
          hint="One of this supplier’s attached documents."
          error={errors.documentId?.message}
        >
          {(p) => (
            <Select
              {...p}
              options={docOptions(supplier)}
              placeholder="Choose a document"
              {...form.register('documentId')}
            />
          )}
        </Field>
        <Field
          label="Reference"
          required
          hint="What was checked, e.g. GST certificate number."
          error={errors.reference?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('reference')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function VerifySupplierDialog(props: {
  open: boolean
  onClose: () => void
  supplier: SupplierDetail
}) {
  return props.open ? <VerifyBody {...props} /> : null
}

// ── Archive supplier ────────────────────────────────────────────────────────

const reasonSchema = z.object({ reason: opt(300, 'Reason') })
type ReasonValues = z.input<typeof reasonSchema>

function ArchiveBody({
  onClose,
  supplier,
}: {
  open: boolean
  onClose: () => void
  supplier: Supplier
}) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const save = useSupplierMutation((v: ReasonValues) =>
    archiveSupplier(supplier.id, {
      version: supplier.version,
      reason: v.reason.trim() || undefined,
    }),
  )
  const form = useForm<ReasonValues, unknown, ReasonValues>({
    resolver: zodResolver(reasonSchema),
    defaultValues: { reason: '' },
  })
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['reason'] }))
    }
  })
  return (
    <FormDialog
      title="Archive supplier"
      description={`“${supplier.legalName}” will be archived. Nothing is deleted.`}
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="Archive supplier"
      danger
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-3 text-sm"
      >
        <ul className="ml-5 list-disc space-y-1 text-ink">
          <li>Contacts, documents and brand relationships stay as history.</li>
          <li>
            The supplier no longer counts as a verified supplier for any brand, and cannot be
            changed.
          </li>
        </ul>
        <Field
          label="Reason"
          hint="Optional. Kept in the audit log."
          error={form.formState.errors.reason?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('reason')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function ArchiveSupplierDialog(props: {
  open: boolean
  onClose: () => void
  supplier: Supplier
}) {
  return props.open ? <ArchiveBody {...props} /> : null
}

// ── Brand relationships ─────────────────────────────────────────────────────

const linkSchema = z.object({
  brandId: z.string().min(1, 'Choose a brand'),
  note: opt(300, 'Note'),
})
type LinkValues = z.input<typeof linkSchema>

function AddLinkBody({
  onClose,
  supplier,
}: {
  open: boolean
  onClose: () => void
  supplier: SupplierDetail
}) {
  const formId = useId()
  const brands = useBrands({ limit: 100 })
  const open = new Set(
    supplier.brandLinks.filter((l) => l.status !== 'ENDED').map((l) => l.brandId),
  )
  const options = (brands.data?.items ?? [])
    .filter((b) => b.lifecycleStatus !== 'ARCHIVED' && !open.has(b.id))
    .map((b) => ({ value: b.id, label: b.name }))
  const { failure, setFailure, refresh } = useFailure()
  const save = useSupplierMutation((v: LinkValues) =>
    createBrandLink(supplier.id, { brandId: v.brandId, note: v.note.trim() || undefined }),
  )
  const form = useForm<LinkValues, unknown, LinkValues>({
    resolver: zodResolver(linkSchema),
    defaultValues: { brandId: '', note: '' },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['brandId', 'note'] }))
    }
  })
  return (
    <FormDialog
      title="Add brand relationship"
      description="Records that this supplier may supply a brand. It starts unverified and proves nothing until verified with an authorisation document."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="Add relationship"
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <Field
          label="Brand"
          required
          hint={
            brands.isPending
              ? 'Loading brands…'
              : options.length === 0
                ? 'Every brand already has an open relationship with this supplier.'
                : 'Archived brands and brands already linked are not offered.'
          }
          error={errors.brandId?.message}
        >
          {(p) => (
            <Select
              {...p}
              options={options}
              placeholder="Choose a brand"
              {...form.register('brandId')}
            />
          )}
        </Field>
        <Field label="Note" error={errors.note?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('note')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function AddBrandLinkDialog(props: {
  open: boolean
  onClose: () => void
  supplier: SupplierDetail
}) {
  return props.open ? <AddLinkBody {...props} /> : null
}

const linkVerifySchema = z
  .object({
    authorisationReference: z
      .string()
      .trim()
      .min(1, 'Reference is required')
      .max(200, 'At most 200 characters'),
    evidenceDocumentId: z.string().min(1, 'Choose the authorisation document'),
    effectiveFrom: z.string().min(1, 'Start date is required'),
    effectiveTo: z.string(),
  })
  .refine((v) => v.effectiveTo === '' || v.effectiveTo >= v.effectiveFrom, {
    message: 'End must not be before the start',
    path: ['effectiveTo'],
  })
type LinkVerifyValues = z.input<typeof linkVerifySchema>

function VerifyLinkBody({
  onClose,
  supplier,
  link,
}: {
  open: boolean
  onClose: () => void
  supplier: SupplierDetail
  link: BrandLink
}) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const save = useSupplierMutation((v: LinkVerifyValues) =>
    verifyBrandLink(link.id, {
      version: link.version,
      authorisationReference: v.authorisationReference.trim(),
      evidenceDocumentId: v.evidenceDocumentId,
      effectiveFrom: v.effectiveFrom,
      effectiveTo: v.effectiveTo || null,
    }),
  )
  const form = useForm<LinkVerifyValues, unknown, LinkVerifyValues>({
    resolver: zodResolver(linkVerifySchema),
    defaultValues: {
      authorisationReference: '',
      evidenceDocumentId: '',
      effectiveFrom: '',
      effectiveTo: '',
    },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, {
          setError: form.setError,
          names: ['authorisationReference', 'evidenceDocumentId', 'effectiveFrom', 'effectiveTo'],
        }),
      )
    }
  })
  return (
    <FormDialog
      title={`Verify relationship with ${link.brandName}`}
      description="Record the authorisation document that allows this supplier to supply the brand. You are recorded as the verifier."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="Verify relationship"
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        {supplier.status !== 'VERIFIED' && (
          <p className="rounded-control bg-warning-subtle px-3 py-2 text-xs text-ink">
            Verify the supplier itself first; the server refuses this until then.
          </p>
        )}
        <Field label="Authorisation document" required error={errors.evidenceDocumentId?.message}>
          {(p) => (
            <Select
              {...p}
              options={docOptions(supplier)}
              placeholder="Choose a document"
              {...form.register('evidenceDocumentId')}
            />
          )}
        </Field>
        <Field
          label="Authorisation reference"
          required
          hint="Agreement or letter number."
          error={errors.authorisationReference?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('authorisationReference')} />}
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Valid from" required error={errors.effectiveFrom?.message}>
            {(p) => <Input {...p} type="date" {...form.register('effectiveFrom')} />}
          </Field>
          <Field label="Valid until" hint="Optional." error={errors.effectiveTo?.message}>
            {(p) => <Input {...p} type="date" {...form.register('effectiveTo')} />}
          </Field>
        </div>
      </form>
    </FormDialog>
  )
}

export function VerifyBrandLinkDialog(props: {
  open: boolean
  onClose: () => void
  supplier: SupplierDetail
  link: BrandLink
}) {
  return props.open ? <VerifyLinkBody {...props} /> : null
}

const endSchema = z.object({
  reason: z.string().trim().min(1, 'Reason is required').max(300, 'At most 300 characters'),
  effectiveTo: z.string(),
})
type EndValues = z.input<typeof endSchema>

function EndLinkBody({ onClose, link }: { open: boolean; onClose: () => void; link: BrandLink }) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const save = useSupplierMutation((v: EndValues) =>
    endBrandLink(link.id, {
      version: link.version,
      reason: v.reason.trim(),
      effectiveTo: v.effectiveTo || null,
    }),
  )
  const form = useForm<EndValues, unknown, EndValues>({
    resolver: zodResolver(endSchema),
    defaultValues: { reason: '', effectiveTo: todayISO() },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['reason', 'effectiveTo'] }))
    }
  })
  return (
    <FormDialog
      title={`End relationship with ${link.brandName}`}
      description="The relationship is kept as history with its end date and reason. It can no longer count for the brand."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="End relationship"
      danger
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <Field label="Reason" required error={errors.reason?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('reason')} />}
        </Field>
        <Field label="End date" error={errors.effectiveTo?.message}>
          {(p) => <Input {...p} type="date" {...form.register('effectiveTo')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function EndBrandLinkDialog(props: { open: boolean; onClose: () => void; link: BrandLink }) {
  return props.open ? <EndLinkBody {...props} /> : null
}
