import { useEffect, useRef, useState, useCallback } from 'react'
import QRDisplay from './components/QRDisplay'
import ResultModal from './components/ResultModal'
import Footer from './components/Footer'

interface ModalState {
  isOpen: boolean
  title: string
  url: string
  shortUrl?: string
}

export default function App() {
  const [token, setToken] = useState<string | null>(null)
  const [url, setUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [sessionUrl, setSessionUrl] = useState<string | null>(null)
  const [wsError, setWsError] = useState<string | null>(null)
  const [status, setStatus] = useState<{
    type: 'idle' | 'success' | 'error'
    message?: string
    shortUrl?: string
  }>({ type: 'idle' })
  const [copied, setCopied] = useState(false)

  const handleCopy = async (text: string) => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text)
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = text
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        document.execCommand('copy')
        document.body.removeChild(textarea)
      }
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  const [modalData, setModalData] = useState<ModalState | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    const path = window.location.pathname
    const match = path.match(/^\/session\/(.+)$/)
    if (match && match[1]) {
      setToken(match[1])
    }
  }, [])

  const connectWebSocket = useCallback(() => {
    if (typeof window === 'undefined') return

    if (wsRef.current) {
      wsRef.current.close()
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string)
        if (msg.type === 'session_created') {
          setSessionUrl(msg.session_url)
          setWsError(null)
        } else if (msg.type === 'link_received') {
          setModalData({
            isOpen: true,
            title: 'Входящая ссылка на экран',
            url: msg.original_url || msg.short_url,
            shortUrl: msg.short_url,
          })
        } else if (msg.type === 'error') {
          setWsError(msg.message || 'Ошибка сервера')
        }
      } catch (err) {
        console.error('Failed to parse WS message:', err)
      }
    }

    ws.onerror = () => {
      setWsError('Не удалось подключиться к серверу')
    }

    ws.onclose = () => {
      reconnectTimeoutRef.current = setTimeout(() => {
        connectWebSocket()
      }, 3000)
    }
  }, [])

  useEffect(() => {
    if (token) return

    connectWebSocket()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
      }
    }
  }, [token, connectWebSocket])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    let targetUrl = url.trim()
    if (!targetUrl) return

    if (!/^https?:\/\//i.test(targetUrl)) {
      targetUrl = 'https://' + targetUrl
    }

    setLoading(true)
    setStatus({ type: 'idle' })

    if (token) {
      // Send to active screen session
      try {
        const res = await fetch('/api/session/push', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            session_token: token,
            url: targetUrl,
          }),
        })

        const data = (await res.json()) as { error?: string; short_url?: string }

        if (!res.ok) {
          let errorMsg = data.error || 'Ошибка при отправке ссылки'
          if (res.status === 410) {
            errorMsg = 'Компьютер отключился или сессия устарела.'
          }
          setStatus({ type: 'error', message: errorMsg })
          return
        }

        setStatus({
          type: 'success',
          message: 'Ссылка отправлена на экран',
          shortUrl: data.short_url,
        })
        setUrl('')
      } catch {
        setStatus({
          type: 'error',
          message: 'Ошибка сети. Проверьте подключение к интернету.',
        })
      } finally {
        setLoading(false)
      }
    } else {
      // Standalone shorten URL
      try {
        const res = await fetch('/api/shorten', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url: targetUrl }),
        })

        const data = (await res.json()) as { error?: string; short_url?: string }

        if (!res.ok) {
          setStatus({
            type: 'error',
            message: data.error || 'Не удалось сократить ссылку',
          })
          return
        }

        if (data.short_url) {
          setModalData({
            isOpen: true,
            title: 'Ссылка сокращена',
            url: data.short_url,
            shortUrl: data.short_url,
          })
          setUrl('')
        }
      } catch {
        setStatus({
          type: 'error',
          message: 'Ошибка сети. Проверьте подключение к интернету.',
        })
      } finally {
        setLoading(false)
      }
    }
  }

  return (
    <div className="min-h-screen flex flex-col justify-between items-center px-4">
      {/* Header */}
      <header className="w-full max-w-md mx-auto pt-6 pb-2 flex justify-center items-center">
        <span className="font-semibold text-sm tracking-tight text-gray-400">
          short.steins.ru
        </span>
      </header>

      {/* Main content - Vertical Flex */}
      <main className="w-full max-w-md mx-auto flex-1 flex flex-col items-center justify-center py-6 gap-6">
        {/* Section 1: Input with action button */}
        <div className="w-full bg-surface p-6 sm:p-7 rounded-3xl border border-white/5 shadow-2xl flex flex-col gap-4">
          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <input
              type="text"
              required
              placeholder="Вставьте ссылку (https://...)"
              value={url}
              onChange={(e) => {
                setUrl(e.target.value)
                if (status.type !== 'idle') {
                  setStatus({ type: 'idle' })
                  setCopied(false)
                }
              }}
              className="w-full px-4 py-3 bg-bg rounded-xl border border-white/10 focus:border-accent focus:outline-none text-white text-sm placeholder:text-gray-600 transition-colors"
            />

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 px-6 rounded-xl bg-accent hover:bg-accentHover disabled:opacity-50 text-white font-medium text-sm transition-colors shadow-lg shadow-accent/25 flex items-center justify-center gap-2"
            >
              {loading ? (
                <div className="w-5 h-5 rounded-full border-2 border-white border-t-transparent animate-spin" />
              ) : token ? (
                'Открыть на экране'
              ) : (
                'Сократить'
              )}
            </button>
          </form>

          {/* Concise green notification or red error directly below input */}
          {status.type === 'success' && (
            <div className="flex flex-col gap-2.5 p-3.5 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 text-xs">
              <div className="flex items-center gap-2 text-emerald-400 font-medium">
                <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                </svg>
                <span>{status.message}</span>
              </div>

              {status.shortUrl && (
                <div className="flex items-center gap-2 p-2.5 bg-bg/80 rounded-xl border border-white/10">
                  <a
                    href={status.shortUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex-1 min-w-0 font-mono text-xs text-white/90 underline decoration-emerald-500/50 underline-offset-2 hover:text-white break-all select-all"
                  >
                    {status.shortUrl}
                  </a>
                  <button
                    type="button"
                    onClick={() => handleCopy(status.shortUrl!)}
                    title={copied ? 'Скопировано!' : 'Скопировать ссылку'}
                    aria-label="Скопировать ссылку"
                    className={`p-1.5 rounded-lg transition-colors flex items-center justify-center shrink-0 ${
                      copied
                        ? 'bg-emerald-500/20 text-emerald-400'
                        : 'text-gray-400 hover:text-white hover:bg-white/10'
                    }`}
                  >
                    {copied ? (
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2.5}
                          d="M5 13l4 4L19 7"
                        />
                      </svg>
                    ) : (
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <rect x="9" y="9" width="13" height="13" rx="2" strokeWidth={2} />
                        <path
                          d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"
                          strokeWidth={2}
                        />
                      </svg>
                    )}
                  </button>
                </div>
              )}
            </div>
          )}

          {status.type === 'error' && (
            <div className="flex items-center gap-2 p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-xs">
              <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <circle cx="12" cy="12" r="10" strokeWidth={2} />
                <path d="M12 8v4m0 4h.01" strokeWidth={2} strokeLinecap="round" />
              </svg>
              <span>{status.message}</span>
            </div>
          )}
        </div>

        {/* Section 2: QR code with short instruction (only on main page, hidden on /session/:token) */}
        {!token && (
          <div className="w-full bg-surface p-6 sm:p-7 rounded-3xl border border-white/5 shadow-2xl flex flex-col items-center gap-4 text-center">
            {wsError ? (
              <div className="flex flex-col items-center gap-3 py-6 text-gray-400 text-xs">
                <p className="text-red-400">{wsError}</p>
                <button
                  type="button"
                  onClick={connectWebSocket}
                  className="px-4 py-2 rounded-xl bg-surfaceHover hover:bg-white/10 text-white text-xs transition-colors border border-white/10"
                >
                  Переподключиться
                </button>
              </div>
            ) : sessionUrl ? (
              <QRDisplay url={sessionUrl} size={220} />
            ) : (
              <div className="w-[244px] h-[244px] flex items-center justify-center bg-white/5 rounded-2xl animate-pulse">
                <div className="w-8 h-8 rounded-full border-3 border-accent border-t-transparent animate-spin" />
              </div>
            )}

            <p className="text-xs text-gray-400 max-w-xs leading-relaxed">
              Наведите камеру смартфона, чтобы открыть ссылку на этом экране
            </p>
          </div>
        )}
      </main>

      <Footer />

      {/* Result Modal with 15s timeout */}
      {modalData && (
        <ResultModal
          isOpen={modalData.isOpen}
          title={modalData.title}
          url={modalData.url}
          shortUrl={modalData.shortUrl}
          onClose={() => setModalData(null)}
        />
      )}
    </div>
  )
}
