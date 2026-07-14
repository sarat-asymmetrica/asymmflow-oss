/**
 * Central Type Definitions for the Sovereign UI
 *
 * This file consolidates ALL business domain types used across components.
 * Extracted from scattered definitions in OpportunityCard, OpportunityDetail,
 * DashboardScreen, CustomersScreen, OrderDetail, SpaceTimeCanvas, etc.
 */

// ============================================================
// CORE BUSINESS ENTITIES
// ============================================================

/** Opportunity/RFQ entity */
export interface Opportunity {
  id?: string | number;
  status?: 'New' | 'Quoted' | 'Won' | 'Lost';
  title?: string;
  customer?: string;
  value?: number;
  updatedAt?: string | number | Date | null;
  paymentGrade?: PaymentGrade;
  customer_payment_grade?: PaymentGrade;
  notes?: string | null;
  competitor?: string | null;
}

/** Customer entity */
export interface Customer {
  id: string | number;
  name: string;
  type?: CustomerType;
  code?: string;
  paymentGrade?: PaymentGrade;
  payment?: PaymentGrade; // Alternate field name
  contacts?: Contact[];
}

/** Contact within a customer */
export interface Contact {
  name?: string;
  role?: string;
  email?: string;
  phone?: string;
}

/** Customer types based on SSOT */
export type CustomerType = 'EC' | 'CO' | 'EP' | 'IR' | 'NR' | 'PB' | 'SI' | 'SP' | 'PH';

/** Payment grades for risk classification */
export type PaymentGrade = 'A' | 'B' | 'C' | 'D';

/** Payment grade configuration */
export interface PaymentGradeConfig {
  color: string;
  label: string;
  hint: string;
  risk: 'low' | 'medium' | 'high' | 'critical';
}

// ============================================================
// HISTORY & AUDIT TRAIL
// ============================================================

/** History entry for opportunity status changes */
export interface HistoryEntry {
  status?: 'New' | 'Quoted' | 'Won' | 'Lost';
  note?: string;
  createdAt?: string | number | Date | null;
}

/** Quick capture (voice/text notes) */
export interface Capture {
  id?: string | number;
  text?: string;
  type?: string;
  createdAt?: string | number | Date | null;
}

// ============================================================
// ORDER MANAGEMENT
// ============================================================

/** Order detail entity */
export interface OrderDetail {
  orderId: string;
  customer: string;
  orderDate: string;
  status: OrderStatus;
  totalAmount: number;
  lineItems: LineItem[];
  shipments: Shipment[];
  history: OrderHistoryEntry[];
}

export type OrderStatus = 'Pending' | 'Processing' | 'Shipped' | 'Delivered' | 'Cancelled';

export interface LineItem {
  id: string;
  description: string;
  quantity: number;
  unitPrice: number;
  total: number;
}

export interface Shipment {
  id: string;
  trackingNumber: string;
  carrier: string;
  shipDate: string;
  estimatedDelivery: string;
  status: string;
}

export interface OrderHistoryEntry {
  timestamp: string;
  event: string;
  note?: string;
}

// ============================================================
// VISUAL CANVAS & GEOMETRY
// ============================================================

/** Regime for three-regime dynamics visualization */
export interface Regime {
  primary_color: string;
  secondary_color: string;
  geometry?: RegimeGeometry | null;
}

export interface RegimeGeometry {
  type?: string; // e.g., 'FluidPlane', 'Grid', 'Cyberpunk'
}

// ============================================================
// DASHBOARD METRICS
// ============================================================

/** Dashboard statistics */
export interface DashboardMetrics {
  cashBalance: number;
  monthlyBurn: number;
  runwayMonths: number;
  pipeline: Opportunity[];
  winProbability: number;
  tasks: Capture[];
  activeTasks: number;
}

/** Inbox statistics from Runtime API */
export interface InboxStats {
  ready: number;
  needs_review: number;
  processed: number;
  total_documents: number;
  by_type?: { [key: string]: number };
  success?: boolean; // API response flag
}

/** Pricing analytics from Runtime API */
export interface PricingAnalytics {
  success?: boolean; // API response flag
  overallWinRate: number;
  averageMargin: number;
  totalRevenue: number;
  totalQuotes: number;
  totalCustomers?: number;
  topCustomers?: CustomerPricingAnalytics[];
  optimalMarginRange?: {
    min: number;
    max: number;
  };
  averageWinningMargin?: number;
}

/** Per-customer pricing analytics */
export interface CustomerPricingAnalytics {
  customer: string;
  totalRevenue: number;
  winRate: number;
  regime: PricingRegime;
  totalQuotes?: number;
  wonQuotes?: number;
  lostQuotes?: number;
  averageMargin?: number;
  regimeDescription?: string;
}

export type PricingRegime = 'PriceSensitive' | 'ValueBalanced' | 'Premium' | 'Unknown';

// ============================================================
// AI SIMULATION
// ============================================================

/** Margin simulation result from AI */
export interface MarginSimulation {
  currentWinRate: number;
  estimatedWinRate: number;
  confidence: number;
  recommendedAction: string;
  warning?: string;
}

// ============================================================
// SETUP WIZARD & ONBOARDING
// ============================================================

/** Message in conversational setup */
export interface SetupMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp?: string | number | Date;
}

/** Setup wizard state */
export interface SetupState {
  step: number;
  completed: boolean;
  selectedFolders?: string[];
  companyName?: string;
  dataSource?: 'local' | 'cloud' | 'hybrid';
}

/** Folder validation result */
export interface FolderValidation {
  path: string;
  valid: boolean;
  reason?: string;
  fileCount?: number;
}

// ============================================================
// ARCHAEOLOGY (DOCUMENT SCANNING)
// ============================================================

/** Archaeology scan message */
export interface ArchaeologyMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string | number | Date;
}

/** Archaeology scan state */
export interface ScanState {
  scanning: boolean;
  progress: number;
  currentFile?: string;
  totalFiles?: number;
  errors?: string[];
}

// ============================================================
// REPORTS
// ============================================================

export type ReportCategory = 'sales' | 'customers' | 'operations' | 'inventory' | 'financial';

export interface Report {
  id: string;
  name: string;
  category: ReportCategory;
  description?: string;
  lastRun?: string | number | Date;
}

// ============================================================
// UTILITIES & META
// ============================================================

/** Generic API response wrapper */
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

/** Pagination metadata */
export interface Pagination {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

/** Filter state for list views */
export interface FilterState {
  search?: string;
  status?: string;
  type?: string;
  customer?: string;
  page: number;
  pageSize: number;
}
