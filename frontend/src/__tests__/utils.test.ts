import { describe, expect, it } from 'bun:test'
import { isMobileDevice } from '../utils/device'

function isValidUrl(raw: string): boolean {
  try {
    const u = new URL(raw)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

function extractToken(path: string): string | null {
  const m = path.match(/^\/session\/(.+)$/)
  return m ? m[1] : null
}

describe('Frontend Utils', () => {
  describe('isValidUrl', () => {
    it('accepts valid http and https urls', () => {
      expect(isValidUrl('http://example.com')).toBe(true)
      expect(isValidUrl('https://short.steins.ru/abc')).toBe(true)
      expect(isValidUrl('https://google.com/search?q=test#hash')).toBe(true)
    })

    it('rejects non-http protocols and malformed strings', () => {
      expect(isValidUrl('ftp://ftp.example.com')).toBe(false)
      expect(isValidUrl('javascript:alert(1)')).toBe(false)
      expect(isValidUrl('random-text')).toBe(false)
      expect(isValidUrl('')).toBe(false)
    })
  })

  describe('extractToken', () => {
    it('extracts token correctly from /session/:token', () => {
      expect(extractToken('/session/test-token-123')).toBe('test-token-123')
      expect(extractToken('/session/uuid-v4-abc')).toBe('uuid-v4-abc')
    })

    it('returns null for other paths', () => {
      expect(extractToken('/')).toBeNull()
      expect(extractToken('/abc123')).toBeNull()
      expect(extractToken('/api/shorten')).toBeNull()
    })
  })

  describe('isMobileDevice', () => {
    it('detects iPhone 12 user agent and screen size', () => {
      const iPhoneUA =
        'Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1'
      expect(
        isMobileDevice({
          userAgent: iPhoneUA,
          width: 390,
          touchPoints: 5,
        }),
      ).toBe(true)
    })

    it('detects Android phone user agent', () => {
      const androidUA =
        'Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36'
      expect(
        isMobileDevice({
          userAgent: androidUA,
          width: 412,
          touchPoints: 5,
        }),
      ).toBe(true)
    })

    it('detects desktop browser on 1920x1080 screen', () => {
      const desktopUA =
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
      expect(
        isMobileDevice({
          userAgent: desktopUA,
          width: 1920,
          touchPoints: 0,
        }),
      ).toBe(false)
    })

    it('detects narrow desktop browser window (<768px)', () => {
      const desktopUA =
        'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
      expect(
        isMobileDevice({
          userAgent: desktopUA,
          width: 600,
          touchPoints: 0,
        }),
      ).toBe(true)
    })

    it('returns false when window is undefined and no options provided (SSR safe)', () => {
      expect(isMobileDevice()).toBe(false)
    })
  })
})
