import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { AttachmentUploader } from '@/features/attachments/components/AttachmentUploader'
import { useAttachmentLimits, useMyAttachments } from '@/features/attachments/hooks/useAttachments'
import type { CatalogueItemKind } from '../api/catalogue'
import type { Brand } from '../api/contracts'
import {
  useArchiveBrand,
  useArchiveItem,
  useActivateBrand,
  useActivateItem,
  useRecordBrandAuthorisation,
  useRefreshCatalogue,
  useVerifyItem,
} from '../hooks/useCatalogue'
import { describeError, todayISO, type ActionError } from '../lib/presentation'
import {
  archiveFormSchema,
  authorisationFormSchema,
  orNull,
  verifyFormSchema,
  type ArchiveFormValues,
  type AuthorisationFormValues,
  type VerifyFormValues,
} from '../lib/schemas'
import { ActionErrorBox } from './ActionErrorBox'

const kindLabel: Record<CatalogueItemKind, string> = {
  models: 'model',
  variants: 'variant',
  colours: 'colour',
}

export interface ItemRef {
  id: string
  name: string
  version: number
}

function useFailure() {
  const [failure, setFailure] = useState<ActionError | null>(null)
  const refresh = useRefreshCatalogue()
  return { failure, setFailure, refresh }
}

// ── Private document picker (B-09 attachments) ──────────────────────────────

/**
 * Picks one of the user's own private uploads or uploads a new file. The file
 * stays private; only its ID is recorded. Needs the attachment permissions.
 */
function DocumentPicker({
  label,
  value,
  onChange,
  error,
  required,
}: {
  label: string
  value: string
  onChange: (id: string) => void
  error?: string
  required?: boolean
}) {
  const me = useMe()
  const perms = me.data?.permissions ?? []
  const canView = perms.includes(Permission.AttachmentsView)
  const canUpload = perms.includes(Permission.AttachmentsUpload)
  const mine = useMyAttachments(canView)
  const limits = useAttachmentLimits(canUpload)
  const options = (mine.data ?? []).map((a) => ({ value: a.id, label: a.fileName }))

  if (!canView && !canUpload) {
    return (
      <div className="flex flex-col gap-1">
        <p className="text-sm font-medium text-ink">{label}</p>
        <p className="text-xs text-muted">
          Attaching a document needs permission to upload and view documents. Ask the owner or your
          administrator.
        </p>
        {error && (
          <p role="alert" className="text-xs font-medium text-danger">
            {error}
          </p>
        )}
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-3">
      {canView && (
        <Field
          label={label}
          required={required}
          hint={
            mine.isPending
              ? 'Loading your uploads…'
              : 'Your own private uploads. The document stays private; only a reference is saved.'
          }
          error={error}
        >
          {(p) => (
            <Select
              {...p}
              options={options}
              placeholder={options.length === 0 ? 'No uploads yet' : 'Choose a document'}
              value={value}
              onChange={(e) => {
                onChange(e.target.value)
              }}
            />
          )}
        </Field>
      )}
      {canUpload && limits.data && (
        <details className="rounded-control border border-border p-3 text-sm">
          <summary className="cursor-pointer font-medium text-ink">Upload a new document</summary>
          <div className="mt-3">
            <AttachmentUploader
              limits={limits.data}
              onUploaded={(a) => {
                onChange(a.id)
              }}
            />
          </div>
        </details>
      )}
    </div>
  )
}

// ── Verify ──────────────────────────────────────────────────────────────────

function VerifyDialogBody({
  open,
  onClose,
  kind,
  item,
}: {
  open: boolean
  onClose: () => void
  kind: CatalogueItemKind
  item: ItemRef
}) {
  const formId = useId()
  const verify = useVerifyItem(kind)
  const { failure, setFailure, refresh } = useFailure()
  const defaults: VerifyFormValues = {
    confidence: 'SUPPLIER_CONFIRMED',
    sourceUrl: '',
    sourceDocumentId: '',
    sourceDate: '',
  }
  const form = useForm<VerifyFormValues, unknown, VerifyFormValues>({
    resolver: zodResolver(verifyFormSchema),
    defaultValues: defaults,
  })
  const docId = useWatch({ control: form.control, name: 'sourceDocumentId' })

  const onSubmit = form.handleSubmit(async (v) => {
    if (verify.isPending) return
    setFailure(null)
    try {
      await verify.mutateAsync({
        id: item.id,
        input: {
          version: item.version,
          confidence: v.confidence,
          sourceUrl: orNull(v.sourceUrl),
          sourceDocumentId: orNull(v.sourceDocumentId),
          sourceDate: v.sourceDate,
        },
      })
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, {
          setError: form.setError,
          names: ['confidence', 'sourceUrl', 'sourceDocumentId', 'sourceDate'],
        }),
      )
    }
  })

  const errors = form.formState.errors
  const label = kindLabel[kind]
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={`Record ${label} verification`}
      description={`Record where the details of “${item.name}” were confirmed. You are recorded as the verifier.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={verify.isPending}>
            Cancel
          </Button>
          <Button type="submit" form={formId} loading={verify.isPending}>
            Record verification
          </Button>
        </>
      }
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <p className="rounded-control bg-info-subtle px-3 py-2 text-xs text-ink">
          Only record verification from a real supplier or client source. Public web pages are
          reference material, not confirmation.
        </p>
        <Field label="Confirmed by" required error={errors.confidence?.message}>
          {(p) => (
            <Select
              {...p}
              options={[
                { value: 'SUPPLIER_CONFIRMED', label: 'Supplier confirmed' },
                { value: 'CLIENT_APPROVED', label: 'Client approved' },
              ]}
              {...form.register('confidence')}
            />
          )}
        </Field>
        <Field
          label="Source link"
          hint="http or https link to the supplier/client source. Give a link, a document or both."
          error={errors.sourceUrl?.message}
        >
          {(p) => <Input {...p} type="url" inputMode="url" {...form.register('sourceUrl')} />}
        </Field>
        <DocumentPicker
          label="Source document"
          value={docId}
          onChange={(id) => {
            form.setValue('sourceDocumentId', id, { shouldValidate: form.formState.isSubmitted })
          }}
          error={errors.sourceDocumentId?.message}
        />
        <Field
          label="Source date"
          required
          hint="Date of the source document or confirmation."
          error={errors.sourceDate?.message}
        >
          {(p) => <Input {...p} type="date" max={todayISO()} {...form.register('sourceDate')} />}
        </Field>
        {failure && (
          <ActionErrorBox
            error={failure}
            onReload={() => {
              void refresh()
              onClose()
            }}
          />
        )}
      </form>
    </Dialog>
  )
}

// ── Confirmations ───────────────────────────────────────────────────────────

function ConfirmDialog({
  open,
  onClose,
  title,
  description,
  children,
  confirmLabel,
  danger,
  pending,
  failure,
  onReload,
  onConfirm,
  confirmDisabled,
}: {
  open: boolean
  onClose: () => void
  title: string
  description: string
  children?: ReactNode
  confirmLabel: string
  danger?: boolean
  pending: boolean
  failure: ActionError | null
  onReload: () => void
  onConfirm: () => void
  confirmDisabled?: boolean
}) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      description={description}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={pending}>
            Cancel
          </Button>
          <Button
            variant={danger ? 'danger' : 'primary'}
            onClick={onConfirm}
            loading={pending}
            disabled={confirmDisabled}
          >
            {confirmLabel}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3 text-sm">
        {children}
        {failure && <ActionErrorBox error={failure} onReload={onReload} />}
      </div>
    </Dialog>
  )
}

function ActivateItemDialogBody({
  open,
  onClose,
  kind,
  item,
  parentNote,
}: {
  open: boolean
  onClose: () => void
  kind: CatalogueItemKind
  item: ItemRef
  /** e.g. "The model must be active first." */
  parentNote?: string
}) {
  const activate = useActivateItem(kind)
  const { failure, setFailure, refresh } = useFailure()
  const label = kindLabel[kind]
  return (
    <ConfirmDialog
      open={open}
      onClose={onClose}
      title={`Activate ${label}`}
      description={`“${item.name}” becomes Active. It can then be used by later workflows when its brand is enabled for purchase and sales.`}
      confirmLabel={`Activate ${label}`}
      pending={activate.isPending}
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
      onConfirm={() => {
        if (activate.isPending) return
        setFailure(null)
        activate.mutate(
          { id: item.id, version: item.version },
          {
            onSuccess: onClose,
            onError: (e) => {
              setFailure(describeError(e))
            },
          },
        )
      }}
    >
      <ul className="ml-5 list-disc space-y-1 text-ink">
        <li>
          Active items cannot be edited afterwards (approved data is not overwritten); they can only
          be archived.
        </li>
        {parentNote && <li>{parentNote}</li>}
        <li>
          The server checks verification and the parent’s status again and refuses if they are not
          met.
        </li>
      </ul>
    </ConfirmDialog>
  )
}

function ActivateBrandDialogBody({
  open,
  onClose,
  brand,
  ready,
}: {
  open: boolean
  onClose: () => void
  brand: Brand
  /** Readiness from the server; the server re-checks on confirm. */
  ready: boolean
}) {
  const activate = useActivateBrand()
  const { failure, setFailure, refresh } = useFailure()
  return (
    <ConfirmDialog
      open={open}
      onClose={onClose}
      title="Enable brand for purchase and sales"
      description={`“${brand.name}” becomes Active. With its authorisation, staff can then use its active models and variants in purchasing and sales.`}
      confirmLabel="Enable for purchase & sales"
      pending={activate.isPending}
      failure={failure}
      confirmDisabled={!ready}
      onReload={() => {
        void refresh()
        onClose()
      }}
      onConfirm={() => {
        if (activate.isPending) return
        setFailure(null)
        activate.mutate(
          { id: brand.id, version: brand.version },
          {
            onSuccess: onClose,
            onError: (e) => {
              setFailure(describeError(e))
            },
          },
        )
      }}
    >
      {!ready && (
        <p className="rounded-control bg-warning-subtle px-3 py-2 text-ink">
          Not all requirements are met yet (see Activation readiness). The server would refuse this
          activation.
        </p>
      )}
      <p className="text-muted">This action is recorded in the audit log with your name.</p>
    </ConfirmDialog>
  )
}

function ArchiveDialogBody({
  open,
  onClose,
  kind,
  item,
  childrenNote,
}: {
  open: boolean
  onClose: () => void
  kind: CatalogueItemKind | 'brands'
  item: ItemRef
  childrenNote?: string
}) {
  const formId = useId()
  const archiveBrand = useArchiveBrand()
  const archiveModel = useArchiveItem('models')
  const archiveVariant = useArchiveItem('variants')
  const archiveColour = useArchiveItem('colours')
  const { failure, setFailure, refresh } = useFailure()
  const form = useForm<ArchiveFormValues, unknown, ArchiveFormValues>({
    resolver: zodResolver(archiveFormSchema),
    defaultValues: { reason: '' },
  })
  const pending =
    archiveBrand.isPending ||
    archiveModel.isPending ||
    archiveVariant.isPending ||
    archiveColour.isPending
  const label = kind === 'brands' ? 'brand' : kindLabel[kind]

  const onSubmit = form.handleSubmit(async (v) => {
    if (pending) return
    setFailure(null)
    const input = { version: item.version, ...(v.reason ? { reason: v.reason } : {}) }
    try {
      if (kind === 'brands') await archiveBrand.mutateAsync({ id: item.id, input })
      else if (kind === 'models') await archiveModel.mutateAsync({ id: item.id, input })
      else if (kind === 'variants') await archiveVariant.mutateAsync({ id: item.id, input })
      else await archiveColour.mutateAsync({ id: item.id, input })
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['reason'] }))
    }
  })

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={`Archive ${label}`}
      description={`“${item.name}” will be archived. Nothing is deleted.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={pending}>
            Cancel
          </Button>
          <Button variant="danger" type="submit" form={formId} loading={pending}>
            Archive {label}
          </Button>
        </>
      }
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-3 text-sm"
      >
        <ul className="ml-5 list-disc space-y-1 text-ink">
          <li>
            It stays visible for history and keeps every reference, but nothing new can be added
            under it.
          </li>
          <li>Archived records cannot be edited, activated or restored from here.</li>
          {childrenNote && <li>{childrenNote}</li>}
        </ul>
        <Field
          label="Reason"
          hint="Optional. Kept in the audit log."
          error={form.formState.errors.reason?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('reason')} />}
        </Field>
        {failure && (
          <ActionErrorBox
            error={failure}
            onReload={() => {
              void refresh()
              onClose()
            }}
          />
        )}
      </form>
    </Dialog>
  )
}

// ── Brand authorisation ─────────────────────────────────────────────────────

function AuthorisationDialogBody({
  open,
  onClose,
  brand,
}: {
  open: boolean
  onClose: () => void
  brand: Brand
}) {
  const formId = useId()
  const record = useRecordBrandAuthorisation()
  const { failure, setFailure, refresh } = useFailure()
  const defaults = (): AuthorisationFormValues => ({
    authorisationStatus: brand.authorisationStatus,
    manufacturerLegalName: brand.manufacturerLegalName ?? '',
    authorisationReference: brand.authorisationReference ?? '',
    authorisationEvidenceId: brand.authorisationEvidenceId ?? '',
  })
  const form = useForm<AuthorisationFormValues, unknown, AuthorisationFormValues>({
    resolver: zodResolver(authorisationFormSchema),
    defaultValues: defaults(),
  })
  const status = useWatch({ control: form.control, name: 'authorisationStatus' })
  const evidence = useWatch({ control: form.control, name: 'authorisationEvidenceId' })

  const onSubmit = form.handleSubmit(async (v) => {
    if (record.isPending) return
    setFailure(null)
    try {
      await record.mutateAsync({
        id: brand.id,
        input: {
          version: brand.version,
          authorisationStatus: v.authorisationStatus,
          manufacturerLegalName: orNull(v.manufacturerLegalName),
          authorisationReference: orNull(v.authorisationReference),
          authorisationEvidenceId: orNull(v.authorisationEvidenceId),
        },
      })
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, {
          setError: form.setError,
          names: [
            'authorisationStatus',
            'manufacturerLegalName',
            'authorisationReference',
            'authorisationEvidenceId',
          ],
        }),
      )
    }
  })

  const errors = form.formState.errors
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Record brand authorisation"
      description={`Record the verified supplier/manufacturer authorisation for “${brand.name}”. You are recorded as the verifier with today’s date.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={record.isPending}>
            Cancel
          </Button>
          <Button type="submit" form={formId} loading={record.isPending}>
            Save authorisation
          </Button>
        </>
      }
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <p className="rounded-control bg-warning-subtle px-3 py-2 text-xs text-ink">
          Being on the showroom’s brand list is not proof of authorisation. Record Authorised only
          with a real dealership/supplier document, and use the legal name from that document — do
          not assume related brand names are the same legal entity.
        </p>
        <Field label="Authorisation status" required error={errors.authorisationStatus?.message}>
          {(p) => (
            <Select
              {...p}
              options={[
                { value: 'UNVERIFIED', label: 'Unverified' },
                { value: 'AUTHORISED', label: 'Authorised' },
                { value: 'NOT_AUTHORISED', label: 'Not authorised' },
                { value: 'REVOKED', label: 'Revoked' },
              ]}
              {...form.register('authorisationStatus')}
            />
          )}
        </Field>
        <Field
          label="Supplier/manufacturer legal name"
          required={status === 'AUTHORISED'}
          error={errors.manufacturerLegalName?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('manufacturerLegalName')} />}
        </Field>
        <Field
          label="Authorisation reference"
          required={status === 'AUTHORISED'}
          hint="Agreement or letter number from the document."
          error={errors.authorisationReference?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('authorisationReference')} />}
        </Field>
        <DocumentPicker
          label="Authorisation document"
          required={status === 'AUTHORISED'}
          value={evidence}
          onChange={(id) => {
            form.setValue('authorisationEvidenceId', id, {
              shouldValidate: form.formState.isSubmitted,
            })
          }}
          error={errors.authorisationEvidenceId?.message}
        />
        {brand.lifecycleStatus === 'ACTIVE' && status !== 'AUTHORISED' && (
          <p className="rounded-control bg-danger-subtle px-3 py-2 text-xs text-ink">
            This brand is active. Saving a status other than Authorised stops its use for purchase
            and sales immediately.
          </p>
        )}
        {failure && (
          <ActionErrorBox
            error={failure}
            onReload={() => {
              void refresh()
              onClose()
            }}
          />
        )}
      </form>
    </Dialog>
  )
}

/** Mounted only while open, so every opening starts from fresh values. */
export function VerifyDialog(props: Parameters<typeof VerifyDialogBody>[0]) {
  return props.open ? <VerifyDialogBody {...props} /> : null
}

/** Mounted only while open, so every opening starts from fresh values. */
export function ActivateItemDialog(props: Parameters<typeof ActivateItemDialogBody>[0]) {
  return props.open ? <ActivateItemDialogBody {...props} /> : null
}

/** Mounted only while open, so every opening starts from fresh values. */
export function ActivateBrandDialog(props: Parameters<typeof ActivateBrandDialogBody>[0]) {
  return props.open ? <ActivateBrandDialogBody {...props} /> : null
}

/** Mounted only while open, so every opening starts from fresh values. */
export function ArchiveDialog(props: Parameters<typeof ArchiveDialogBody>[0]) {
  return props.open ? <ArchiveDialogBody {...props} /> : null
}

/** Mounted only while open, so every opening starts from fresh values. */
export function AuthorisationDialog(props: Parameters<typeof AuthorisationDialogBody>[0]) {
  return props.open ? <AuthorisationDialogBody {...props} /> : null
}
