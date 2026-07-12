import { GetTranslations } from "../../../wailsjs/go/main/InfraService";

export type Locale = "en" | "ar" | "hi" | "fr" | "es";

export const localeOptions: Array<{ value: Locale; label: string }> = [
  { value: "en", label: "English" },
  { value: "ar", label: "العربية" },
  { value: "hi", label: "हिन्दी" },
  { value: "fr", label: "Français" },
  { value: "es", label: "Español" },
];

let messages = $state<Record<string, string>>({});
let currentLocale = $state<Locale>("en");

export function t(key: string, ...args: unknown[]): string {
  const template = messages[key] || key;
  return template.replace(/\{(\d+)\}/g, (_, i) => String(args[Number(i)] ?? ""));
}

export function getLocale(): Locale {
  return currentLocale;
}

export async function setLocale(locale: Locale): Promise<void> {
  const response = await GetTranslations(locale);
  messages = response || {};
  currentLocale = locale;
  document.documentElement.lang = locale;
  document.documentElement.dir = locale === "ar" ? "rtl" : "ltr";
}

export async function initI18n(locale: string = "en"): Promise<void> {
  const normalized = isLocale(locale) ? locale : "en";
  await setLocale(normalized);
}

function isLocale(locale: string): locale is Locale {
  return localeOptions.some((option) => option.value === locale);
}
