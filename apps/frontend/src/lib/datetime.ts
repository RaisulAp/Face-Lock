// Date and time utilities.
// Important: All timestamps from backend are in UTC.
// Rendering defaults to application timezone (Asia/Jakarta unless configured).

export const DEFAULT_TIMEZONE = "Asia/Jakarta";

export function formatDateTime(isoString?: string | null, timeZone: string = DEFAULT_TIMEZONE): string {
  if (!isoString) return "-";
  try {
    const d = new Date(isoString);
    if (isNaN(d.getTime())) return "-";
    return new Intl.DateTimeFormat("id-ID", {
      dateStyle: "medium",
      timeStyle: "medium",
      timeZone,
    }).format(d);
  } catch {
    return isoString;
  }
}

export function formatDate(isoString?: string | null, timeZone: string = DEFAULT_TIMEZONE): string {
  if (!isoString) return "-";
  try {
    const d = new Date(isoString);
    if (isNaN(d.getTime())) return "-";
    return new Intl.DateTimeFormat("id-ID", {
      dateStyle: "medium",
      timeZone,
    }).format(d);
  } catch {
    return isoString;
  }
}

export function formatTime(isoString?: string | null, timeZone: string = DEFAULT_TIMEZONE): string {
  if (!isoString) return "-";
  try {
    const d = new Date(isoString);
    if (isNaN(d.getTime())) return "-";
    return new Intl.DateTimeFormat("id-ID", {
      timeStyle: "medium",
      timeZone,
    }).format(d);
  } catch {
    return isoString;
  }
}

export function getTodayDateString(timeZone: string = DEFAULT_TIMEZONE): string {
  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
  return formatter.format(new Date());
}

export function getStartOfMonthString(timeZone: string = DEFAULT_TIMEZONE): string {
  const today = getTodayDateString(timeZone);
  return `${today.substring(0, 8)}01`;
}
