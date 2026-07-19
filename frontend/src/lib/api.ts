// Em produção o front (Vercel) e o back (Fly.io) ficam em domínios diferentes,
// então as chamadas precisam de uma URL absoluta. Em dev, a var fica vazia e
// o Vite faz proxy de /api para o backend local (ver vite.config.ts).
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? ''

const TOKEN_KEY = 'auth_token'

export function apiUrl(path: string): string {
  return `${API_BASE_URL}${path}`
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

// Monta a URL de um endpoint SSE incluindo o token como query param, já que o
// EventSource do navegador não permite enviar o header Authorization.
export function apiUrlWithToken(path: string): string {
  const token = getToken() ?? ''
  const separator = path.includes('?') ? '&' : '?'
  return `${apiUrl(path)}${separator}token=${encodeURIComponent(token)}`
}

// fetch autenticado — adiciona o Bearer token automaticamente.
export async function authFetch(path: string, options: RequestInit = {}): Promise<Response> {
  const token = getToken()
  const headers = new Headers(options.headers)
  headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)

  return fetch(apiUrl(path), { ...options, headers })
}
