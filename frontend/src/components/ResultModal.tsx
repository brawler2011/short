import { useEffect, useState } from 'react'

interface ResultModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  url: string
  shortUrl?: string
}

export default function ResultModal({
  isOpen,
  onClose,
  title,
  url,
  shortUrl,
}: ResultModalProps) {
  const [remainingMs, setRemainingMs] = useState(15000)
  const [copied, setCopied] = useState(false)
  const copyTarget = shortUrl || url

  useEffect(() => {
    if (!isOpen) return

    setRemainingMs(15000)
    setCopied(false)

    const interval = 100
    const timer = setInterval(() => {
      setRemainingMs((prev) => {
        if (prev <= interval) {
          clearInterval(timer)
          onClose()
          return 0
        }
        return prev - interval
      })
    }, interval)

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)

    return () => {
      clearInterval(timer)
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [isOpen, onClose])

  if (!isOpen) return null

  const handleCopy = async () => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(copyTarget)
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = copyTarget
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

  const secondsLeft = Math.ceil(remainingMs / 1000)
  const progressPercent = (remainingMs / 15000) * 100

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="relative w-full max-w-md bg-surface border border-white/10 rounded-3xl p-6 sm:p-7 shadow-2xl overflow-hidden flex flex-col gap-5 text-left"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Auto-closing progress bar */}
        <div className="absolute top-0 left-0 right-0 h-1 bg-white/5">
          <div
            className="h-full bg-red-500 transition-all duration-100 ease-linear"
            style={{ width: `${progressPercent}%` }}
          />
        </div>

        {/* Modal Header */}
        <div className="flex items-center justify-between pt-1">
          <h3 className="text-lg font-semibold text-white tracking-tight">
            {title}
          </h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Закрыть"
            className="p-1.5 -mr-1.5 text-gray-400 hover:text-white rounded-lg hover:bg-white/10 transition-colors"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* URL box with copy icon */}
        <div className="flex items-center gap-2 p-3 bg-bg rounded-xl border border-white/10">
          <div className="flex-1 min-w-0 font-mono text-xs text-gray-300 break-all select-all">
            {copyTarget}
          </div>
          <button
            type="button"
            onClick={handleCopy}
            title={copied ? 'Скопировано!' : 'Скопировать ссылку'}
            aria-label="Скопировать ссылку"
            className={`p-2 rounded-lg transition-colors flex items-center justify-center shrink-0 ${
              copied
                ? 'bg-emerald-500/20 text-emerald-400'
                : 'text-gray-400 hover:text-white hover:bg-white/10'
            }`}
          >
            {copied ? (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2.5}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            ) : (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <rect x="9" y="9" width="13" height="13" rx="2" strokeWidth={2} />
                <path
                  d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"
                  strokeWidth={2}
                />
              </svg>
            )}
          </button>
        </div>

        {/* Big Red Button "Перейти" in the center */}
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          onClick={onClose}
          className="w-full py-4 px-6 rounded-2xl bg-red-600 hover:bg-red-500 active:bg-red-700 text-white font-bold text-lg text-center transition-all shadow-lg shadow-red-600/30 flex items-center justify-center gap-2 tracking-wide"
        >
          <span>Перейти</span>
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2.5}
              d="M14 5l7 7m0 0l-7 7m7-7H3"
            />
          </svg>
        </a>

        {/* Countdown footer */}
        <div className="flex items-center justify-between text-xs text-gray-500 pt-1">
          <span>Закроется через {secondsLeft} сек.</span>
          <button
            type="button"
            onClick={onClose}
            className="text-gray-400 hover:text-gray-200 transition-colors"
          >
            Закрыть
          </button>
        </div>
      </div>
    </div>
  )
}
