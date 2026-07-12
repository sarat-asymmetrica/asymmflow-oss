// File Upload Security - MIME Type and Magic Byte Validation
// Prevents malware.exe renamed to invoice.pdf attacks

// Allowed file types with their MIME types and magic bytes
const FILE_SIGNATURES: Record<string, { mimes: string[], magic: number[][] }> = {
    pdf: {
        mimes: ['application/pdf'],
        magic: [[0x25, 0x50, 0x44, 0x46]] // %PDF
    },
    xlsx: {
        mimes: [
            'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
            'application/vnd.ms-excel'
        ],
        magic: [[0x50, 0x4B, 0x03, 0x04]] // PK.. (ZIP format)
    },
    xls: {
        mimes: ['application/vnd.ms-excel'],
        magic: [[0xD0, 0xCF, 0x11, 0xE0]] // OLE format
    },
    rtf: {
        mimes: ['application/rtf', 'text/rtf', 'text/plain'],
        magic: [[0x7B, 0x5C, 0x72, 0x74, 0x66]] // {\rtf
    },
    png: {
        mimes: ['image/png'],
        magic: [[0x89, 0x50, 0x4E, 0x47]] // .PNG
    },
    bmp: {
        mimes: ['image/bmp', 'image/x-ms-bmp'],
        magic: [[0x42, 0x4D]] // BM
    },
    jpg: {
        mimes: ['image/jpeg'],
        magic: [[0xFF, 0xD8, 0xFF]] // JPEG start
    },
    jpeg: {
        mimes: ['image/jpeg'],
        magic: [[0xFF, 0xD8, 0xFF]] // JPEG start
    },
    tiff: {
        mimes: ['image/tiff'],
        magic: [
            [0x49, 0x49, 0x2A, 0x00], // II*.
            [0x4D, 0x4D, 0x00, 0x2A]  // MM.*
        ]
    },
    tif: {
        mimes: ['image/tiff'],
        magic: [
            [0x49, 0x49, 0x2A, 0x00],
            [0x4D, 0x4D, 0x00, 0x2A]
        ]
    },
    webp: {
        mimes: ['image/webp'],
        magic: [[0x52, 0x49, 0x46, 0x46]] // RIFF....WEBP (prefix check)
    },
    msg: {
        mimes: ['application/vnd.ms-outlook', 'application/octet-stream'],
        magic: [[0xD0, 0xCF, 0x11, 0xE0]] // OLE format
    },
    docx: {
        mimes: ['application/vnd.openxmlformats-officedocument.wordprocessingml.document'],
        magic: [[0x50, 0x4B, 0x03, 0x04]] // PK.. (ZIP format)
    },
    doc: {
        mimes: ['application/msword'],
        magic: [[0xD0, 0xCF, 0x11, 0xE0]] // OLE format
    },
    xml: {
        mimes: ['application/xml', 'text/xml'],
        magic: [[0x3C, 0x3F, 0x78, 0x6D, 0x6C]] // <?xml
    },
    eml: {
        mimes: ['message/rfc822', 'application/octet-stream'],
        magic: [] // EML files are text-based, check content differently
    },
    csv: {
        mimes: ['text/csv', 'application/vnd.ms-excel', 'text/plain'],
        magic: [] // CSV files are text-based, no magic bytes
    },
    txt: {
        mimes: ['text/plain'],
        magic: [] // Text files have no magic bytes
    }
};

export interface ValidationResult {
    valid: boolean;
    error?: string;
    detectedType?: string;
}

/**
 * Validates a file by checking:
 * 1. File size (max 50MB)
 * 2. File extension (must be in allowed list)
 * 3. MIME type (must match expected for extension)
 * 4. Magic bytes (file header must match expected signature)
 */
export async function validateFile(file: File): Promise<ValidationResult> {
    // Step 1: Check file size (max 50MB)
    const MAX_SIZE = 50 * 1024 * 1024;
    if (file.size > MAX_SIZE) {
        return { valid: false, error: `File too large. Maximum size is 50MB.` };
    }

    // Step 2: Get extension
    const ext = file.name.split('.').pop()?.toLowerCase() || '';

    // Step 3: Check if extension is allowed
    if (!FILE_SIGNATURES[ext]) {
        const allowed = Object.keys(FILE_SIGNATURES).join(', ');
        return { valid: false, error: `File type .${ext} not allowed. Allowed: ${allowed}` };
    }

    // Step 4: Check MIME type
    const expectedMimes = FILE_SIGNATURES[ext].mimes;
    // Allow empty MIME type (some browsers don't set it) but validate if present
    if (file.type && !expectedMimes.includes(file.type)) {
        // Special case: some files may have generic octet-stream
        if (file.type !== 'application/octet-stream') {
            return {
                valid: false,
                error: `MIME type mismatch. Expected ${expectedMimes.join(' or ')}, got ${file.type}`
            };
        }
    }

    // Step 5: Check magic bytes (skip for text-based formats like EML)
    const expectedMagics = FILE_SIGNATURES[ext].magic;
    if (expectedMagics.length > 0) {
        try {
            const buffer = await file.slice(0, 8).arrayBuffer();
            const bytes = new Uint8Array(buffer);

            let magicMatch = false;

            for (const magic of expectedMagics) {
                let matches = true;
                for (let i = 0; i < magic.length; i++) {
                    if (bytes[i] !== magic[i]) {
                        matches = false;
                        break;
                    }
                }
                if (matches) {
                    magicMatch = true;
                    break;
                }
            }

            if (!magicMatch) {
                return {
                    valid: false,
                    error: `File content does not match .${ext} format. File may be corrupted or mislabeled.`
                };
            }
        } catch (e) {
            return { valid: false, error: 'Could not read file for validation.' };
        }
    }

    return { valid: true, detectedType: ext };
}

/**
 * Returns a string suitable for the HTML accept attribute
 * that restricts file selection to allowed types
 */
export function getAcceptString(): string {
    const mimes: string[] = [];
    const exts: string[] = [];

    for (const [ext, config] of Object.entries(FILE_SIGNATURES)) {
        exts.push(`.${ext}`);
        mimes.push(...config.mimes);
    }

    return Array.from(new Set([...exts, ...mimes])).join(',');
}

/**
 * Validates multiple files and returns results for each
 */
export async function validateFiles(files: FileList | File[]): Promise<Map<File, ValidationResult>> {
    const results = new Map<File, ValidationResult>();

    for (const file of Array.from(files)) {
        const result = await validateFile(file);
        results.set(file, result);
    }

    return results;
}
