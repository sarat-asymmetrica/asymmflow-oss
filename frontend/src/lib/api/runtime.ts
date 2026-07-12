/**
 * Asymmetrica Runtime API Client
 *
 * Connects Sovereign UI to the .NET Asymmetrica.Runtime service
 * for AI kernels, graph processing, and document analysis.
 *
 * STATE FIXES:
 * - Retry logic for transient failures
 * - AbortController support
 * - Proper error handling
 */

import { fetchWithRetry, fetchJsonWithRetry, postJsonWithRetry } from '../utils/fetchWithRetry';
import { RUNTIME_URL } from '../config';

const RUNTIME_BASE_URL = RUNTIME_URL;

export interface RuntimeHealth {
    status: 'healthy' | 'unhealthy' | 'unknown';
    version?: string;
    uptime?: string;
    kernelCount?: number;
    error?: string;
}

export interface Kernel {
    name: string;
    description: string;
    inputType: string;
    outputType: string;
    tags: string[];
}

export interface ExecuteRequest {
    kernelName: string;
    input: Record<string, unknown>;
}

export interface ExecuteResponse {
    success: boolean;
    output?: Record<string, unknown>;
    error?: string;
    executionTimeMs?: number;
}

/**
 * Check if Asymmetrica.Runtime is running and healthy
 */
export async function checkRuntimeHealth(signal?: AbortSignal): Promise<RuntimeHealth> {
    try {
        const response = await fetchWithRetry(`${RUNTIME_BASE_URL}/health`, {
            method: 'GET',
            headers: { 'Accept': 'application/json' },
            timeout: 3000,
            retries: 1, // Only one retry for health check
            signal,
        });

        const data = await response.json();
        return {
            status: 'healthy',
            version: data.version || '2.0.0',
            uptime: data.uptime,
            kernelCount: data.kernelCount,
        };
    } catch (error) {
        return {
            status: 'unknown',
            error: error instanceof Error ? error.message : 'Connection failed',
        };
    }
}

/**
 * List all available AI kernels
 */
export async function listKernels(signal?: AbortSignal): Promise<Kernel[]> {
    return fetchJsonWithRetry(`${RUNTIME_BASE_URL}/api/ai/kernels`, {
        retries: 2,
        timeout: 5000,
        signal,
    });
}

/**
 * Execute an AI kernel
 */
export async function executeKernel(
    request: ExecuteRequest,
    signal?: AbortSignal
): Promise<ExecuteResponse> {
    try {
        const data: any = await postJsonWithRetry(
            `${RUNTIME_BASE_URL}/api/ai/execute`,
            request,
            {
                retries: 2,
                timeout: 30000, // AI execution can take longer
                signal,
            }
        );

        return {
            success: true,
            output: data?.output,
            executionTimeMs: data?.executionTimeMs,
        };
    } catch (error) {
        return {
            success: false,
            error: error instanceof Error ? error.message : 'Execution failed',
        };
    }
}

/**
 * Read a document as a graph
 */
export async function readDocument(
    appType: 'excel' | 'word' | 'outlook' | 'powerpoint' | 'onenote' | 'pdf',
    filePath: string,
    signal?: AbortSignal
): Promise<{ nodes: unknown[]; edges: unknown[] }> {
    return postJsonWithRetry(
        `${RUNTIME_BASE_URL}/api/${appType}/read`,
        { filePath },
        {
            retries: 2,
            timeout: 10000,
            signal,
        }
    );
}
