/**
 * Archaeology Report Types
 * These mirror the Go structs in artifacts.go (Wave 1)
 */

export interface WorkspaceOverview {
  total_files: number;
  total_folders: number;
  total_size_bytes: number;
  total_size_mb: number;
  oldest_file: string; // ISO timestamp
  newest_file: string; // ISO timestamp
  formats_detected: string[]; // e.g., [".pdf", ".xlsx"]
}

export interface ClusterInfo {
  cluster_type: 'folder' | 'name_pattern' | 'revision';
  description: string;
  file_paths: string[];
  confidence: number; // 0.0 - 1.0
}

export interface QualitySummary {
  high_confidence: number;   // clarity >= 0.8
  medium_confidence: number; // 0.5 <= clarity < 0.8
  low_confidence: number;    // 0.2 <= clarity < 0.5
  unreadable: number;        // clarity < 0.2
}

export interface LanguageFormatSection {
  english: number;
  arabic: number;
  mixed: number;
  scanned_pdfs: number;
  native_pdfs: number;
  excel_files: number;
  word_files: number;
  other_formats: number;
}

export interface UncertaintyItem {
  type: 'duplicates' | 'clarification_needed' | 'conflicting_totals';
  description: string;
  affected_docs: string[];
}

export interface ArchaeologyReportData {
  generated_at: string; // ISO timestamp
  workspace_overview: WorkspaceOverview;
  detected_clusters: ClusterInfo[];
  quality_summary: QualitySummary;
  languages_formats: LanguageFormatSection;
  uncertainties: UncertaintyItem[];
  markdown_content: string;
}

// Additional types for Evidence Extracts (Wave 1)
export interface NumericField {
  label: string;      // e.g., "Total Amount", "Invoice Number"
  value: string;      // Raw extracted value
  confidence: number; // 0.0 - 1.0
  type: 'amount' | 'date' | 'number' | 'id';
}

export interface QualityMetrics {
  clarity: number;              // Text clarity (0.0 - 1.0)
  layout: number;               // Layout preservation (0.0 - 1.0)
  language_confidence: number;  // Language detection confidence
  scan_noise: number;           // Scan artifacts/noise level
  table_integrity: number;      // Table structure preservation
}

export interface ProvenanceInfo {
  engine: string;  // "ACE", "Tesseract", etc.
  backend: string; // "GPU", "CPU", "AIMLAPI"
  time_ms: number; // Processing time in milliseconds
}

export interface EvidenceExtract {
  source_path: string;
  pages: number;
  languages: string[];
  layout_tokens: boolean;
  extracted_text: string;
  numeric_fields: NumericField[];
  quality: QualityMetrics;
  provenance: ProvenanceInfo;
}

// Workspace Index types (Artifact A)
export interface FileNode {
  path: string;
  type: 'file' | 'folder';
  extension?: string;  // e.g., ".pdf", ".xlsx"
  size_bytes: number;
  modified_at: string; // ISO timestamp
  hash?: string;       // SHA256 for files only
  parent: string;
  children?: FileNode[];
}

export interface WorkspaceIndex {
  root_path: string;
  scan_time: string; // ISO timestamp
  total_files: number;
  total_folders: number;
  total_bytes: number;
  root: FileNode;
  extensions: Record<string, number>; // extension -> count
}
