/**
 * Vehicle identifier rules shared by every screen that captures identifiers
 * (D-05, ADR-019). The server decides; these only keep the screen consistent
 * with it.
 */

/**
 * The comparison form of a chassis/frame number: surrounding spaces trimmed and
 * letters upper-cased, nothing else — the server's `upper(btrim())`. Inner
 * spaces and separators are kept ("CH 0001", "CH/0001" and "CH0001" differ)
 * until a separator rule is approved (OQ-26). Identifiers are printable ASCII
 * only, so JavaScript and PostgreSQL agree on both steps.
 */
export function normaliseChassis(value: string): string {
  return value.trim().toUpperCase()
}

/** Error codes the vehicle identifier rules add (API_CONVENTIONS "Duplicate vehicle identifiers"). */
export const VehicleErrorCode = {
  /** 409: an identifier is already registered to another vehicle unit. */
  DuplicateIdentifier: 'DUPLICATE_IDENTIFIER',
} as const

/**
 * The identifier fields a DUPLICATE_IDENTIFIER detail may name. Only the
 * chassis/frame number is unique so far; motor, battery and charger numbers
 * have no approved uniqueness policy (OQ-4, OQ-30), so the screen never treats
 * them as unique itself and only shows what the server reports.
 */
export const identifierFieldNames = [
  'chassisNumber',
  'motorSerial',
  'batterySerial',
  'chargerSerial',
] as const
export type IdentifierFieldName = (typeof identifierFieldNames)[number]

/** Whether a form path ends in one of the identifier fields. */
export function isIdentifierPath(path: string): boolean {
  const last = path.slice(path.lastIndexOf('.') + 1)
  return (identifierFieldNames as readonly string[]).includes(last)
}
