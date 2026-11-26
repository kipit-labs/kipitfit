import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import en from "@/locales/en.json";
import fr from "@/locales/fr.json";

export const languages = ["en", "fr"] as const;
export type Language = (typeof languages)[number];

const STORAGE_KEY = "lang";

export function isLanguage(v: unknown): v is Language {
  return languages.includes(v as Language);
}

/** The last choice made on this device, else the browser's language, else English. */
function initialLanguage(): Language {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (isLanguage(saved)) return saved;
  } catch {
    // Storage blocked.
  }
  for (const tag of navigator.languages ?? [navigator.language]) {
    const base = tag?.split("-")[0];
    if (isLanguage(base)) return base;
  }
  return "en";
}

void i18n.use(initReactI18next).init({
  resources: { en: { translation: en }, fr: { translation: fr } },
  lng: initialLanguage(),
  fallbackLng: "en",
  supportedLngs: languages,
  // The catalogs are bundled, so the first render is already translated.
  initAsync: false,
  interpolation: { escapeValue: false },
});

const applyLanguage = (lng: string) => (document.documentElement.lang = lng);
applyLanguage(i18n.language);
i18n.on("languageChanged", applyLanguage);

/** Switches the UI language and remembers it on this device. */
export function setLanguage(lng: Language) {
  try {
    localStorage.setItem(STORAGE_KEY, lng);
  } catch {
    // Storage blocked; the choice lasts until reload.
  }
  if (i18n.language !== lng) void i18n.changeLanguage(lng);
}

/** The current language, for Intl formatters. */
export function currentLanguage(): Language {
  return isLanguage(i18n.language) ? i18n.language : "en";
}

export default i18n;
