import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import ThemeToggle from '../components/ThemeToggle'
import { useToast } from '../components/ToastProvider'
import { apiUrl, setSession } from '../lib/api'
import '../App.css'

export default function RegisterScreen() {
  const navigate = useNavigate()
  const toast = useToast()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()

    if (password.length < 8) {
      toast('A senha precisa ter pelo menos 8 caracteres.', 'error')
      return
    }
    if (password !== confirmPassword) {
      toast('As senhas não coincidem.', 'error')
      return
    }

    setLoading(true)
    try {
      const response = await fetch(apiUrl('/api/v1/auth/register'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })

      const data = await response.json().catch(() => ({}))

      if (!response.ok) {
        toast(data.error || 'Não foi possível criar a conta', 'error')
        return
      }

      setSession(data.token, !!data.is_admin)
      toast('Conta criada com sucesso!', 'success')
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
      <div className="logo">📝</div>
      <h1>Criar conta</h1>
      <p className="subtitle">
        Sua conta poderá escanear o QR Code para conectar o WhatsApp. Um administrador cuida dos disparos.
      </p>

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
          autoComplete="new-password"
          required
        />

        <label htmlFor="confirmPassword">Confirmar senha</label>
        <input
          id="confirmPassword"
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          autoComplete="new-password"
          required
        />

        <button className="sendBtn" type="submit" disabled={loading}>
          {loading ? 'Criando...' : 'Criar conta'}
        </button>
      </form>

      <p className="hint">
        Já tem conta?{' '}
        <button type="button" className="linkBtn" onClick={() => navigate('/login')}>
          Entrar
        </button>
      </p>
    </div>
  )
}
