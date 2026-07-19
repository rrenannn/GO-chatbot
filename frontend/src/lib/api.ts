// Em produção o front (Vercel) e o back (Fly.io) ficam em domínios diferentes,
// então as chamadas precisam de uma URL absoluta. Em dev, a var fica vazia e
// o Vite faz proxy de /api para o backend local (ver vite.config.ts).
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? ''

export function apiUrl(path: string): string {
  return `${API_BASE_URL}${path}`
}
