export function formatPercent(val?: number | null, decimals: number = 1): string {
  if (val === null || val === undefined || isNaN(val)) return "-";
  return `${(val * 100).toFixed(decimals)}%`;
}

export function formatNumber(val?: number | null): string {
  if (val === null || val === undefined || isNaN(val)) return "-";
  return new Intl.NumberFormat("id-ID").format(val);
}

export function formatDurationMinutes(minutes?: number | null): string {
  if (minutes === null || minutes === undefined) return "-";
  const h = Math.floor(minutes / 60);
  const m = Math.round(minutes % 60);
  if (h === 0) return `${m}m`;
  if (m === 0) return `${h}j`;
  return `${h}j ${m}m`;
}

export function formatDistance(meters?: number | null): string {
  if (meters === null || meters === undefined) return "-";
  if (meters < 1000) return `${Math.round(meters)} m`;
  return `${(meters / 1000).toFixed(2)} km`;
}
