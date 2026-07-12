#!/usr/bin/env python3
"""
PyMuPDF Batch Extractor - Processes multiple PDFs in one call
Eliminates subprocess overhead for maximum throughput.

Usage:
    python pymupdf_batch.py file1.pdf file2.pdf ...
    python pymupdf_batch.py --stdin  # Read file list from stdin (one per line)
    
Output: JSON array of results
"""

import sys
import json
import time

try:
    import fitz  # PyMuPDF
except ImportError:
    print(json.dumps({"error": "PyMuPDF not installed. Run: pip install pymupdf"}))
    sys.exit(1)


def extract_pdf(filepath: str) -> dict:
    """Extract text from a single PDF."""
    start = time.perf_counter()
    
    try:
        doc = fitz.open(filepath)
        text = ""
        is_vector = False
        page_count = doc.page_count
        
        for page_num in range(page_count):
            page = doc.load_page(page_num)
            page_text = page.get_text()
            if page_text.strip():
                is_vector = True
            text += page_text + "\n"
        
        doc.close()
        
        method = "vector_pdf" if is_vector and len(text.strip()) > 50 else "scanned_pdf"
        
        return {
            "success": True,
            "filepath": filepath,
            "text": text.strip(),
            "method": method,
            "pages": page_count,
            "characters": len(text.strip()),
            "duration_ms": int((time.perf_counter() - start) * 1000)
        }
    except Exception as e:
        return {
            "success": False,
            "filepath": filepath,
            "error": str(e),
            "method": "pymupdf_error",
            "duration_ms": int((time.perf_counter() - start) * 1000)
        }


def main():
    start_time = time.perf_counter()
    
    # Get file list
    if len(sys.argv) > 1 and sys.argv[1] == "--stdin":
        # Read from stdin
        files = [line.strip() for line in sys.stdin if line.strip()]
    elif len(sys.argv) > 1:
        # From command line args
        files = sys.argv[1:]
    else:
        print(json.dumps({"error": "No files provided"}))
        sys.exit(1)
    
    # Process all files
    results = []
    for filepath in files:
        result = extract_pdf(filepath)
        results.append(result)
    
    total_time = time.perf_counter() - start_time
    
    # Output summary
    output = {
        "total_files": len(files),
        "total_time_ms": int(total_time * 1000),
        "throughput_per_sec": len(files) / total_time if total_time > 0 else 0,
        "results": results
    }
    
    print(json.dumps(output))


if __name__ == "__main__":
    main()
