-- E-06: ledger reconciliation (ADR-028; LED-001…LED-004, INV-009).
--
-- Reconciliation is read-only: it compares vehicle units and parts balances with
-- the stock movement ledger inside one read-only, repeatable-read transaction and
-- returns what disagrees. It stores nothing and changes nothing — no run table,
-- no correction, no movement. The only schema change is the permission that
-- lets a role open the report (no grant: the permission matrix awaits owner
-- sign-off, RB-5 / A-05).

INSERT INTO permissions (key, description, sensitive) VALUES
    ('stock.reconciliation.view', 'Run and view the read-only stock ledger reconciliation report (vehicle units and parts balances at the caller''s locations)', false);
