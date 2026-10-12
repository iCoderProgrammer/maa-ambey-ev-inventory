# UI / UX Guidelines

Mandatory for every frontend change (source: the user's "Master UI/UX Engineering Prompt v2",
2026-10-12). The approved Design Specification v1.2 (`04_Design_Specification`) remains the authority
for screens and tokens; this file records how we build them. Task workflow: `TASK_EXECUTION_WORKFLOW.md`.

## Product feel

Internal operational console for owners, inventory managers, sales executives, accountants and viewers.
Modern, professional, EV/engineering-inspired, calm. Clarity, speed, accuracy, consistency and
accessibility over decoration: no heavy gradients, neon, glassmorphism, oversized charts or marketing
visuals. Use cards only where they group information.

## Components and dependencies

| Need | Use |
| --- | --- |
| Class names | `cn()` in `ui/src/lib/utils/cn.ts` (clsx + tailwind-merge, knows `rounded-card`/`rounded-control`) |
| Buttons / button-styled links | `Button`, `buttonVariants()` (`primary`, `secondary`, `ghost`, `danger`; `sm`, `md`, `icon`) |
| Modal dialogs, drawers | `ui/Dialog` / Radix `Dialog` from `radix-ui` |
| Popovers, tooltips | Radix `Popover`; `ui/Tooltip` (supplementary only; triggers keep their own accessible name) |
| Status | `Badge` (always with text) |
| Containers | `Card` (title, description, actions) |
| Forms | React Hook Form + Zod with `Field`, `Input`, `Select` |
| Tables | `ui/Table` (sticky header, horizontal scroll); extend it before adding a table library |
| States | `LoadingState`, `EmptyState` (icon, explanation, permitted action), `ErrorState` (safe message, request ID, retry) |
| Icons | `lucide-react` |

Installed for UI (2026-10-12): `radix-ui@1.6.7`, `clsx@2.1.1`, `tailwind-merge@3.7.0`.

Before adding a dependency: inspect `ui/package.json` and the lockfile, reuse what exists, pick one
maintained, compatible package per purpose, pin a release at least two weeks old, install only what
the task needs, run lint/typecheck/tests/build and record it in the task log. Never fake an
install. No competing component libraries, no animation library for effects CSS can do, no
smooth-scroll library, no font package without need.

Radix `asChild` (Slot) merges `className` as strings: pass child components a string className,
never a function (see `SidebarLink`).

## Tokens, type and spacing

Use the `@theme` tokens in `ui/src/styles/index.css` (`primary`, `accent`, `ink`, `canvas`, `surface`,
`surface-muted`, `border`, `muted`, `success|warning|danger|neutral|info` + `-subtle`), never raw hex in
components. Primary blue = main actions/selection/links; teal = restrained accent; green success,
amber pending/warning, red error/destructive, grey neutral/archived — always paired with text or icon.
Page title 24–32 px, section 18–20 px, body/labels 14–16 px, dense metadata 12–13 px. 4 px spacing
scale; radii `rounded-card` (12 px) and `rounded-control` (8 px).

## Layout

One shell: sidebar (collapsible on desktop, Radix drawer below `lg`), sticky header (API status,
theme, account), page header (breadcrumbs, title, optional status badge, description, primary action),
then filters, content, pagination. Planned modules are labelled "Planned" everywhere (visual chip,
screen-reader text, collapsed tooltip). No visible control may be fake or silently inert; Logout only
once authentication exists. No horizontal page overflow; wide tables scroll inside their container.

## Motion and scrolling

Native scrolling. `scroll-behavior: smooth` only for in-page anchors and only without reduced motion;
route changes jump to the top instantly. Transitions 150–250 ms via the `animate-fade-*`,
`animate-pop-*`, `animate-drawer-*` tokens and Radix `data-state`. `prefers-reduced-motion: reduce`
cuts animations/transitions to 1 ms. No bounce, parallax, endless or large-content animation;
motion is never the only status signal.

## Tables, forms, confirmations

Tables: aligned columns, readable headers, status badges, record count, server pagination and
search/filter where supported, loading/empty/no-results/error states; never hide identifiers or
statuses on narrow screens. Forms: grouped fields, labels and required markers, helper text,
field-level errors, correct input types, keyboard flow, descriptive submit/cancel, duplicate-submit
protection, unsaved-change warning where justified; dependent options (brand → model → variant) come
from the API. High-impact actions use a confirmation dialog stating the effect, with a verb label
("Post receipt", "Finalise sale"), and only after server-side rules allow them.

## Data honesty and multi-brand

Render only real API data. Never invent prices, specs, warranties, supplier authorisation, stock,
KPIs or totals; label unverified/pending data. Brands come from the API (initial roster Lectrix EV,
Ceeon India, Ujagar Electric; more can be added). Brand selector where relevant; "All brands" only when
permissions and API allow; keep catalogue status separate from supplier/dealership authorisation.

## Responsive and accessibility

Desktop-first; usable on tablet (touch targets ≥ 24 px, ideally 36–40) and mobile browsers; 200 % zoom
without overlap. WCAG 2.2 AA where practical: semantic headings, labelled controls, keyboard
operation, visible focus (`:focus-visible` outline), contrast, accessible dialogs/menus, announced
validation, no colour-only meaning, reduced-motion support.

## States

Loading only while work is pending (no indefinite spinners; explain offline waits). Empty states
explain and offer a permitted next action. Errors are safe, show the request ID and retry when safe.
Success only after the server confirms. Network loss: warn, never pretend stock/financial actions
succeeded. Permission denied: explain without leaking details.

## UI checklist (each task)

Consistent tokens/spacing/type · existing components reused, any new dependency justified · natural
scrolling, subtle motion, reduced motion respected · tables/forms efficient · loading/empty/error/
success/permission states · desktop/tablet/mobile · keyboard, focus, labels · no fake controls or
false success · lint, typecheck, tests, build run.
