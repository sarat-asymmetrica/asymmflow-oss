<script lang="ts">

import { devLog } from "$lib/utils/devLog";
import { validateFile } from "$lib/utils/fileValidation";
/**
   * Quick Capture Component - Drag & Drop RFQ Upload
   *
   * Wabi-Sabi Design Philosophy:
   * - φ-based spacing (8, 13, 21, 34, 55px)
   * - Rice paper aesthetic with ink accents
   * - Smooth drag & drop interaction
   */
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { toast } from '$lib/stores/toasts';
  // REAL OCR BACKEND INTEGRATION - Unified Pipeline!
  import { QuickCaptureDocument } from '../../../wailsjs/go/main/App';
import { QuickCaptureDocumentFromBase64 } from '../../../wailsjs/go/main/DocumentsService';
  import { OnFileDrop, OnFileDropOff } from '../../../wailsjs/runtime/runtime';
  import WabiSpinner from './ui/WabiSpinner.svelte';

  const dispatch = createEventDispatcher();

  let uploading = $state(false);
  let dragging = $state(false);
  let progress = $state(0);
  let selectedFile = $state(null);
  let ocrResult = $state(null); // Store OCR results for display

  // File input reference
  let fileInput;

  // Setup Wails file drop handler
  onMount(() => {
    // Register file drop handler - Wails provides REAL file paths!
    OnFileDrop((x: number, y: number, paths: string[]) => {
      if (paths.length > 0 && !uploading) {
        processFileFromPath(paths[0]);
      }
    }, false);
  });

  // Cleanup on unmount
  onDestroy(() => {
    OnFileDropOff();
  });

  // Handle drag over
  function handleDragOver(event) {
    event.preventDefault();
    dragging = true;
  }

  // Handle drag leave
  function handleDragLeave(event) {
    event.preventDefault();
    dragging = false;
  }

  // Handle drop - Wails OnFileDrop handles File Explorer, this handles Outlook/browser
  async function handleDrop(event) {
    event.preventDefault();
    dragging = false;

    // Give Wails OnFileDrop 100ms to fire first (for File Explorer drops)
    // If it doesn't, fall back to browser DataTransfer (for Outlook drops)
    setTimeout(async () => {
      if (uploading) return; // Wails handler already processing

      const items = event.dataTransfer?.items;
      if (items && items.length > 0) {
        for (let i = 0; i < items.length; i++) {
          const item = items[i];
          if (item.kind === 'file') {
            const file = item.getAsFile();
            if (file) {
              await processFileFromDataTransfer(file);
              break; // Process first file only
            }
          }
        }
      }
    }, 100);
  }

  // Process file from real file system path (provided by Wails)
  async function processFileFromPath(filePath: string) {
    if (!filePath || uploading) return;

    devLog.info('Processing file:', filePath);

    // Extract filename for display
    const fileName = filePath.split(/[\\/]/).pop() || 'document';

    // Validate file type
    const validExts = ['.pdf', '.xlsx', '.xls', '.docx', '.rtf', '.msg', '.eml', '.png', '.jpg', '.jpeg', '.bmp', '.tiff', '.tif', '.webp'];
    const fileExt = '.' + fileName.split('.').pop()?.toLowerCase();

    if (!validExts.includes(fileExt)) {
      toast.danger(`Unsupported file type: ${fileExt}. Please upload PDF, Excel, Word, RTF, Outlook, Email, or image files.`);
      return;
    }

    uploading = true;
    selectedFile = fileName;
    progress = 0;
    ocrResult = null;

    try {
      // Animate progress while processing
      const progressInterval = setInterval(() => {
        progress = Math.min(progress + 5, 90);
      }, 150);

      // REAL OCR BACKEND CALL!
      const result = await QuickCaptureDocument(filePath);

      clearInterval(progressInterval);
      progress = 100;

      // Store result for display
      ocrResult = result;

      devLog.info('OCR Complete:', result);

      // Show success toast
      toast.success(`${result.documentType} processed! Confidence: ${(result.confidence * 100).toFixed(0)}%`);

      // Dispatch event for parent components
      dispatch('documentProcessed', result);

      // Keep results visible for 5 seconds, then reset
      setTimeout(() => {
        uploading = false;
        selectedFile = null;
        progress = 0;
        ocrResult = null;
      }, 5000);

    } catch (err) {
      devLog.error('OCR processing failed:', err);

      const errorMsg = err?.message || String(err);
      toast.danger(`Processing failed: ${errorMsg}`);

      uploading = false;
      selectedFile = null;
      progress = 0;
      ocrResult = null;
    }
  }

  // Process file from browser DataTransfer (for Outlook/Thunderbird drag-drop)
  async function processFileFromDataTransfer(file: File) {
    if (!file || uploading) return;

    const fileName = file.name;
    devLog.info('Processing file from DataTransfer:', fileName);

    // SECURITY FIX: Validate file with magic bytes (prevents malware.exe renamed to invoice.pdf)
    const validation = await validateFile(file);
    if (!validation.valid) {
      toast.danger(validation.error || 'Invalid file');
      devLog.warn('File validation failed:', fileName, validation.error);
      return;
    }

    devLog.info('File validation passed:', fileName, 'Type:', validation.detectedType);

    uploading = true;
    selectedFile = fileName;
    progress = 0;
    ocrResult = null;

    try {
      // Read file as base64
      const arrayBuffer = await file.arrayBuffer();
      const base64 = btoa(
        new Uint8Array(arrayBuffer).reduce((data, byte) => data + String.fromCharCode(byte), '')
      );

      // Animate progress
      const progressInterval = setInterval(() => {
        progress = Math.min(progress + 5, 90);
      }, 150);

      // Call backend with base64 data
      const result = await QuickCaptureDocumentFromBase64(base64, fileName);

      clearInterval(progressInterval);
      progress = 100;

      ocrResult = result;
      devLog.info('OCR Complete (DataTransfer):', result);
      toast.success(`${result.documentType} processed!`);
      dispatch('documentProcessed', result);

      setTimeout(() => {
        uploading = false;
        selectedFile = null;
        progress = 0;
        ocrResult = null;
      }, 5000);

    } catch (err) {
      devLog.error('DataTransfer processing failed:', err);
      toast.danger(`Failed to process: ${err}`);
      uploading = false;
      selectedFile = null;
      progress = 0;
    }
  }
</script>

<div class="quick-capture">
  <div
    class="drop-zone"
    class:dragging
    class:uploading
    role="region"
    aria-label="Document upload area"
    ondragover={handleDragOver}
    ondragleave={handleDragLeave}
    ondrop={handleDrop}
  >
    {#if uploading && !ocrResult}
      <!-- Processing state -->
      <div class="upload-progress" role="status" aria-live="polite">
        <WabiSpinner size="md" />
        <p class="upload-file-name">{selectedFile}</p>
        <div
          class="progress-bar"
          role="progressbar"
          aria-valuenow={progress}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label="Upload progress"
        >
          <div class="progress-fill" style="width: {progress}%"></div>
        </div>
        <p class="upload-status">
          <span class="sr-only">Processing: </span>
          {progress < 100 ? 'Processing with OCR...' : 'Complete!'}
        </p>
      </div>
    {:else if ocrResult}
      <!-- Results state -->
      <div class="ocr-results" role="region" aria-live="polite">
        <div class="result-header">
          <span class="result-icon" aria-hidden="true">
            Doc
          </span>
          <h4>{ocrResult.documentType}</h4>
        </div>
        <div class="result-stats">
          <div class="stat">
            <span class="stat-label">Confidence</span>
            <span class="stat-value">{(ocrResult.confidence * 100).toFixed(0)}%</span>
          </div>
          <div class="stat">
            <span class="stat-label">Engine</span>
            <span class="stat-value">{ocrResult.engine || 'OCR'}</span>
          </div>
          <div class="stat">
            <span class="stat-label">Processing</span>
            <span class="stat-value">{ocrResult.processingTimeMS}ms</span>
          </div>
        </div>
        {#if ocrResult.extractedData}
          <div class="extracted-fields">
            {#each Object.entries(ocrResult.extractedData) as [key, value]}
              {#if value}
                <div class="field">
                  <span class="field-label">{key}</span>
                  <span class="field-value">{value}</span>
                </div>
              {/if}
            {/each}
          </div>
        {/if}
        <p class="result-hint">Results saved to Inbox</p>
      </div>
    {:else}
      <!-- Initial state -->
      <div class="drop-content">
        <span class="drop-icon" aria-hidden="true">Upload</span>
        <h3 id="quick-capture-title">Quick Capture</h3>
        <p class="drop-hint">Drag & drop RFQ, Invoice, or PO here</p>
        <p class="drop-formats">Supports: PDF, DOCX, MSG, EML, XLSX, Images</p>
      </div>
    {/if}
  </div>
</div>

<style>
  .quick-capture {
    width: 100%;
  }

  .drop-zone {
    border: 2px dashed rgba(0,0,0,0.15);
    border-radius: 8px;
    padding: 34px;
    text-align: center;
    background: rgba(255,255,255,0.3);
    transition: all 0.3s ease;
    cursor: pointer;
  }

  .drop-zone.dragging {
    border-color: #15803d;
    background: rgba(21, 128, 61, 0.08);
    transform: scale(1.02);
  }

  .drop-zone.uploading {
    border-color: rgba(0,0,0,0.1);
    background: rgba(0,0,0,0.02);
    cursor: default;
  }

  .drop-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 13px;
  }

  .drop-icon {
    font-size: 48px;
    opacity: 0.6;
  }

  h3 {
    font-family: Georgia, serif;
    font-size: 20px;
    font-weight: normal;
    margin: 0;
  }

  .drop-hint {
    font-family: Georgia, serif;
    font-size: 14px;
    color: #57534e;
    margin: 0;
  }

  .drop-formats {
    font-family: 'Courier Prime', monospace;
    font-size: 11px;
    color: #888;
    margin: 0;
  }

  .upload-progress {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 13px;
  }

  .upload-file-name {
    font-family: Georgia, serif;
    font-size: 14px;
    font-weight: 500;
    margin: 0;
  }

  .progress-bar {
    width: 100%;
    max-width: 300px;
    height: 8px;
    background: rgba(0,0,0,0.08);
    border-radius: 4px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #15803d 0%, #22c55e 100%);
    border-radius: 4px;
    transition: width 0.3s ease;
  }

  .upload-status {
    font-family: 'Courier Prime', monospace;
    font-size: 11px;
    color: #57534e;
    margin: 0;
  }

  /* Screen reader only text */
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border-width: 0;
  }

  /* Improved focus indicator for drop zone */
  .drop-zone:focus-visible {
    outline: 3px solid var(--color-ink, #1c1c1c);
    outline-offset: 4px;
  }

  /* OCR Results Display - Wabi-Sabi Style */
  .ocr-results {
    display: flex;
    flex-direction: column;
    gap: 21px;
    width: 100%;
  }

  .result-header {
    display: flex;
    align-items: center;
    gap: 13px;
    padding-bottom: 13px;
    border-bottom: 1px solid rgba(0,0,0,0.1);
  }

  .result-icon {
    font-size: 34px;
  }

  .result-header h4 {
    font-family: Georgia, serif;
    font-size: 18px;
    font-weight: 500;
    margin: 0;
    color: #1c1c1c;
  }

  .result-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 13px;
  }

  .stat {
    display: flex;
    flex-direction: column;
    gap: 5px;
    padding: 13px;
    background: rgba(0,0,0,0.02);
    border-radius: 6px;
  }

  .stat-label {
    font-family: 'Courier Prime', monospace;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: #888;
  }

  .stat-value {
    font-family: Georgia, serif;
    font-size: 16px;
    font-weight: 500;
    color: #1c1c1c;
  }

  .extracted-fields {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 200px;
    overflow-y: auto;
    padding: 8px;
    background: rgba(0,0,0,0.02);
    border-radius: 6px;
  }

  .field {
    display: flex;
    justify-content: space-between;
    padding: 8px;
    background: white;
    border-radius: 4px;
    font-size: 13px;
  }

  .field-label {
    font-family: 'Courier Prime', monospace;
    font-size: 11px;
    text-transform: capitalize;
    color: #57534e;
  }

  .field-value {
    font-family: Georgia, serif;
    font-size: 13px;
    color: #1c1c1c;
    font-weight: 500;
  }

  .result-hint {
    font-family: 'Courier Prime', monospace;
    font-size: 11px;
    text-align: center;
    color: #15803d;
    margin: 0;
    padding: 8px;
    background: rgba(21, 128, 61, 0.05);
    border-radius: 4px;
  }
</style>
