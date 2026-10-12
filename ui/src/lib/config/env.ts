/**
 * Browser-visible configuration. Only VITE_* variables reach this bundle, and
 * they are public: never read secrets here.
 */
function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

export const env = {
  apiBaseUrl: trimTrailingSlash(import.meta.env.VITE_API_BASE_URL || '/api/v1'),
} as const
