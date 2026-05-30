import { Link as RouterLink, type LinkProps as RouterLinkProps } from 'react-router-dom'

type LinkProps = Omit<RouterLinkProps, 'to'> & {
  href: RouterLinkProps['to']
}

export default function Link({ href, ...props }: LinkProps) {
  return <RouterLink to={href} {...props} />
}
