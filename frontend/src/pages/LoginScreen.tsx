import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import ThemeToggle from '../components/ThemeToggle'
import { useToast } from '../components/ToastProvider'
import { apiUrl, setToken } from '../lib/api'
import '../App.css'

export default function LoginScreen() {
  const navigate = useNavigate()
  const toast = useToast()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)

    try {
      const response = await fetch(apiUrl('/api/v1/auth/login'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })

      const data = await response.json().catch(() => ({}))

      if (!response.ok) {
        toast(data.error || 'E-mail ou senha inválidos', 'error')
        return
      }

      setToken(data.token)
      navigate('/')
    } catch {
      toast('Erro ao conectar com a API.', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card">
      <div className="topBar">
        <ThemeToggle />
        <span />
      </div>
      <div className="logo">🔒</div>
      <h1>Entrar</h1>
      <p className="subtitle">Acesso restrito ao painel de WhatsApp</p>

      <form onSubmit={handleSubmit}>
        <label htmlFor="email">E-mail</label>
        <input
          id="email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="username"
          required
        />

        <label htmlFor="password">Senha</label>
        <input
          id="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
          required
        />

        <button className="sendBtn" type="submit" disabled={loading}>
          {loading ? 'Entrando...' : 'Entrar'}
        </button>
      </form>

      <p className="hint">Não tem acesso? Fale com o administrador do sistema.</p>
    </div>
  )
}
