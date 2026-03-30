"use client";

import { useLanguage } from "./LanguageProvider";

export default function Trans({
  id,
  values
}: {
  id: string;
  values?: Record<string, string | number | undefined>;
}) {
  const { t } = useLanguage();
  return <>{t(id, values)}</>;
}
