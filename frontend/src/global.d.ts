// Global type declarations for the Sovereign UI

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          // Survival Intelligence API
          GetSurvivalMetrics?: () => Promise<SurvivalMetrics>;
          GetAlertSummary?: () => Promise<AlertSummary>;
          AcknowledgeAlert?: (alertId: number) => Promise<void>;
          DismissAlert?: (alertId: number) => Promise<void>;
          ComputeAlerts?: () => Promise<void>;
          // ... other App methods
          [key: string]: any;
        };
      };
    };
    runtime?: {
      WindowOpenURL?: (url: string) => void;
      [key: string]: any;
    };
  }

  // Survival Intelligence Types
  interface SurvivalMetrics {
    cash_balance: number;
    monthly_burn: number;
    days_of_runway: number;
    runway_status: string;
    week_collections_target: number;
    week_collections_actual: number;
    collection_efficiency: number;
    overdue_by_grade: OverdueGradeBreakdown[];
    win_rate_by_discount: WinRateBreakdown[];
    last_updated: string;
  }

  interface OverdueGradeBreakdown {
    grade: string;
    overdue_30: number;
    overdue_60: number;
    overdue_120: number;
    total_overdue: number;
    invoice_count: number;
  }

  interface WinRateBreakdown {
    discount_band: string;
    total_quoted: number;
    won: number;
    lost: number;
    win_rate: number;
    avg_margin: number;
  }

  interface AlertSummary {
    active_critical: number;
    active_warning: number;
    active_info: number;
    total_active: number;
    top_alerts: Alert[];
  }

  interface Alert {
    id: number;
    alert_type: string;
    severity: string;
    title: string;
    message: string;
    customer_id?: number;
    customer_name?: string;
    opportunity_id?: number;
    invoice_id?: number;
    current_value?: number;
    threshold_value?: number;
    is_active: boolean;
    is_acknowledged: boolean;
    resolved_at?: string;
    resolved_by?: string;
    created_at: string;
    updated_at: string;
  }
}

export {};
