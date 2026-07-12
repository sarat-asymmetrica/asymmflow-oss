/**
 * Shared Component Types
 * Type definitions for enterprise UI components
 */

export interface Tab {
  id: string;
  label: string;
  count?: number;
  disabled?: boolean;
}

export interface DropdownOption {
  value: string;
  label: string;
  icon?: string;
  disabled?: boolean;
}
