'use client'

import { createContext, useContext, useEffect, useState } from 'react'

type Theme = 'light' | 'dark' | 'system'

interface ThemeProviderState {
  theme: Theme
  setTheme: (theme: Theme) => void
  compactMode: boolean
  setCompactMode: (compact: boolean) => void
}

const ThemeProviderContext = createContext<ThemeProviderState | undefined>(undefined)

interface ThemeProviderProps {
  children: React.ReactNode
  defaultTheme?: Theme
  storageKey?: string
}

export function ThemeProvider({
  children,
  defaultTheme = 'system',
  storageKey = 'gateway-admin-theme',
}: ThemeProviderProps) {
  const [theme, setTheme] = useState<Theme>(defaultTheme)
  const [compactMode, setCompactMode] = useState(false)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
    const savedTheme = localStorage.getItem(storageKey) as Theme | null
    const savedCompact = localStorage.getItem(`${storageKey}-compact`)
    
    if (savedTheme) {
      setTheme(savedTheme)
    }
    if (savedCompact) {
      setCompactMode(savedCompact === 'true')
    }
  }, [storageKey])

  useEffect(() => {
    if (!mounted) return

    const root = window.document.documentElement

    root.classList.remove('light', 'dark')

    if (theme === 'system') {
      const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark'
        : 'light'
      root.classList.add(systemTheme)
    } else {
      root.classList.add(theme)
    }
  }, [theme, mounted])

  useEffect(() => {
    if (!mounted) return

    const root = window.document.documentElement
    if (compactMode) {
      root.classList.add('compact')
    } else {
      root.classList.remove('compact')
    }
  }, [compactMode, mounted])

  const handleSetTheme = (newTheme: Theme) => {
    localStorage.setItem(storageKey, newTheme)
    setTheme(newTheme)
  }

  const handleSetCompactMode = (compact: boolean) => {
    localStorage.setItem(`${storageKey}-compact`, String(compact))
    setCompactMode(compact)
  }

  if (!mounted) {
    return null
  }

  return (
    <ThemeProviderContext.Provider
      value={{
        theme,
        setTheme: handleSetTheme,
        compactMode,
        setCompactMode: handleSetCompactMode,
      }}
    >
      {children}
    </ThemeProviderContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeProviderContext)
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}
