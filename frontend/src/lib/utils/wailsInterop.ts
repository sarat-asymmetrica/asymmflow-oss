type WailsFactory<T> = {
  createFrom(source?: any): T;
};

export function buildWailsInput<T>(factory: WailsFactory<T>, value: Record<string, any>): T {
  return factory.createFrom(value);
}

export function normalizeWailsDateTime(value: unknown): string | undefined {
  if (value === null || value === undefined || value === "") {
    return undefined;
  }

  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return undefined;
    const parsed = new Date(trimmed);
    return Number.isNaN(parsed.getTime()) ? trimmed : parsed.toISOString();
  }

  if (typeof value === "number") {
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? undefined : parsed.toISOString();
  }

  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? undefined : value.toISOString();
  }

  if (typeof value === "object") {
    const record = value as Record<string, unknown>;

    for (const key of ["Time", "time", "value", "Value", "date", "Date"]) {
      const nested = normalizeWailsDateTime(record[key]);
      if (nested) return nested;
    }

    if (typeof record.toString === "function") {
      const text = record.toString();
      if (text && text !== "[object Object]") {
        return normalizeWailsDateTime(text) ?? text;
      }
    }
  }

  return undefined;
}

export function toWailsDate(value: unknown): Date | null {
  const normalized = normalizeWailsDateTime(value);
  if (!normalized) return null;
  const parsed = new Date(normalized);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}
