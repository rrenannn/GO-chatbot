// Em produção o front (Vercel) e o back (Fly.io) ficam em domínios diferentes,
// então as chamadas precisam de uma URL absoluta. Em dev, a var fica vazia e
// o Vite faz proxy de /api para o backend local (ver vite.config.ts).
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? ''

const TOKEN_KEY = 'auth_token'
const IS_ADMIN_KEY = 'auth_is_admin'
// Guardam o token/e-mail do admin logado enquanto ele está "assumindo" outra conta,
// para permitir voltar para a própria sessão depois.
const ORIGINAL_TOKEN_KEY = 'auth_original_token'
const IMPERSONATING_EMAIL_KEY = 'auth_impersonating_email'

export function apiUrl(path: string): string {
  return `${API_BASE_URL}${path}`
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function isAdmin(): boolean {
  return localStorage.getItem(IS_ADMIN_KEY) === 'true'
}

export function setSession(token: string, admin: boolean): void {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(IS_ADMIN_KEY, admin ? 'true' : 'false')
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(IS_ADMIN_KEY)
  localStorage.removeItem(ORIGINAL_TOKEN_KEY)
  localStorage.removeItem(IMPERSONATING_EMAIL_KEY)
}

export function getImpersonatingEmail(): string | null {
  return localStorage.getItem(IMPERSONATING_EMAIL_KEY)
}

// Troca a sessão atual (do admin) pela de outro usuário, guardando a original
// para permitir voltar depois com stopImpersonating().
export function startImpersonating(token: string, email: string): void {
  const ownToken = getToken()
  if (ownToken && !localStorage.getItem(ORIGINAL_TOKEN_KEY)) {
    localStorage.setItem(ORIGINAL_TOKEN_KEY, ownToken)
  }
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(IS_ADMIN_KEY, 'true')
  localStorage.setItem(IMPERSONATING_EMAIL_KEY, email)
}

export function stopImpersonating(): void {
  const original = localStorage.getItem(ORIGINAL_TOKEN_KEY)
  if (original) {
    localStorage.setItem(TOKEN_KEY, original)
  }
  localStorage.setItem(IS_ADMIN_KEY, 'true')
  localStorage.removeItem(ORIGINAL_TOKEN_KEY)
  localStorage.removeItem(IMPERSONATING_EMAIL_KEY)
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
