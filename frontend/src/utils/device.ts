export interface DeviceOptions {
  userAgent?: string
  width?: number
  touchPoints?: number
}

/**
 * Detects whether the current client is running on a mobile device or small screen.
 * Evaluates user agent, touch support, and screen width / media queries.
 */
export function isMobileDevice(opts?: DeviceOptions): boolean {
  if (typeof window === 'undefined' && !opts) return false

  const ua =
    opts?.userAgent ??
    (typeof navigator !== 'undefined'
      ? navigator.userAgent || navigator.vendor || ''
      : '')
  const isMobileUA =
    /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(ua)

  const touch =
    opts?.touchPoints ??
    (typeof navigator !== 'undefined' ? navigator.maxTouchPoints || 0 : 0)
  const isTouch =
    touch > 0 || (typeof window !== 'undefined' && 'ontouchstart' in window)

  const width =
    opts?.width ?? (typeof window !== 'undefined' ? window.innerWidth : 1024)
  const isSmallScreen =
    (typeof window !== 'undefined' &&
      typeof window.matchMedia === 'function' &&
      window.matchMedia('(max-width: 767px)').matches) ||
    width < 768

  return isMobileUA || (isTouch && isSmallScreen) || isSmallScreen
}
