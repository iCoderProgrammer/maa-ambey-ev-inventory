-- Eligible-PO fixture (placeholder data, throwaway DB only), as in purchasing api_integration_test.go.
DO $$
DECLARE c uuid; ap uuid; b uuid; m uuid; v uuid; s uuid; att uuid := gen_random_uuid(); pl uuid;
BEGIN
  SELECT id INTO c FROM users WHERE username = 'e2e.catalogue';
  SELECT id INTO ap FROM users WHERE username = 'e2e.catviewer';
  INSERT INTO brands (name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name, authorisation_reference, verified_by, verified_at)
    VALUES ('E2E Fixture Brand', 'e2e-fixture-brand', 'ACTIVE', 'AUTHORISED', 'Example Manufacturer', 'TEST-AUTH', ap, now()) RETURNING id INTO b;
  INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
    VALUES (b, 'Fixture Model', 'FM', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, ap, now()) RETURNING id INTO m;
  INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
    VALUES (b, m, 'Fixture Variant', 'FV', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/v', current_date, ap, now()) RETURNING id INTO v;
  INSERT INTO suppliers (legal_name, status, created_by) VALUES ('E2E Fixture Supplier', 'PENDING_VERIFICATION', c) RETURNING id INTO s;
  INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes, sha256, original_name, uploaded_by)
    VALUES (att, to_char(now(), 'YYYY/MM/') || gen_random_uuid()::text, 'local', 'application/pdf', 10, sha256(att::text::bytea), 'evidence.pdf', c);
  INSERT INTO supplier_documents (supplier_id, attachment_id, document_type, added_by) VALUES (s, att, 'AUTHORISATION', c);
  UPDATE suppliers SET status = 'VERIFIED', verification_reference = 'TEST', verification_document_id = att, verified_by = ap, verified_at = now() WHERE id = s;
  INSERT INTO supplier_brand_links (supplier_id, brand_id, status, authorisation_reference, evidence_document_id, effective_from, verified_by, verified_at, created_by)
    VALUES (s, b, 'VERIFIED', 'TEST-LINK', att, current_date - 1, ap, now(), c);
  INSERT INTO price_lists (brand_id, name, created_by) VALUES (b, 'E2E Fixture Costs', c) RETURNING id INTO pl;
  INSERT INTO price_items (price_list_id, brand_id, model_id, variant_id, price_type, amount, effective_from, status, created_by, approved_by, approved_at)
    VALUES (pl, b, m, v, 'COST', 900.00, now() - interval '1 hour', 'APPROVED', c, ap, now());
END $$;
