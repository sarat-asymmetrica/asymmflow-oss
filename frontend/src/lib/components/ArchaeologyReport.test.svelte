<script lang="ts">
  import ArchaeologyReport from './ArchaeologyReport.svelte';
  import type { ArchaeologyReportData } from '../types/archaeology';

  // Mock data for testing
  const mockReport: ArchaeologyReportData = {
    generated_at: "2024-12-17T04:27:18Z",
    workspace_overview: {
      total_files: 42,
      total_folders: 8,
      total_size_bytes: 15728640,
      total_size_mb: 15.0,
      oldest_file: "2023-01-15T10:30:00Z",
      newest_file: "2024-12-15T14:22:33Z",
      formats_detected: [".pdf", ".xlsx", ".docx", ".jpg"]
    },
    detected_clusters: [
      {
        cluster_type: "folder",
        description: "Folders with similar names: 'invoices'",
        file_paths: [
          "/workspace/invoices_2023",
          "/workspace/invoices_2024"
        ],
        confidence: 0.6
      },
      {
        cluster_type: "name_pattern",
        description: "Documents sharing number pattern: '12345'",
        file_paths: [
          "/workspace/invoice_12345.pdf",
          "/workspace/receipt_12345.pdf"
        ],
        confidence: 0.5
      }
    ],
    quality_summary: {
      high_confidence: 28,
      medium_confidence: 10,
      low_confidence: 3,
      unreadable: 1
    },
    languages_formats: {
      english: 25,
      arabic: 12,
      mixed: 5,
      scanned_pdfs: 18,
      native_pdfs: 10,
      excel_files: 8,
      word_files: 4,
      other_formats: 2
    },
    uncertainties: [
      {
        type: "duplicates",
        description: "Multiple documents with same amount: 1500.00",
        affected_docs: [
          "/workspace/invoice_001.pdf",
          "/workspace/invoice_002.pdf"
        ]
      },
      {
        type: "clarification_needed",
        description: "Documents with very low quality - manual review recommended",
        affected_docs: [
          "/workspace/scanned_old_doc.pdf"
        ]
      }
    ],
    markdown_content: "# Archaeology Report..."
  };
</script>

<div style="background: #f5f5f5; padding: 2rem;">
  <h1>ArchaeologyReport Component Test</h1>

  <h2>With Data</h2>
  <ArchaeologyReport report={mockReport} />

  <h2 style="margin-top: 3rem;">Without Data</h2>
  <ArchaeologyReport report={null} />
</div>
