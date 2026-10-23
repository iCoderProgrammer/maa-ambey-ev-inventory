import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'
import { ApiError, ClientErrorCode } from '@/lib/api/client'
import { CatalogueErrorCode } from '../api/catalogue'
import type {
  AuthorisationStatus,
  Confidence,
  LifecycleStatus,
  ReadinessCheck,
} from '../api/contracts'

/**
 * Labels for the status value sets the database defines (C-01 CHECKs). Brands,
 * models and other catalogue data always come from the API.
 */
export const lifecycleLabels: Record<LifecycleStatus, string> = {
  PENDING_VERIFICATION: 'Pending verification',
  ACTIVE: 'Active',
  ARCHIVED: 'Archived',
}

export const authorisationLabels: Record<AuthorisationStatus, string> = {
  UNVERIFIED: 'Authorisation unverified',
  AUTHORISED: 'Authorised',
  NOT_AUTHORISED: 'Not authorised',
  REVOKED: 'Authorisation revoked',
}

export const confidenceLabels: Record<Confidence, string> = {
  UNVERIFIED: 'Unverified',
  SUPPLIER_CONFIRMED: 'Supplier confirmed',
  CLIENT_APPROVED: 'Client approved',
}

/** Readable names for the server's readiness check keys; unknown keys fall back to the key. */
const checkLabels: Record<string, string> = {
  lifecycle: 'Pending verification (not archived or already active)',
  authorisation: 'Authorisation recorded as AUTHORISED',
  legal_identity: 'Supplier/manufacturer legal identity',
  authorisation_evidence: 'Authorisation reference, evidence, verifier and date',
  verified_catalogue: 'At least one active, verified model and variant',
  supplier_link: 'Verified supplier relationship',
  approved_prices: 'Approved, effective-dated prices',
  warranty_terms: 'Approved warranty terms',
  identifier_rules: 'Approved vehicle identifier rules',
  verification: 'Source and verification recorded',
  parent: 'Parent is active',
  // Price approval (C-04)
  price_list: 'Price list is active',
  brand: 'Brand is not archived',
  variant: 'Model and variant are not archived',
  variant_verification: 'Variant is verified',
  // Purchase order submission and approval (D-02)
  lines: 'At least one line',
  expected_date: 'Expected delivery date',
  supplier_verified: 'Verified supplier',
  supplier_brand_link: 'Current verified supplier relationship with the brand',
  brand_active: 'Brand activated for purchase',
  model_active: 'Active model',
  variant_active: 'Active variant',
}

export function checkLabel(key: string): string {
  return checkLabels[key] ?? key
}

export function readinessSummary(checks: ReadinessCheck[]) {
  return {
    met: checks.filter((c) => c.met).length,
    pending: checks.filter((c) => !c.met && c.pending).length,
    open: checks.filter((c) => !c.met && !c.pending).length,
  }
}

export function formatDateTime(value: string | null): string {
  if (!value) return '—'
  return new Intl.DateTimeFormat('en-IN', { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(value),
  )
}

export function formatDate(value: string | null): string {
  if (!value) return '—'
  return new Intl.DateTimeFormat('en-IN', { dateStyle: 'medium', timeZone: 'UTC' }).format(
    new Date(`${value}T00:00:00Z`),
  )
}

/** Today's date (YYYY-MM-DD) in the user's time zone, for date inputs. */
export function todayISO(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** A user ID shown as "you" for the signed-in user, otherwise shortened (no name lookup API yet). */
export function personLabel(userId: string | null, meId: string | undefined): string {
  if (!userId) return '—'
  if (userId === meId) return 'You'
  return `User ${userId.slice(0, 8)}`
}

export interface ActionError {
  /** Message for the dialog or page. */
  message: string
  /** Server unmet checks for ACTIVATION_BLOCKED. */
  blockers?: { key: string; message: string }[]
  /** The record changed meanwhile: offer to reload. */
  stale?: boolean
  requestId?: string
}

/**
 * Turns a failed catalogue call into a message, and puts field-level problems
 * (VALIDATION_FAILED, DUPLICATE) on the form fields that exist.
 */
export function describeError<T extends FieldValues>(
  error: unknown,
  fields?: { setError: UseFormSetError<T>; names: readonly string[] },
): ActionError {
  if (!(error instanceof ApiError)) {
    return { message: 'Something went wrong. Please try again.' }
  }
  const base = { requestId: error.requestId }
  const placed = new Set<string>()
  if (
    fields &&
    (error.code === 'VALIDATION_FAILED' || error.code === CatalogueErrorCode.Duplicate)
  ) {
    for (const d of error.details) {
      if (fields.names.includes(d.field)) {
        fields.setError(d.field as Path<T>, {
          message:
            error.code === CatalogueErrorCode.Duplicate
              ? 'Already used — choose another'
              : d.message,
        })
        placed.add(d.field)
      }
    }
  }
  const unplaced = error.details.filter((d) => !placed.has(d.field))
  switch (error.code) {
    case ClientErrorCode.Network:
      return {
        ...base,
        message: 'Cannot reach the server. Nothing was saved. Check your connection and try again.',
      }
    case 'FORBIDDEN':
      return { ...base, message: 'You do not have permission to do this. Nothing was changed.' }
    case 'CSRF_FAILED':
      return {
        ...base,
        message: 'Your session security token expired. Reload the page and try again.',
      }
    case CatalogueErrorCode.VersionConflict:
      return {
        ...base,
        stale: true,
        message:
          'Someone else changed this record after you opened it. Reload to see the latest version, then try again.',
      }
    case CatalogueErrorCode.ActivationBlocked:
      return {
        ...base,
        message: 'Activation is not allowed yet. The server reported these unmet requirements:',
        blockers: error.details.map((d) => ({ key: d.field, message: d.message })),
      }
    case 'APPROVAL_BLOCKED':
      return {
        ...base,
        message: 'This price cannot be approved yet. The server reported these unmet requirements:',
        blockers: error.details.map((d) => ({ key: d.field, message: d.message })),
      }
    case 'PRICE_OVERLAP':
      return {
        ...base,
        message:
          'An approved price already covers part of this period. Supersede that price instead, or change the dates. Nothing was approved.',
      }
    case 'SELF_APPROVAL_NOT_ALLOWED':
      return {
        ...base,
        message: 'You drafted this price, so another authorised person must approve it.',
      }
    case CatalogueErrorCode.Duplicate:
      return { ...base, message: placed.size > 0 ? 'That value is already used.' : error.message }
    case 'VALIDATION_FAILED':
      return {
        ...base,
        message:
          unplaced.length > 0
            ? `Check these fields: ${unplaced.map((d) => `${d.field} ${d.message}`).join('; ')}.`
            : 'Check the highlighted fields.',
      }
    default:
      return { ...base, message: error.message }
  }
}
