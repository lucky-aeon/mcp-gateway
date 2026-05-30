import { useLocation, useNavigate } from 'react-router-dom'

export function usePathname() {
  return useLocation().pathname
}

export function useRouter() {
  const navigate = useNavigate()

  return {
    push: (path: string) => navigate(path),
    replace: (path: string) => navigate(path, { replace: true }),
  }
}
