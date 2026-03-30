"use client";

import { createContext, useContext, useEffect } from "react";
import type { ReactNode } from "react";
import { translate } from "@/lib/i18n";

type LanguageContextValue = {
  t: (
    key: string,
    vars?: Record<string, string | number | undefined>,
  ) => string;
};

const LanguageContext = createContext<LanguageContextValue | undefined>(
  undefined,
);

export function LanguageProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    document.documentElement.lang = "en";
    document.documentElement.dataset.lang = "en";
  }, []);

  return (
    <LanguageContext.Provider value={{ t: translate }}>
      {children}
    </LanguageContext.Provider>
  );
}

export function useLanguage() {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error("useLanguage must be used within LanguageProvider");
  }
  return context;
}
