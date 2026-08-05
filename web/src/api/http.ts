/** Central login redirect — one place for the 401 → /login policy. */
export function redirectToLogin(): void {
  window.location.href = '/login'
}

/**
 * fetch with session credentials that treats a 401 as an expired session:
 * redirect to login and throw so the caller's flow stops. Non-401 failures
 * stay with the caller — only the auth outcome is centralised here.
 */
export async function authAwareFetch(input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> {
  const res = await fetch(input, { ...init, credentials: 'include' })
  if (res.status === 401) {
    redirectToLogin()
    throw new Error('Unauthorized')
  }
  return res
}
