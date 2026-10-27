// Package isolation holds cross-module tests for brand isolation (C-06): the
// catalogue and pricing APIs together, over a real PostgreSQL database and
// real sessions. It has no production code.
//
// What is covered: cross-brand mismatches fail (writes, list filters, price
// lookup, database foreign keys); brand filters return only that brand's rows;
// readiness of one brand is never satisfied by another brand's data; an
// arbitrary new brand works end to end with data only; no brand name appears in
// application code. Not covered because the features do not exist yet:
// supplier mappings (D-01), warranty terms (RB-2), identifier rules (D-04),
// exports (G-05) and per-user brand scope (RB-5) — see ADR-014.
package isolation
