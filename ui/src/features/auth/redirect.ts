/**
 * Where to go after signing in. Only paths inside this app are allowed, so a
 * crafted link cannot bounce the user to another site (no open redirect).
 */
export function safeDestination(from: unknown): string {
  if (typeof from !== 'string' || !from.startsWith('/') || from.startsWith('//')) return '/'
  if (from.startsWith('/\\') || from === '/login' || from.startsWith('/login?')) return '/'
  return from
}
