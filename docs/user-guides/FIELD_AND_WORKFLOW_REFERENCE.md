# AsymmFlow Field And Workflow Reference

Last updated: 2026-04-25

This reference explains what users should enter in the major fields and what each workflow does. It is organized in the same order as the application navigation.

## Global Controls

| Control | How to use it | Function |
|---|---|---|
| Sidebar navigation | Click visible module labels | Opens screens allowed by your role |
| Sync indicator | Admin/authorized users can click when cloud sync is configured | Runs manual sync with progress |
| Search boxes | Type customer, supplier, order, invoice, project, or description text | Filters the visible table/cards |
| Year filters | Select a calendar or financial year | Limits records to selected year |
| Company selector | Choose `Acme Instrumentation` or `Beacon Controls` in Finance | Changes finance data context |
| Toast messages | Read top-level success/error feedback | Confirms saves, blocks invalid actions, or explains permission denial |

## License Activation

| Field | Required | What to enter | Function |
|---|---:|---|---|
| License key | Yes | Full key such as `PH-SLS-XXXXXX` | Activates the device and loads role permissions |

Role prefixes:

| Prefix | Role |
|---|---|
| `PH-ADM` | Admin |
| `PH-MGR` | Manager |
| `PH-SLS` | Sales |
| `PH-OPS` | Operations |
| `PH-STF` | Staff |

## Dashboard

Dashboard is read-focused. It summarizes current health, tasks, finance visibility, and alerts according to permissions.

| User action | Function |
|---|---|
| Review cards | Understand workload and business state |
| Open related screen | Take action in the source module rather than editing dashboard data |

## Sales Flow: RFQ / Opportunity

Path: `Opportunities -> RFQs`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Customer | Yes | Existing customer from list or exact customer name from enquiry | Links RFQ to customer pipeline and CRM |
| Project | Yes | Short project/enquiry title | Identifies the opportunity |
| Value (BHD) | No | Expected revenue value in BHD | Feeds pipeline value and priority review |
| Priority | No | Low, Medium, High, or Urgent | Helps triage |
| Notes | No | Scope, deadlines, model codes, contact names, exclusions, next action | Preserves sales context |

Filters:

| Filter | Function |
|---|---|
| Year | Shows opportunities for one year |
| Stage/status | Filters by Qualified, Proposal, Quoted, Won, Lost, etc. |
| Sort | Orders the opportunity list |
| Search | Finds customer/project text |

Actions:

| Action | Function |
|---|---|
| New Opportunity | Creates RFQ/opportunity |
| Open opportunity | Shows details and comments/tasks |
| Create Task | Creates a collaborative task from the opportunity |
| Delete | Deletes RFQ; cascade option also removes linked child data where allowed |

## Sales Flow: Costing

Path: `Opportunities -> Costing`

### Costing Selection

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Opportunity | Recommended | Existing opportunity/RFQ | Pulls customer and line-item context |
| Revision | No | Existing revision if present | Lets users compare/update prior costing |

Actions:

| Action | Function |
|---|---|
| New blank costing | Starts costing without opportunity linkage |
| Create new revision | Clones or creates a revision for the selected RFQ |
| Set active revision | Marks selected revision as active |
| Delete costing | Removes selected costing sheet where allowed |

### Costing Header

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Customer | Yes | Customer receiving quote | Controls customer-facing name and credit context |
| Contact Person | No | Buyer/engineer/contact name | Prints or flows into offer context |
| RFQ Reference | No | Customer enquiry/reference | Traceability on offer/PDF |
| Division | Yes | `Acme Instrumentation` or `Beacon Controls` | Company context |
| Date | Yes | Costing/quotation date | Document date |
| Prepared By | Yes | Employee preparing quote | Shows internal owner |
| Quote Type | Yes | Quotation, Budgetary Quote, Estimate, Technical, Commercial | Controls document title/type |
| Folder Number | No | Folder/reference number, e.g. `42-26` | Physical/electronic filing |
| Costing ID | No | Auto or internal ID | Internal reference |
| Payment Terms | Yes | Customer payment terms | Carries to offer |
| Delivery Terms | Yes | Delivery basis | Carries to offer |
| Estimated Delivery | Yes | Delivery lead time | Carries to offer |
| Subject | No | Customer-facing subject | Appears on PDF |
| Opening Body | No | Cover note | Appears before line items |

### Advanced Header

| Field | What to enter | Function |
|---|---|---|
| Order Type | Supply, service, calibration, etc. | Commercial classification |
| Country of Origin | Country/source | Customer and compliance reference |
| COC/COO | Certificate requirement | Certificate handling |
| Test Certificate | Required/not required or details | Quality documentation |
| Installation | Included/excluded/details | Scope clarity |
| Commissioning | Included/excluded/details | Scope clarity |
| Place of Supply | VAT supply location | VAT and invoice context |
| Tax Category | Standard/export/exempt as applicable | VAT handling |
| Customer TRN | Customer tax registration number | VAT/tax document reference |

### Costing Line Items

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Equipment | Yes | Product/equipment name | Customer-facing item name |
| Model | Yes | Supplier model number | Technical identity |
| Quantity | Yes | Positive quantity | Calculates totals |
| Currency | Yes | Supplier currency | Drives conversion |
| Exchange Rate | Yes | BHD conversion rate | Converts cost to BHD |
| FOB | Yes | Supplier unit cost in foreign currency | Base landed cost |
| Freight % | No | Freight percentage | Adds logistics cost |
| Other Costs | No | Internal miscellaneous cost | Adds internal cost |
| Suggested Price | System | Calculated value | Price recommendation |
| User Price | No | Manual sell price | Overrides suggested price |
| Long Code | No | Full supplier ordering code | Technical traceability |
| Detailed Description | No | Specs, approvals, HS codes, detailed scope | Printed in annexure/detail sections |
| Customs % | No | Customs duty percent | Adds cost |
| Handling % | No | Handling percent | Adds cost |
| Finance % | No | Finance percent | Adds cost |
| Margin % | Yes | Target margin | Calculates sell price |

### Costing Summary

| Field | What to enter | Function |
|---|---|---|
| Discount | Customer-facing discount | Reduces customer total |
| Hidden Charges | Internal adjustment only | Reduces profit, not shown to customer |
| VAT Rate | 0 to 100 | Calculates VAT; 0 is allowed only when valid |
| Terms and Conditions | Commercial terms | Prints on quotation |

Actions:

| Action | Function |
|---|---|
| Export Excel | Creates CSV/Excel-style costing export |
| Export PDF | Creates customer-facing PDF |
| Save as Offer | Persists costing as offer with line item costing data |

## Sales Flow: Offers

Path: `Opportunities -> Offers`

Create/edit fields:

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Customer | Yes | Customer name/ID | Links offer to customer |
| Project Name | No | Project/reference title | Sales context |
| Quotation Date | Yes | Offer date | Document date |
| Validity Date | Yes | Date on/after quotation date | Offer expiry |
| Description | Yes | Product/service description | Customer-facing line |
| Equipment | No | Product/equipment name | Detailed item identity |
| Model | No | Model number | Technical identity |
| Currency | Yes | Source currency | Costing context |
| Specification | No | Short technical specification | Customer-facing detail |
| Detailed Description | No | Extended specs/codes/approvals | Detail/annexure content |
| Quantity | Yes | Positive quantity | Calculates line total |
| FOB | No | Supplier cost | Cost visibility |
| Freight | No | Freight cost | Cost visibility |
| Total Cost | System/read-only | Calculated cost | Pricing reference |
| Margin % | Yes | Margin percent | Pricing control |
| Unit Price | Yes | Selling unit price | Customer price |

Status and follow-up fields:

| Field | What to enter | Function |
|---|---|---|
| Customer PO Number | Customer purchase order reference | Required when marking offer won |
| Lost Reason | Reason from dropdown | Required when marking lost |
| Follow-Up Date | Future date | Creates follow-up task |
| Follow-Up Notes | What to check next | Follow-up context |
| Offer header inline fields | Offer number, folder number, RFQ ref, payment/delivery terms, contact details, issued by | Updates quotation metadata |

Actions:

| Action | Function |
|---|---|
| Generate PDF | Exports offer PDF |
| Mark Won | Moves offer into won/order workflow |
| Mark Lost | Closes offer as lost |
| Requote | Starts revised quote path |
| Add Note | Adds offer note |
| Complete Follow-Up | Closes scheduled follow-up |

## Sales Flow: Customer Orders

Path: `Opportunities -> Customer Orders`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Order Number | No | Official or generated order number | Primary order reference |
| Customer PO Number | Yes | Customer PO reference | Legal/commercial traceability |
| Customer | Yes | Customer name | Links order to customer |
| Order Date | Yes | Date order received | Order timeline |
| Required Date | No | Customer required delivery date | Delivery planning |
| Total Value | Yes | Total BHD value | Order value |
| Status | Yes | Current order state | Pipeline tracking |
| Payment Terms | No | Customer payment terms | Invoice context |
| Delivery Terms | No | Delivery agreement | Delivery/invoice context |
| Line Product Code | No | Product/model code | Item identity |
| Line Description | Yes | Item description | Order details |
| Line Quantity | Yes | Positive quantity | Fulfillment quantity |
| Line Unit Price | Yes | BHD unit price | Calculates line value |

Actions:

| Action | Function |
|---|---|
| Create Delivery Note | Creates DN for shipment |
| Create Purchase Order | Creates supplier PO from order |
| Create Invoice | Creates customer invoice |
| Create Proforma | Creates proforma invoice |
| Quick Mark Delivered | Sets delivered state |
| Edit/Delete | Updates/removes order where allowed |

## Operations: Supplier Orders

Path: `Operations -> Supplier Orders`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Supplier | Yes | Supplier from master list | Links PO to supplier |
| Linked Customer Order | No | Customer order supported by PO | Traceability |
| PO Date | Yes | Date PO is raised | Supplier order date |
| Expected Delivery | No | Expected supplier delivery date | Planning |
| Currency | Yes | Supplier currency | Multi-currency support |
| Exchange Rate | Yes | Conversion rate to BHD | Calculates BHD totals |
| Payment Terms | No | Supplier payment terms | AP planning |
| Item Description | Yes | Product/service description | PO line |
| Quantity | Yes | Positive quantity | PO quantity |
| Unit Price Foreign | Yes | Unit price in supplier currency | PO value |

Actions:

| Action | Function |
|---|---|
| Create/Edit PO | Saves supplier order |
| Update Status | Moves lifecycle state |
| Generate PDF | Creates PO PDF |
| Approve | Allows high-value PO to proceed |

## Operations: Supplier Invoices

Path: `Operations -> Supplier Invoices`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Supplier | Yes | Invoice supplier | AP identity |
| Invoice Number | Yes | Supplier invoice number | Supplier document reference |
| Customer Order | No | Internal order link | Traceability |
| Purchase Order ID / PO Ref | Recommended | PO reference | Matching |
| Invoice Date | Yes | Supplier invoice date | AP aging |
| Due Date | Yes | Payment due date | Payables planning |
| Currency | Yes | Invoice currency | Multi-currency support |
| Exchange Rate | Yes | Currency to BHD | BHD reporting |
| Item Description | Yes | Invoice line description | AP line |
| Quantity | Yes | Quantity invoiced | Match quantity |
| Unit Price | Yes | Supplier invoice unit price | Match value |
| Subtotal | Yes | Net amount | Total calculation |
| VAT | No | VAT amount | VAT input tracking |
| Payment Reference | When paid | Transfer/cheque reference | Payment audit |
| Payment Method | When paid | Bank Transfer, Cheque, etc. | Payment audit |

Actions:

| Action | Function |
|---|---|
| Perform Three-Way Match | Checks PO, receiving/GRN, and invoice agreement |
| Approve | Approves for payment |
| Mark Paid | Updates payment status |
| Edit | Corrects supplier invoice fields |

## Operations: Delivery Notes

Path: `Operations -> Delivery Notes`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| DN Number | No | Generated or official DN number | Delivery reference |
| Delivery Date | Yes | Planned/actual delivery date | Delivery timeline |
| Order | Yes | Customer order being delivered | Links DN to order |
| Ship Quantity | Yes | Quantity delivered now | Partial delivery support |
| Serial Selection | If applicable | Available serial numbers | Serial traceability |
| Delivery Address | Yes | Full address | Delivery instruction |
| Contact Person | No | Receiver/contact name | Delivery contact |
| Contact Phone | No | Receiver phone | Delivery contact |
| Driver Name | No | Driver or courier name | Dispatch record |
| Vehicle Number | No | Vehicle/courier reference | Dispatch record |
| Transport Method | Yes | Own Vehicle, Courier, Customer Pickup, etc. | Transport tracking |
| Status | Yes | Prepared, Dispatched, Delivered, Signed, Cancelled | Lifecycle state |
| Partial Delivery | No | Check for staged delivery | Enables sequence fields |
| Delivery Sequence | If partial | Current delivery number | Partial sequence |
| Total Deliveries | If partial | Expected total deliveries | Partial sequence |

Actions:

| Action | Function |
|---|---|
| Dispatch | Marks shipped and updates serial status |
| Confirm Delivery | Marks delivered and starts serial warranty where applicable |
| Generate PDF | Exports DN PDF |
| Delete | Deletes DN where allowed |

## Finance: Customer Invoices

Path: `Finance -> Customer Invoices`

Create invoice fields:

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Order | Yes | Order to invoice | Pulls customer/order data |
| Delivery Note | Optional | DN to invoice | Links invoice to delivery |
| Field visibility checkboxes | No | Select data to show/hide | Controls invoice PDF detail |

Visibility options include equipment, specification, detailed description, FOB, freight, cost, margin, currency, contact, RFQ, country of origin, and delivery weeks. Customer-facing invoices should normally hide internal costs and margins.

Edit fields:

| Field | What to enter | Function |
|---|---|---|
| Status | Draft, Sent, Paid, PartiallyPaid, Overdue, Cancelled, Void, Proforma | Invoice lifecycle |
| Amount | BHD invoice amount | Correct only with authorization |
| Outstanding | Remaining amount | Prefer payments/credit notes instead of manual edits |
| Customer PO Number | Customer PO reference | Document traceability |

Actions:

| Action | Function |
|---|---|
| Generate PDF | Creates invoice PDF |
| Send | Marks invoice as sent |
| Create Credit Note | Creates credit note against invoice |
| Apply Credit Note | Applies credit to invoice |
| Open Bank Recon | Opens reconciliation workflow |

Credit note fields:

| Field | What to enter | Function |
|---|---|---|
| Invoice | Invoice being credited | Links credit note |
| Reason | Return, pricing correction, quantity adjustment, etc. | Audit reason |
| Description | Credited item/service | Credit line |
| Quantity | Credited quantity | Credit amount |
| Rate | BHD rate | Credit amount |

## Finance: Payments Received

Path: `Finance -> Payments Received`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Invoice | Yes | Invoice being paid | Applies payment to invoice |
| Amount | Yes | BHD amount received | Reduces outstanding balance |
| Payment Method | Yes | Cash, Cheque, Bank Transfer, Credit Card, LC, PDC, Other | Payment audit |
| Payment Date | Yes | Receipt date | AR and GL date |
| Reference | Recommended | Transfer/cheque/reference number | Reconciliation |

Payment controls:

| Control | Function |
|---|---|
| Amount cannot exceed outstanding | Prevents over-payment |
| Transaction locking | Prevents double-recording race conditions |
| Idempotency key | Reduces duplicate payment risk |

## Finance: Payments Made

Path: `Finance -> Payments Made`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Supplier Invoice | Yes | Invoice being paid | Applies payment to AP record |
| Amount | Yes | Amount in selected currency | Payment amount |
| Currency | Yes | Payment currency | Multi-currency AP |
| Exchange Rate to BHD | Yes | Currency conversion | BHD reporting |
| Payment Method | Yes | Bank Transfer, Cheque, LC, Cash, Wire Transfer, PDC, Other | Payment audit |
| Date | Yes | Payment date | AP and GL date |
| Reference | Recommended | Cheque/transfer reference | Reconciliation |

## Finance: Expenses

Path: `Finance -> Expenses` and `Finance -> Approvals`

Expense fields:

| Field | Required | What to enter | Function |
|---|---:|---|---|
| New category | No | Category name | Creates expense category |
| New vendor | No | Vendor name | Creates expense vendor |
| Description | Yes | Expense description | Audit description |
| Category | Yes | Expense category | GL/tax classification |
| Vendor | No | Expense vendor | AP/payee context |
| Amount | Yes | Net amount | Expense value |
| VAT Amount | No | VAT amount | Input VAT tracking |
| Cost Center / Memo | No | Cost center or memo | Internal reporting |
| Expense Date | Yes | Expense date | Accounting date |
| Due Date | No | Payment due date | Payables planning |
| Notes | No | Supporting explanation | Audit trail |

Recurring expense fields:

| Field | What to enter | Function |
|---|---|---|
| Recurring expense name | Schedule name | Identifies recurring item |
| Category / Vendor | Classification and payee | Recurring default |
| Amount / VAT amount | Default values | Generated expense values |
| Frequency | Monthly/weekly/etc. | Schedule |
| Next run date | Next generation date | Recurrence trigger |
| Auto submit | Whether generated items submit automatically | Workflow control |

Payment fields:

| Field | What to enter | Function |
|---|---|---|
| Paid at | Payment date | Marks expense paid |
| Payment method | Method used | Payment audit |
| Bank account | Paying bank | Reconciliation |
| Payment reference | Transfer/cheque reference | Reconciliation |

## Finance: Payroll

Path: `Finance -> Payroll`

Compensation profile fields:

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Employee | Yes | Employee profile | Links payroll profile |
| Pay Frequency | Yes | Monthly or configured frequency | Payroll schedule |
| Base Salary | Yes | BHD base salary | Gross pay component |
| Employer Cost | No | Employer-side cost | Cost reporting |
| Housing Allowance | No | BHD allowance | Gross pay component |
| Transport Allowance | No | BHD allowance | Gross pay component |
| Other Allowance | No | BHD allowance | Gross pay component |
| Standard Deduction | No | BHD deduction | Net pay calculation |
| Tax Deduction | No | BHD deduction | Net pay calculation |
| Effective From/To | No | Date range | Profile validity |
| Active | Yes | Checked if current | Payroll generation filter |
| Notes | No | Assumptions/context | HR/payroll audit |

Payroll period fields:

| Field | What to enter | Function |
|---|---|---|
| Period Name | e.g. `Apr 2026 Payroll` | Identifies period |
| Period Start/End | Pay period range | Payroll calculation window |
| Payment Date | Planned pay date | Payout schedule |
| Notes | Period notes | Payroll audit |

Payout fields:

| Field | What to enter | Function |
|---|---|---|
| Paid At | Actual payment date | Marks paid |
| Bank Account | Paying bank | Reconciliation |
| Payment Reference | Bank batch/transfer reference | Audit/reconciliation |

## Finance: Bank Reconciliation

Path: `Finance -> Bank Recon`

Statement selection:

| Field | What to enter | Function |
|---|---|---|
| Bank Account | Account being reconciled | Loads statements |
| Import Account | Bank account for imported statement | Import target |

Statement edit fields:

| Field | What to enter | Function |
|---|---|---|
| Period Start/End | Statement period | Reconciliation period |
| Opening Balance | Bank opening balance | Balance verification |
| Closing Balance | Bank closing balance | Balance verification |
| Status | Imported, InProgress, Reconciled, Verified, Cancelled | Statement lifecycle |
| Notes | OCR summary/import context | Audit context |

Line fields:

| Field | What to enter | Function |
|---|---|---|
| Transaction Date | Bank transaction date | Matching date |
| Description | Bank line description | Matching/search |
| Reference | Bank reference | Matching/search |
| Debit | Outflow amount | Statement amount |
| Credit | Inflow amount | Statement amount |

Matching fields:

| Field | What to enter | Function |
|---|---|---|
| Matching Type | Customer payment, supplier payment, payroll, expense, journal, split | Candidate type |
| Matching Search | Invoice/customer/supplier/reference/amount | Finds candidate |

Bank account fields:

| Field | What to enter | Function |
|---|---|---|
| Division | Acme Instrumentation or Beacon Controls | Company context |
| Bank Name | Bank name | Account identity |
| Account Name | Ledger/display name | Finance display |
| Account Number | Bank account number | Reconciliation |
| IBAN | IBAN | Payment docs |
| Swift/BIC | SWIFT code | Payment docs |
| Currency | Account currency | Cash position and FX |
| Booking Rate | Opening FX booking rate | FX revaluation baseline |

Actions:

| Action | Function |
|---|---|
| Import Statement | Opens import dialog and creates statement/lines |
| Auto Match | Attempts automatic matching |
| Manual Match | Links selected line to selected candidate |
| Unmatch | Removes existing match |
| Add Line | Adds missing bank line |
| Finalize | Locks/finalizes when unmatched count is zero |

## Relationships: Customers

Path: `Relationships -> Customers`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Customer Code | No | Business code or leave blank | Auto-generated if blank |
| Short Code | No | Short acronym | Search/display shorthand |
| CR Number | No | Commercial registration | Identity |
| Company Name | Yes | Legal customer name | Primary customer identity |
| Trading Name | No | DBA/trading name | Alternate identity |
| VAT / TRN | No | Tax registration number | VAT reference |
| Type | No | Customer type | Segmentation |
| Grade | No | A, B, C, D | Payment/risk behavior |
| Status | No | Active/inactive | Record status |
| Industry | No | Customer industry | Segmentation |
| Primary Phone | No | Main phone | Contact |
| Mobile Number | No | Mobile phone | Contact |
| Primary Email | No | Main email | Contact |
| Website | No | Website | Reference |
| Building/Flat, Road/Street, Block, Area, City, Country | No | Address details | Delivery/tax context |
| Payment Terms | No | Customer payment terms | Commercial defaults |
| Credit Limit | No | BHD credit limit | Invoice credit check |

## Relationships: Suppliers

Path: `Relationships -> Suppliers`

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Supplier Name | Yes | Legal supplier name | Primary supplier identity |
| Supplier Code | No | Leave blank for auto-generation | Supplier reference |
| Supplier Type | No | Manufacturer, Distributor, Agent, etc. | Segmentation |
| Contact Person | No | Primary supplier contact | Contact |
| Email | No | Supplier email | Contact |
| Phone | No | Supplier phone | Contact |
| Country | No | Supplier country | Procurement context |
| Tax ID / TRN | No | Supplier VAT/tax ID | Tax reference |
| Lead Time | No | Days | Planning |
| Address | No | Supplier address | Reference |
| Brands Handled | No | Brands, e.g. Rhine Instruments, Oxan Analytics | Supplier capability |

## Work Hub

Path: `Work`

Task fields:

| Field | Required | What to enter | Function |
|---|---:|---|---|
| Task title | Yes | Clear action title | Task identity |
| Priority | Yes | Low, Medium, High, Urgent | Work triage |
| Due date | No | Due date | Deadline |
| Project | No | Related project | Context |
| Assignee | No | Responsible employee | Ownership |
| Description | No | Context, blockers, expected outcome | Task detail |
| Blocker | No | Dependency/owner/next step | Blocked-state context |
| Comment | No | Progress note | Task history |

Project fields:

| Field | What to enter | Function |
|---|---|---|
| Project name | Project title | Project identity |
| Project type | Category | Project grouping |
| Description | Purpose/context | Project detail |
| Member role | Owner, member, reviewer, etc. | Project team role |

## People Hub

Path: `People`

| Field | What to enter | Function |
|---|---|---|
| Full name | Employee name | Directory identity |
| Preferred name | Short name | Display |
| Email | Work email | Contact |
| Phone | Phone | Contact |
| Department | Sales, Operations, Finance, etc. | Org grouping |
| Job title | Job title | Org context |
| Manager | Reporting manager | Org chart |
| Employment status | Active/inactive status | HR state |
| Start date | Employment start | HR context |
| Emergency contact | Emergency contact | HR support |
| Notes | Responsibilities/context | HR notes |
| License key | Available license key | Access linking |

## Intelligence: Butler

Path: `Intelligence`

| Field | What to enter | Function |
|---|---|---|
| Chat input | Plain-language question or command | Calls Butler AI |

Examples:

| Prompt | Use |
|---|---|
| `Show overdue invoices by customer.` | AR review |
| `Create a task for Quinn Hale to confirm delivery.` | Work creation |
| `Summarize supplier invoices pending approval.` | AP review |
| `Draft an offer from the selected opportunity.` | Sales support |

Always review proposed actions before accepting.

## Settings

Path: `Settings`

Settings are admin-controlled. See `ADMIN_USER_GUIDE.md` for field-by-field setup.

