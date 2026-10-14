import {
  Boxes,
  Building2,
  ClipboardCheck,
  ClipboardList,
  FileText,
  History,
  LayoutDashboard,
  type LucideIcon,
  PackageCheck,
  Paperclip,
  Receipt,
  Settings,
  ShoppingCart,
  Tags,
  Truck,
  UserCog,
  Users,
  Wrench,
  Zap,
} from 'lucide-react'

export interface NavItem {
  label: string
  path: string
  icon: LucideIcon
  /** Backlog task IDs (05_Implementation_Tasks) that deliver this module. Empty = implemented. */
  backlog: string[]
  summary: string
  /**
   * Permission key the server must grant before the link is shown (with a list: any one of
   * them). Hiding the link is presentation only: the page and its API check the same permission.
   */
  permission?: string | readonly string[]
}

export interface NavSection {
  label?: string
  items: NavItem[]
}

/** Information architecture from 04_Design_Specification §3. */
export const navigation: NavSection[] = [
  {
    items: [
      {
        label: 'Dashboard',
        path: '/',
        icon: LayoutDashboard,
        backlog: [],
        summary: 'Overview of stock and system status.',
      },
      {
        label: 'Documents',
        path: '/documents',
        icon: Paperclip,
        backlog: [],
        summary: 'Private attachments: upload files and open your uploads.',
        permission: ['attachments.upload', 'attachments.view'],
      },
    ],
  },
  {
    label: 'Inventory',
    items: [
      {
        label: 'Vehicle Units',
        path: '/inventory/vehicles',
        icon: Zap,
        backlog: ['D-04', 'D-05', 'D-08', 'D-09'],
        summary: 'Serialised vehicle registry with chassis, motor and battery identifiers.',
      },
      {
        label: 'Spare Parts & Accessories',
        path: '/inventory/parts',
        icon: Wrench,
        backlog: ['E-03', 'E-04'],
        summary: 'Quantity- or serial-tracked parts and accessories.',
      },
      {
        label: 'Stock Movements',
        path: '/inventory/movements',
        icon: History,
        backlog: ['E-01', 'E-02'],
        summary: 'Append-only ledger of every stock and status change.',
      },
      {
        label: 'Stock Counts & Adjustments',
        path: '/inventory/counts',
        icon: ClipboardCheck,
        backlog: ['E-05', 'E-06'],
        summary: 'Cycle counts, variances and approved adjustments.',
      },
    ],
  },
  {
    label: 'Procurement',
    items: [
      {
        label: 'Suppliers',
        path: '/procurement/suppliers',
        icon: Building2,
        backlog: [],
        summary: 'Supplier master and verified supplier–brand relationships.',
        permission: 'suppliers.view',
      },
      {
        label: 'Purchase Orders',
        path: '/procurement/purchase-orders',
        icon: ShoppingCart,
        backlog: ['D-02'],
        summary: 'Purchase order lifecycle from draft to closed.',
      },
      {
        label: 'Goods Receipts & PDI',
        path: '/procurement/goods-receipts',
        icon: PackageCheck,
        backlog: ['D-03', 'D-06', 'D-07'],
        summary: 'Receiving, serial capture and pre-delivery inspection.',
      },
    ],
  },
  {
    label: 'Sales',
    items: [
      {
        label: 'Customers',
        path: '/sales/customers',
        icon: Users,
        backlog: ['F-01'],
        summary: 'Customer records with minimum necessary personal data.',
      },
      {
        label: 'Quotations',
        path: '/sales/quotations',
        icon: FileText,
        backlog: ['F-02', 'F-03'],
        summary: 'Price-snapshot quotations and discount approvals.',
      },
      {
        label: 'Reservations',
        path: '/sales/reservations',
        icon: ClipboardList,
        backlog: ['F-04', 'F-05'],
        summary: 'Unit reservations with expiry.',
      },
      {
        label: 'Sales Orders & Delivery',
        path: '/sales/orders',
        icon: Truck,
        backlog: ['F-06', 'F-07', 'F-08', 'F-09', 'F-10', 'F-11'],
        summary: 'Sale finalisation, payments, delivery checklist and documents.',
      },
    ],
  },
  {
    label: 'Reports',
    items: [
      {
        label: 'Reports',
        path: '/reports',
        icon: Receipt,
        backlog: ['G-01', 'G-02', 'G-03', 'G-04', 'G-05'],
        summary: 'Brand-aware inventory, procurement and sales reports with exports.',
      },
    ],
  },
  {
    label: 'Admin',
    items: [
      {
        label: 'Catalogue & Brands',
        path: '/admin/catalogue',
        icon: Tags,
        backlog: [],
        summary: 'Brands, models, variants and colours with verification and authorisation status.',
        permission: 'catalogue.view',
      },
      {
        label: 'Users & Roles',
        path: '/admin/users',
        icon: UserCog,
        backlog: [],
        summary: 'Roles, permission grants and their approval status.',
        permission: 'access.roles.view',
      },
      {
        label: 'Store Settings',
        path: '/admin/settings',
        icon: Settings,
        backlog: ['F-10'],
        summary: 'Legal entity, invoice series and document templates.',
      },
      {
        label: 'Audit Log',
        path: '/admin/audit',
        icon: Boxes,
        backlog: ['B-08'],
        summary: 'Append-only audit events for sensitive actions.',
      },
    ],
  },
]

export const plannedModules = navigation.flatMap((section) =>
  section.items
    .filter((item) => item.backlog.length > 0)
    .map((item) => ({ section: section.label, item })),
)
