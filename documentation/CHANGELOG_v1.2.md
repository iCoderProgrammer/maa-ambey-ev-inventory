# Maa Ambey Inventory Management System — Documentation Update v1.2

Updated: 12 October 2026

## Client decisions now reflected
- Initial intended operational brand roster: Lectrix EV, Ceeon India, Ujagar Electric; future brands must be addable. Brand purchasing/sales activation remains dependent on verified supplier, authorisation, catalogue, pricing, warranty and identifier evidence. Relationship between Ujagar Electric and Saffron EV Group is not assumed.
- Client confirms legal business name, GST details and showroom address exist, and GST/income-tax filings are performed. Exact values and accountant-approved invoice settings have not yet been supplied. Tax-return filing remains outside MVP scope.
- Verified catalogue data will be collected per brand; actual catalogue data is still pending.
- Opening-stock import template is required; actual stock data and reconciliation sign-off remain pending.
- Initial roles are Owner, Inventory Manager, Sales Executive, Accountant and Viewer; detailed permission grants and approval thresholds need owner sign-off.
- Delivery checklist categories confirmed: keys, charger, manuals, warranty documents, payment, insurance and registration; mandatory/conditional policy remains open.

## Technology stack alignment
- Single Git repository.
- Frontend: React + TypeScript + Vite.
- Backend: Go + Gin REST API.
- Database: PostgreSQL via pgx and versioned SQL migrations.
- Replaced prior Next.js/NestJS/Prisma-oriented architecture language in Architecture and Implementation Tasks.

## Documents
- 00 Research Brief & Decision Register
- 01 Product Requirements Document
- 02 System Architecture & Technical Design
- 03 Business Rules & Validation Specification
- 04 UX / UI Design Specification
- 05 Implementation Plan & Engineering Task Backlog

## Notes
This update records high-level client decisions. It does not manufacture exact legal details, supplier authorisation, catalogue values, opening stock, granular permissions, invoice/tax rules, or mandatory/conditional delivery policies. These remain data collection or approval tasks.
