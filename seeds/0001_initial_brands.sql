-- Client-confirmed initial brands (docs/REQUIREMENTS_BASELINE.md §1).
-- Idempotent: safe to run any number of times.
--
-- Brands start PENDING_VERIFICATION / UNVERIFIED. Per BR-002 a brand becomes
-- ACTIVE for purchase/sales only after supplier identity, authorisation
-- reference, approver and verified date are recorded through the audited
-- catalogue workflow (backlog C-02). No evidence is fabricated here.
--
-- Only brand identity is seeded: no models, variants, prices, warranties,
-- suppliers, serials or stock.
INSERT INTO brands (name, slug)
VALUES
    ('Lectrix EV', 'lectrix-ev'),
    ('Ceeon India', 'ceeon-india'),
    ('Ujagar Electric', 'ujagar-electric')
ON CONFLICT DO NOTHING;
