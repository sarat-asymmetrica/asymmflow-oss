/**
 * SHARED TYPESCRIPT TYPES
 *
 * Proper type definitions to replace unsafe `any` usage across the app.
 * Uses Wails-generated types from models.ts where applicable.
 */

import type { main, crm } from '../../wailsjs/go/models';

// ============================================================
// INTERVAL TYPES
// ============================================================

/**
 * Browser timer interval handle
 * Use this instead of `any` for setInterval/setTimeout
 */
export type TimerHandle = ReturnType<typeof setInterval> | null;

// ============================================================
// CHART DATA TYPES
// ============================================================

/**
 * Revenue chart data point
 * Matches main.CustomerRevenueData from backend
 */
export interface RevenueChartData {
  customer_id: string;
  customer_name: string;
  revenue: number;
  invoice_count: number;
}

// ============================================================
// TASK/FOLLOWUP TYPES
// ============================================================

/**
 * Task priority levels
 */
export type TaskPriority = 'Low' | 'Medium' | 'High' | 'Urgent';

/**
 * Task status
 */
export type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'cancelled';

/**
 * Internal task format for UI components
 * Maps backend FollowUpTask to UI-friendly format
 */
export interface UITask {
  id: string | number;
  title: string;
  subtitle?: string;
  description?: string;
  time?: string;
  priority: TaskPriority;
  status?: TaskStatus;
  urgent?: boolean;
  tag?: string;
  done: boolean;
}

/**
 * Backend FollowUpTask (re-export from Wails models)
 */
export type FollowUpTask = crm.FollowUpTask;

// ============================================================
// DASHBOARD TYPES
// ============================================================

/**
 * Dashboard statistics
 * Matches main.DashboardStats from backend
 */
export interface DashboardStats {
  active_rfqs: number;
  active_orders: number;
  pending_review: number;
  urgent_count: number;
  avg_velocity_days: number;
  win_rate: number;
  total_revenue: number;
  month_growth: number;
  system_health: string;
  runway_months: number;
}

// ============================================================
// REPORT TYPES
// ============================================================

/**
 * Report categories
 */
export type ReportCategory = 'sales' | 'customers' | 'operations' | 'inventory' | 'financial';

/**
 * Sales report data structure
 */
export interface SalesReportData {
  winRate: number;
  conversionRate: number;
  avgDealSize: number;
  pipeline: Array<{
    stage: string;
    value: number;
  }>;
}

/**
 * Customer report data structure
 */
export interface CustomerReportData {
  avgPaymentDays: number;
  collectionEfficiency: number;
  gradeDistribution: Array<{
    grade: string;
    percentage: number;
  }>;
}

/**
 * Operations report data structure
 */
export interface OperationsReportData {
  avgLeadTime: number;
  onTimeDelivery: number;
  pendingShipments: number;
}

/**
 * Inventory report data structure
 */
export interface InventoryReportData {
  totalItems: number;
  totalValue: number;
  lowStockAlerts: number;
}

/**
 * Financial report data structure — trader vocabulary (receivables/payables/
 * collections), matching reports.go's ReportData. No runway/burn/MRR-target:
 * those were either hardcoded fiction or rough revenue*0.3 guesses.
 */
export interface FinancialReportData {
  receivablesOutstanding: number;
  payablesOutstanding: number;
  avgMonthlyRevenue: number;
  collected: number;
  collectionTarget: number;
}

/**
 * Union type for all report data structures
 */
export type ReportData =
  | SalesReportData
  | CustomerReportData
  | OperationsReportData
  | InventoryReportData
  | FinancialReportData;

// ============================================================
// CONFIDENCE METER TYPES
// ============================================================

/**
 * Confidence levels
 */
export type ConfidenceLevel = 'low' | 'medium' | 'high' | 'very_high';

/**
 * Confidence meter data
 */
export interface ConfidenceData {
  score: number; // 0-100
  level: ConfidenceLevel;
  factors?: Array<{
    name: string;
    impact: number;
  }>;
}

// ============================================================
// ECOSYSTEM TYPES
// ============================================================

/**
 * Ecosystem summary data
 */
export interface EcosystemSummary {
  totalEngines: number;
  activeEngines: number;
  totalIntegrations: number;
  healthScore: number;
  lastSync?: string;
}

/**
 * Ecosystem file entry
 */
export interface EcosystemFile {
  id: string;
  name: string;
  path: string;
  type: string;
  size: number;
  modified: string;
}

/**
 * Ecosystem search result
 */
export interface EcosystemSearchResult {
  id: string;
  title: string;
  description: string;
  category: string;
  relevance: number;
}

// ============================================================
// ERROR BOUNDARY TYPES
// ============================================================

/**
 * Error boundary fallback component props
 */
export interface ErrorFallbackProps {
  error: Error;
  errorInfo: { stack?: string } | null;
  reset: () => void;
  reload: () => void;
}

/**
 * Custom error handler function type
 */
export type ErrorHandler = (error: Error) => void;

/**
 * Error boundary fallback component type
 */
export type ErrorFallbackComponent = any; // Svelte component constructor

// ============================================================
// SURVIVAL PANEL TYPES
// ============================================================

/**
 * Survival metrics (re-export from Wails models)
 */
export type SurvivalMetrics = main.SurvivalMetrics;

/**
 * Runway status
 */
export type RunwayStatus = 'safe' | 'warning' | 'critical';

// ============================================================
// ALERT PANEL TYPES
// ============================================================

/**
 * Alert summary for dashboard alert counts
 * (defined locally as backend stub not yet implemented)
 */
export interface AlertSummary {
  Critical: number;
  Warning: number;
  Info: number;
  TotalActive: number;
}

/**
 * Alert severity
 */
export type AlertSeverity = 'info' | 'warning' | 'critical' | 'opportunity';

// ============================================================
// UTILITY TYPES
// ============================================================

/**
 * Generic loading state
 */
export interface LoadingState {
  loading: boolean;
  error: string;
}

/**
 * Generic paginated response
 */
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

/**
 * Generic sort direction
 */
export type SortDirection = 'asc' | 'desc';

/**
 * Generic filter operator
 */
export type FilterOperator = 'eq' | 'ne' | 'gt' | 'gte' | 'lt' | 'lte' | 'contains' | 'startsWith' | 'endsWith';
