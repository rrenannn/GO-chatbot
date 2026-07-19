import { useEffect, useRef, useState } from 'react'
import QRCode from 'react-qr-code'
import { useNavigate } from 'react-router-dom'
import StepIndicator from '../components/StepIndicator'
import ThemeToggle from '../components/ThemeToggle'
import { useToast } from '../components/ToastProvider'
import { apiUrlWithToken, clearToken } from '../lib/api'
import '../App.css'

type Status = 'connecting' | 'qr' | 'connected' | 'error'

export default function ConnectScreen() {
  const [status, setStatus] = useState<Status>('connecting')
  const [qrValue, setQrValue] = useState('')
  const navigate = useNavigate()
  const toast = useToast()
  const redirectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    const evtSource = new EventSource(apiUrlWithToken('/api/v1/whatsapp/qr'))

    evtSource.onmessage = (event) => {
      const data = JSON.parse(event.data)

      if (data.status === 'QR_CODE') {
        setStatus('qr')
        setQrValue(data.code)
      } else if (data.status === 'CONNECTED') {
        setStatus('connected')
        evtSource.close()
        redirectTimer.current = setTimeout(() => {
          navigate('/broadcast')
        }, 1800)
      } else if (data.status === 'ERROR') {
        setStatus('error')
        toast('Erro ao gerar o QR Code. Tente novamente.', 'error')
        evtSource.close()
      }
    }

    evtSource.onerror = () => {
      setStatus('error')
      toast('Erro ao conectar com a API.', 'error')
      evtSource.close()
    }

    return () => {
      evtSource.close()
      if (redirectTimer.current) clearTimeout(redirectTimer.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navigate])

  if (status === 'connected') {
    return (
      <div className="card">
        <div className="logo">💬</div>
        <div className="loadingScreen">
          <div className="connectedIcon">✅</div>
          <h1>Dispositivo conectado!</h1>
          <div className="spinner large" />
          <p>Preparando o painel de disparo...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="card">
      <div className="topBar">
        <ThemeToggle />
        <button
          className="logoutBtn"
          onClick={() => {
            clearToken()
            navigate('/login')
          }}
        >
          🚪 Sair
        </button>
      </div>
      <StepIndicator current={1} />
      <div className="logo">💬</div>
      <h1>Conexão WhatsApp</h1>
      <p className="subtitle">Escaneie o código para vincular o dispositivo</p>

      <div className={`statusPill ${status === 'error' ? 'error' : ''}`}>
        <span className={`dot ${status !== 'error' ? 'pulse' : ''}`} />
        <span>
          {status === 'connecting' && 'Conectando ao servidor...'}
          {status === 'qr' && 'Aguardando leitura do QR Code...'}
          {status === 'error' && 'Erro ao conectar com a API'}
        </span>
      </div>

      <div className={`qrFrame ${status !== 'qr' ? 'empty' : ''}`}>
        {status === 'qr' && qrValue && <QRCode value={qrValue} size={240} />}
        {status === 'connecting' && <div className="skeleton qrSkeleton" />}
      </div>

      <p className="hint">
        {status === 'error'
          ? 'Verifique se o servidor está no ar.'
          : 'Abra o WhatsApp no celular → Aparelhos conectados → Conectar um aparelho.'}
      </p>

      {status === 'error' && (
        <button className="retryBtn" onClick={() => window.location.reload()}>
          Tentar novamente
        </button>
      )}
    </div>
  )
}
