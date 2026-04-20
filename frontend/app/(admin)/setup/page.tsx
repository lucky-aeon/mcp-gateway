'use client'

import { SetupPage } from '@/components/pages/setup-page'
import { useAppStore } from '@/lib/store'
import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

export default function Page() {
  const { currentUser } = useAppStore()
  const router = useRouter()

  useEffect(() => {
    if (!currentUser || currentUser.builtin !== true) {
      router.replace('/dashboard')
    }
  }, [currentUser, router])

  if (!currentUser || currentUser.builtin !== true) {
    return null
  }

  return <SetupPage />
}
