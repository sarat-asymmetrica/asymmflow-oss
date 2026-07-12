/**
 * Development-only logging utility
 * Automatically strips logs in production builds
 */

const isDev = (): boolean => {
  try {
    return Boolean((import.meta as ImportMeta & { env?: { DEV?: boolean } }).env?.DEV);
  } catch {
    return false;
  }
};

export const devLog = {
  log: (...args: any[]) => {
    if (isDev()) {
      console.log(...args);
    }
  },

  info: (...args: any[]) => {
    if (isDev()) {
      console.info(...args);
    }
  },

  warn: (...args: any[]) => {
    if (isDev()) {
      console.warn(...args);
    }
  },

  error: (...args: any[]) => {
    // Always log errors, even in production
    console.error(...args);
  },

  debug: (...args: any[]) => {
    if (isDev()) {
      console.debug(...args);
    }
  }
};
