import {
  DEFAULT_TABLE_PAGE_SIZE,
  getConfiguredTableDefaultPageSize,
  normalizeTablePageSize
} from '@/utils/tablePreferences'

const STORAGE_KEY = 'table-page-size'
const STORAGE_SOURCE_KEY = 'table-page-size-source'
const STORAGE_DEFAULT_KEY = 'table-page-size-default'

export function getPersistedPageSize(fallback = getConfiguredTableDefaultPageSize()): number {
  const configuredDefault = getConfiguredTableDefaultPageSize()
  if (typeof window !== 'undefined') {
    try {
      const stored = window.localStorage.getItem(STORAGE_KEY)
      if (stored !== null) {
        const parsed = Number(stored)
        if (Number.isFinite(parsed)) {
          if (configuredDefault !== DEFAULT_TABLE_PAGE_SIZE) {
            const source = window.localStorage.getItem(STORAGE_SOURCE_KEY)
            const storedDefault = Number(window.localStorage.getItem(STORAGE_DEFAULT_KEY))
            if (source === 'user' && Number.isFinite(storedDefault) && storedDefault === configuredDefault) {
              return normalizeTablePageSize(parsed)
            }
            return normalizeTablePageSize(configuredDefault)
          }
          return normalizeTablePageSize(parsed)
        }
      }
    } catch (error) {
      console.warn('Failed to read persisted page size:', error)
    }
  }
  return normalizeTablePageSize(configuredDefault || fallback)
}

export function setPersistedPageSize(size: number): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, String(normalizeTablePageSize(size)))
    window.localStorage.setItem(STORAGE_SOURCE_KEY, 'user')
    window.localStorage.setItem(STORAGE_DEFAULT_KEY, String(getConfiguredTableDefaultPageSize()))
  } catch (error) {
    console.warn('Failed to persist page size:', error)
  }
}
