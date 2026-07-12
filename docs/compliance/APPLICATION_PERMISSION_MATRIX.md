# AsymmFlow Application Permission Matrix

Last updated: 2026-04-25

This matrix summarizes practical access by production license role. Actual screen visibility is enforced in the frontend through screen permissions and in the backend through `requirePermission(...)`.

## Role Legend

| Role | License prefix | Access style |
|---|---|---|
| Admin | `PH-ADM-*` | Full access |
| Manager | `PH-MGR-*` | Sales, operations, finance, HR/work, approvals |
| Sales | `PH-SLS-*` | Sales pipeline, customer relationships, customer invoices draft/view, limited supplier/PO access |
| Operations | `PH-OPS-*` | Supplier orders, supplier invoices, delivery, order fulfillment |
| Staff | `PH-STF-*` | Dashboard, work, tasks, notifications |

## Screen Access Matrix

| Screen / Area | Admin | Manager | Sales | Operations | Staff |
|---|---:|---:|---:|---:|---:|
| Dashboard | Yes | Yes | Yes | Yes | Yes |
| Opportunities / Sales Hub | Yes | Yes | Yes | No by default | No |
| RFQs | Yes | Yes | Yes | No by default | No |
| Costing | Yes | Yes | Yes | No by default | No |
| Offers | Yes | Yes | Yes | No by default | No |
| Customer Orders | Yes | Yes | Yes | View/update where permitted | No |
| Operations Hub | Yes | Yes | Limited PO create/view | Yes | No |
| Supplier Orders / POs | Yes | Yes | Create/view limited | Yes | No |
| Supplier Invoices | Yes | Yes | No by default | Yes | No |
| Delivery Notes | Yes | Yes | View | Yes | No |
| Finance Hub | Yes | Yes | No by default | No by default | No |
| Customer Invoices | Yes | Yes | View/create draft where permitted | View/create draft where permitted | No |
| Payments Received | Yes | Yes | View | No by default | No |
| Payments Made | Yes | Yes | No | No by default | No |
| Expenses | Yes | Yes | No | No | No |
| Payroll | Yes | Yes | No | No | No |
| Bank Reconciliation | Yes | Yes | No | No | No |
| Relationships / CRM | Yes | Yes | Customers, supplier lookup | Suppliers | No by default |
| Work Hub | Yes | Yes | Yes | Yes | Yes |
| People Hub | Yes | Yes where enabled | No by default | No by default | No |
| Notifications | Yes | Yes | Yes | Yes | Yes |
| Intelligence / Butler | Yes | Yes | Yes | Yes | No by default |
| Settings | Yes | Admin only in production license profile | No | Limited view only where granted | No |
| Deployment | Yes | No | No | No | No |

## Permission Families

| Permission family | Controls |
|---|---|
| `dashboard:*` | Dashboard visibility |
| `offers:*` | Opportunities, RFQ/costing/offer actions |
| `orders:*` | Customer order actions |
| `po:*` | Supplier order / purchase order actions |
| `delivery_notes:*` | Delivery note lifecycle |
| `invoices:*` | Customer invoice actions |
| `payments:*` | Customer payment actions |
| `finance:*` | Finance hub and financial data |
| `expenses:*` | Expense workspace and approvals |
| `payroll:*` | Payroll profiles, runs, approvals |
| `customers:*` | Customer views and edits |
| `suppliers:*` | Supplier views and edits |
| `documents:*` | OCR/document processing |
| `intelligence:*` | Butler chat and reports |
| `tasks:*`, `projects:*`, `notifications:*` | Work collaboration |
| `hr:*` | People hub |
| `settings:*` | Settings, sync, backup, deployment controls |
| `users:*` | User/role administration |
| `*` | Admin wildcard |

## Separation Of Duties

| Control | Enforced behavior |
|---|---|
| Admin wildcard | Only admin/developer roles receive unrestricted access |
| Finance hidden from Sales/Operations | Finance screen requires `finance:view` |
| Supplier invoice approval | Approval function is separate from creation/update |
| Supplier payment recording | Uses dedicated supplier payment permission path |
| Customer payment recording | Amount validation and invoice balance update are transactional |
| Settings/sync controls | Manual sync and sensitive settings require settings permissions |
| Staff role | Restricted to collaboration and visibility by default |

## Audit Notes

1. Frontend hiding is usability control only; backend service functions still enforce permissions.
2. License role permissions are authoritative during production license activation.
3. Some legacy RBAC role definitions in the database may be broader than the current license profile. For production training, use the license profile described here.
4. Admin should review any custom database role before assigning it to live users.

