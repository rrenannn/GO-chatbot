import { useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import StepIndicator from '../components/StepIndicator'
import ConfirmModal from '../components/ConfirmModal'
import ThemeToggle from '../components/ThemeToggle'
import { useToast } from '../components/ToastProvider'
import { apiUrl } from '../lib/api'
import '../App.css'

const MAX_RECIPIENTS = 200

type Contact = { phone: string; name?: string }

type LogEntry = {
  phone: string
  success: boolean
  error?: string
}

type ProgressEvent = {
  status: 'PROGRESS' | 'ERROR' | 'DONE'
  phone?: string
  success?: boolean
  error?: string
  sent?: number
  total?: number
}

// Números brasileiros sem DDI (DDD + 8 ou 9 dígitos) recebem o "55" automaticamente
function normalizePhone(digits: string): string {
  if ((digits.length === 10 || digits.length === 11) && !digits.startsWith('55')) {
    return '55' + digits
  }
  return digits
}

// Aceita "telefone" ou "telefone,nome" por linha (também funciona com CSV simples)
function parseContacts(raw: string): Contact[] {
  const contacts: Contact[] = []
  const seen = new Set<string>()

  for (const rawLine of raw.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line) continue

    const parts = line.split(',')
    const phone = normalizePhone(parts[0].replace(/\D/g, ''))
    const name = parts.slice(1).join(',').trim()

    if (!phone || seen.has(phone)) continue
    seen.add(phone)
    contacts.push({ phone, name: name || undefined })
  }

  return contacts
}

export default function BroadcastScreen() {
  const navigate = useNavigate()
  const toast = useToast()
  const [phonesRaw, setPhonesRaw] = useState('')
  const [message, setMessage] = useState('')
  const [sending, setSending] = useState(false)
  const [progress, setProgress] = useState({ sent: 0, total: 0 })
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [dragging, setDragging] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const messageRef = useRef<HTMLTextAreaElement>(null)

  const contacts = parseContacts(phonesRaw)

  function insertVariable(token: string) {
    const el = messageRef.current
    if (!el) {
      setMessage((m) => m + token)
      return
    }
    const start = el.selectionStart ?? message.length
    const end = el.selectionEnd ?? message.length
    const next = message.slice(0, start) + token + message.slice(end)
    setMessage(next)
    requestAnimationFrame(() => {
      el.focus()
      el.selectionStart = el.selectionEnd = start + token.length
    })
  }

  function loadFile(file: File) {
    if (!/\.(csv|txt)$/i.test(file.name)) {
      toast('Envie um arquivo .csv ou .txt', 'error')
      return
    }
    const reader = new FileReader()
    reader.onload = () => {
      const text = String(reader.result ?? '')
      setPhonesRaw((prev) => (prev.trim() ? prev.trim() + '\n' + text : text))
      toast('Contatos importados do arquivo', 'success')
    }
    reader.readAsText(file, 'utf-8')
  }

  function requestBroadcast() {
    if (contacts.length === 0) {
      toast('Adicione ao menos um número válido.', 'error')
      return
    }
    if (!message.trim()) {
      toast('Digite a mensagem a ser enviada.', 'error')
      return
    }
    if (contacts.length > MAX_RECIPIENTS) {
      toast(`Máximo de ${MAX_RECIPIENTS} números por disparo.`, 'error')
      return
    }
    setConfirmOpen(true)
  }

  async function startBroadcast() {
    setConfirmOpen(false)
    setSending(true)
    setLogs([])
    setProgress({ sent: 0, total: contacts.length })

    try {
      const response = await fetch(apiUrl('/api/v1/broadcast'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ contacts, message }),
      })

      if (!response.ok || !response.body) {
        const errData = await response.json().catch(() => ({}))
        throw new Error(errData.error || 'Falha ao iniciar disparo')
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const events = buffer.split('\n\n')
        buffer = events.pop() ?? ''

        for (const evt of events) {
          const line = evt.replace(/^data: /, '').trim()
          if (!line) continue
          const data: ProgressEvent = JSON.parse(line)
          handleEvent(data)
        }
      }
      toast('Disparo concluído!', 'success')
    } catch (e) {
      toast(e instanceof Error ? e.message : String(e), 'error')
    } finally {
      setSending(false)
    }
  }

  function handleEvent(data: ProgressEvent) {
    if (data.status === 'PROGRESS' && data.phone !== undefined) {
      setProgress({ sent: data.sent ?? 0, total: data.total ?? contacts.length })
      setLogs((prev) => [
        { phone: data.phone!, success: !!data.success, error: data.error },
        ...prev,
      ])
    } else if (data.status === 'ERROR') {
      toast(data.error ?? 'Erro desconhecido', 'error')
    }
  }

  const pct = progress.total > 0 ? Math.round((progress.sent / progress.total) * 100) : 0
  const remaining = progress.total - progress.sent
  const etaMin = remaining > 0 ? Math.ceil((remaining * 10) / 60) : 0

  return (
    <div className="card wide">
      <div className="topBar">
        <ThemeToggle />
        <button className="linkBtn" onClick={() => navigate('/')}>
          Reconectar dispositivo
        </button>
      </div>
      <StepIndicator current={2} />
      <div className="logo">📢</div>
      <h1>Disparo em Massa</h1>
      <p className="subtitle">Envie a mesma mensagem para vários contatos com segurança</p>

      <label htmlFor="phones">Números (um por linha, com DDD — opcional: telefone,nome)</label>
      <textarea
        id="phones"
        rows={5}
        placeholder={'11999999999,Maria\n11988888888'}
        value={phonesRaw}
        onChange={(e) => setPhonesRaw(e.target.value)}
        disabled={sending}
      />
      <div
        className={`dropzone ${dragging ? 'dragging' : ''}`}
        onClick={() => fileInputRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragging(false)
          const file = e.dataTransfer.files?.[0]
          if (file) loadFile(file)
        }}
      >
        📁 Arraste um arquivo .csv/.txt aqui ou clique para importar
        <input
          ref={fileInputRef}
          type="file"
          accept=".csv,.txt"
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) loadFile(file)
            e.target.value = ''
          }}
        />
      </div>
      <div className="fieldHint">
        {contacts.length} número{contacts.length === 1 ? '' : 's'} válido
        {contacts.length === 1 ? '' : 's'} · o código do Brasil (55) é adicionado automaticamente
      </div>

      <label htmlFor="message">Mensagem</label>
      <textarea
        id="message"
        ref={messageRef}
        rows={4}
        placeholder="Digite a mensagem que será enviada para todos os contatos..."
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        disabled={sending}
      />
      <div className="messageToolbar">
        <button type="button" className="varChip" onClick={() => insertVariable('{{nome}}')}>
          + nome do contato
        </button>
      </div>

      <div className="safetyNote">
        ⚠️ Para reduzir o risco de bloqueio, as mensagens são enviadas uma por vez com
        intervalo aleatório de 5 a 15 segundos entre cada envio. Limite de {MAX_RECIPIENTS}{' '}
        contatos por disparo.
      </div>

      <button className="sendBtn" onClick={requestBroadcast} disabled={sending}>
        {sending ? 'Enviando...' : 'Iniciar Disparo'}
      </button>

      {progress.total > 0 && (
        <div className="progressWrap">
          <div className="progressBarTrack">
            <div className="progressBarFill" style={{ width: `${pct}%` }} />
          </div>
          <div className="progressMeta">
            <span>
              {progress.sent} / {progress.total}
            </span>
            <span>{remaining > 0 ? `~${etaMin} min restante(s)` : 'Concluído'}</span>
          </div>
          <div className="logList">
            {logs.map((log, i) => (
              <div key={i} className={`logItem ${log.success ? 'ok' : 'fail'}`}>
                <span>{log.phone}</span>
                <span>{log.success ? '✔ enviado' : `✖ ${log.error || 'falhou'}`}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <ConfirmModal
        open={confirmOpen}
        title="Confirmar disparo em massa"
        description={`Enviar para ${contacts.length} contato(s)? O envio pode levar alguns minutos por segurança.`}
        confirmLabel="Enviar"
        onConfirm={startBroadcast}
        onCancel={() => setConfirmOpen(false)}
      />
    </div>
  )
}
