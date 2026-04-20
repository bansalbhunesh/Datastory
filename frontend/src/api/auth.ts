const configuredAPIKey = (import.meta.env.VITE_API_KEY as string | undefined)?.trim() ?? "";

export function apiHeaders(extra?: Record<string, string>): Record<string, string> {
  const headers: Record<string, string> = { ...(extra ?? {}) };
  if (configuredAPIKey) {
    headers["X-API-Key"] = configuredAPIKey;
  }
  return headers;
}
