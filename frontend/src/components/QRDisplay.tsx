import { useEffect, useRef } from 'react'
import QRCode from 'qrcode'

interface QRDisplayProps {
  url: string
  size?: number
}

export default function QRDisplay({ url, size = 260 }: QRDisplayProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    if (!canvasRef.current) return
    QRCode.toCanvas(canvasRef.current, url, {
      width: size,
      margin: 1,
      color: {
        dark: '#0f1117',
        light: '#ffffff',
      },
    }).catch((err: unknown) => {
      console.error('Failed to render QR code:', err)
    })
  }, [url, size])

  return (
    <div className="p-3 bg-white rounded-2xl shadow-xl inline-flex items-center justify-center">
      <canvas ref={canvasRef} />
    </div>
  )
}
