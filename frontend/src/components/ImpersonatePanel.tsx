import { useState, type FormEvent } from 'react'
import { authFetch, getImpersonatingEmail, startImpersonating, stopImpersonating } from '../lib/api'
import { useToast } from './ToastProvider'
import '../App.css'

export default function ImpersonatePanel() {
  const toast = useToast()
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const impersonating = getImpersonatingEmail()

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)

    try {
      const response = await authFetch('/api/v1/impersonate', {
        method: 'POST',
        body: JSON.stringify({ email }),
      })
      const data = await response.json().catch(() => ({}))

      if (!response.ok) {
        toast(data.error || 'Não foi possível assumir essa conta', 'error')
        return
      }

      startImpersonating(data.token, data.email)
      toast(`Agora você está agindo como ${data.email}`, 'success')
      window.location.hash = '/'
      window.location.reload()
    } catch {
      toast('Erro ao conectar com a API.', 'error')
    } finally {
      setLoading(false)
    }
  }

  function handleStop() {
    stopImpersonating()
    window.location.hash = '/'
    window.location.reload()
  }

  if (impersonating) {
    return (
      <div className="safetyNote" style={{ marginTop: 18, marginBottom: 0 }}>
        <span>
          🎭 Você está agindo como <strong>{impersonating}</strong>.
        </span>
        <button type="button" className="linkBtn" onClick={handleStop} style={{ marginLeft: 'auto' }}>
          Voltar para minha conta
        </button>
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit} style={{ marginTop: 18 }}>
      <label htmlFor="impersonateEmail">Admin: enviar como outro usuário</label>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          id="impersonateEmail"
          type="email"
          placeholder="email@usuario.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <button type="submit" className="logoutBtn" disabled={loading || !email}>
          {loading ? '...' : 'Assumir'}
        </button>
      </div>
    </form>
  )
}
