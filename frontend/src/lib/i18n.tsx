"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from "react";

type Messages = Record<string, string>;

interface I18nContextValue {
  locale: string;
  t: (key: string) => string;
  setLocale: (locale: string) => void;
}

const SUPPORTED = ["en", "ru", "uz", "kk", "tg"];
const FALLBACK = "en";
const STORAGE_KEY = "stickstock_locale";

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState(FALLBACK);
  const [messages, setMessages] = useState<Messages>({});
  const [fallbackMessages, setFallbackMessages] = useState<Messages>({});

  // Restore a previously chosen locale on mount (client-only; avoids an
  // SSR/client markup mismatch since localStorage isn't available server-side).
  useEffect(() => {
    const saved = typeof window !== "undefined" ? window.localStorage.getItem(STORAGE_KEY) : null;
    if (saved && SUPPORTED.includes(saved)) {
      setLocaleState(saved);
    }
  }, []);

  useEffect(() => {
    fetch(`/locales/${FALLBACK}.json`)
      .then((r) => r.json())
      .then(setFallbackMessages)
      .catch(() => setFallbackMessages({}));
  }, []);

  useEffect(() => {
    if (locale === FALLBACK) return;
    fetch(`/locales/${locale}.json`)
      .then((r) => r.json())
      .then(setMessages)
      .catch(() => setMessages({}));
  }, [locale]);

  const setLocale = useCallback((next: string) => {
    if (!SUPPORTED.includes(next)) return;
    setLocaleState(next);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, next);
    }
  }, []);

  const t = useCallback(
    (key: string) => {
      if (locale !== FALLBACK && messages[key]) return messages[key];
      return fallbackMessages[key] ?? key;
    },
    [locale, messages, fallbackMessages]
  );

  return <I18nContext.Provider value={{ locale, t, setLocale }}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
