export function formatUsd(value?: number | null) {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return "$0.00";
  }

  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2
  }).format(value);
}

export function formatStatus(
  status?: string,
  t?: (key: string, vars?: Record<string, string | number | undefined>) => string
) {
  if (!status) {
    return t ? t("status.unknown") : "unknown";
  }
  if (t) {
    const translated = t(`status.${status}`);
    if (translated === `status.${status}`) {
      return status.replace(/_/g, " ");
    }
    return translated;
  }
  return status.replace(/_/g, " ");
}

/** Human-readable relative timestamp, e.g. "3m ago", "2h ago", "1d ago". */
export function timeAgo(iso?: string) {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  const m = Math.floor(diff / 60000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}
