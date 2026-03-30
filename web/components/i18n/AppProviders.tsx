"use client";

import type { ReactNode } from "react";
import { LanguageProvider } from "./LanguageProvider";

export default function AppProviders({ children }: { children: ReactNode }) {
  return <LanguageProvider>{children}</LanguageProvider>;
}
