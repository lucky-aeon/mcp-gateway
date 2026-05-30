'use client'

import * as React from 'react'
import { ThemeProvider as AppThemeProvider } from '@/components/providers/theme-provider'

type ThemeProviderProps = React.ComponentProps<typeof AppThemeProvider>

export function ThemeProvider({ children, ...props }: ThemeProviderProps) {
  return <AppThemeProvider {...props}>{children}</AppThemeProvider>
}
