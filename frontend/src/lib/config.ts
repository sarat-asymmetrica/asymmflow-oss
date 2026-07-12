/**
 * Application Configuration
 * Centralized config for runtime URLs and environment variables
 *
 * This file provides a single source of truth for configuration,
 * eliminating hardcoded URLs throughout the codebase.
 */

// Runtime API URL - can be overridden via environment variable
export const RUNTIME_URL = import.meta.env.VITE_RUNTIME_URL || 'http://localhost:5263';

// API Endpoints
export const API_ENDPOINTS = {
  // Quotation
  QUOTATION_FROM_EXCEL: `${RUNTIME_URL}/api/quotation/from-excel`,
  QUOTATION_TO_PDF: `${RUNTIME_URL}/api/quotation/to-pdf`,

  // Pricing
  PRICING_RECOMMEND: `${RUNTIME_URL}/api/pricing/recommend`,

  // Ecosystem
  ECOSYSTEM_SUMMARY: `${RUNTIME_URL}/api/ecosystem/summary`,
  ECOSYSTEM_SCAN: `${RUNTIME_URL}/api/ecosystem/scan`,
  ECOSYSTEM_SEARCH: `${RUNTIME_URL}/api/ecosystem/search`,

  // Nodes
  NODES_BY_TAG: (tag: string) => `${RUNTIME_URL}/api/nodes/tag/${tag}`,

  // Edge
  EDGE_CURRENT: `${RUNTIME_URL}/api/edge/current`,
  EDGE_EXTRACT: `${RUNTIME_URL}/api/edge/extract`,
} as const;

// Environment info
export const IS_DEV = import.meta.env.DEV;
export const IS_PROD = import.meta.env.PROD;

export default {
  RUNTIME_URL,
  API_ENDPOINTS,
  IS_DEV,
  IS_PROD,
};
