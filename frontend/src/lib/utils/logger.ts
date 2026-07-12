/**
 * Production-Safe Logger Utility
 *
 * Prevents sensitive data leakage in production builds by only logging
 * in development mode. Critical errors always log for debugging.
 *
 * Usage:
 *   import { logger } from '$lib/utils/logger';
 *   logger.debug('User data:', userData);  // Only logs in dev
 *   logger.error('Critical error:', err);  // Always logs
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

interface LoggerConfig {
    enableInProduction: boolean;
    minLevel: LogLevel;
}

const config: LoggerConfig = {
    enableInProduction: false,  // Never log sensitive data in production
    minLevel: 'debug'
};

const isDevelopment = import.meta.env.DEV;

export const logger = {
    /**
     * Debug logging (dev only)
     * Use for: development debugging, verbose data inspection
     */
    debug: (...args: any[]) => {
        if (isDevelopment) {
            console.debug(...args);
        }
    },

    /**
     * Info logging (dev only)
     * Use for: general information, non-sensitive state changes
     */
    info: (...args: any[]) => {
        if (isDevelopment) {
            console.log(...args);
        }
    },

    /**
     * Warning logging (dev only)
     * Use for: potential issues, deprecation warnings
     */
    warn: (...args: any[]) => {
        if (isDevelopment) {
            console.warn(...args);
        }
    },

    /**
     * Error logging (ALWAYS logs)
     * Use for: exceptions, critical failures that need debugging in production
     * CAUTION: Never log sensitive user data (passwords, tokens, PII)
     */
    error: (...args: any[]) => {
        console.error(...args);
    },

    /**
     * Performance timing (dev only)
     * Use for: measuring operation durations
     */
    time: (label: string) => {
        if (isDevelopment) {
            console.time(label);
        }
    },

    timeEnd: (label: string) => {
        if (isDevelopment) {
            console.timeEnd(label);
        }
    },

    /**
     * Group logging (dev only)
     * Use for: collapsible log groups
     */
    group: (label: string) => {
        if (isDevelopment) {
            console.group(label);
        }
    },

    groupEnd: () => {
        if (isDevelopment) {
            console.groupEnd();
        }
    }
};

/**
 * Safe error logging that sanitizes sensitive fields
 * Use this for errors that might contain user data
 */
export function logErrorSafe(message: string, error: any) {
    const sanitized = {
        message: error?.message || 'Unknown error',
        type: error?.constructor?.name || 'Error',
        // Never log full error object (might contain sensitive data)
    };
    logger.error(message, sanitized);
}
