'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Loader2, Server, Shield, Zap } from 'lucide-react'

export default function LoginPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [rememberMe, setRememberMe] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    // Simulate login
    await new Promise((resolve) => setTimeout(resolve, 1000))

    if (email === 'admin@gateway.local' && password === 'admin') {
      localStorage.setItem('isLoggedIn', 'true')
      router.push('/dashboard')
    } else {
      setError('邮箱或密码错误')
      setIsLoading(false)
    }
  }

  const features = [
    { icon: Server, title: '统一管理', desc: '集中管理所有 MCP 服务' },
    { icon: Shield, title: '安全可控', desc: '细粒度的权限和访问控制' },
    { icon: Zap, title: '高效运维', desc: '实时监控与日志分析' },
  ]

  return (
    <div className="min-h-screen bg-background flex">
      {/* Left Panel - Branding */}
      <div className="hidden lg:flex lg:w-1/2 bg-gradient-to-br from-primary/10 via-primary/5 to-background relative overflow-hidden">
        <div className="absolute inset-0 bg-grid-white/5" />
        <div className="relative z-10 flex flex-col justify-between p-12 w-full">
          <div>
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary">
                <Server className="h-5 w-5 text-primary-foreground" />
              </div>
              <span className="text-xl font-bold">Gateway Admin</span>
            </div>
          </div>

          <div className="space-y-8">
            <div>
              <h1 className="text-4xl font-bold tracking-tight text-balance">
                MCP Gateway 管理控制台
              </h1>
              <p className="mt-4 text-lg text-muted-foreground text-pretty max-w-md">
                统一管理和监控您的 MCP 服务，提供安全、高效的工具调用网关
              </p>
            </div>

            <div className="grid gap-4">
              {features.map((feature) => (
                <div key={feature.title} className="flex items-start gap-4 p-4 rounded-xl bg-card/50 backdrop-blur border border-border/50">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10">
                    <feature.icon className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <h3 className="font-semibold">{feature.title}</h3>
                    <p className="text-sm text-muted-foreground">{feature.desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <p className="text-sm text-muted-foreground">
            MCP Gateway v1.0.0
          </p>
        </div>
      </div>

      {/* Right Panel - Login Form */}
      <div className="flex flex-1 items-center justify-center p-8">
        <Card className="w-full max-w-md border-0 shadow-none lg:border lg:shadow-sm">
          <CardHeader className="space-y-1 pb-6">
            <div className="flex items-center gap-3 lg:hidden mb-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary">
                <Server className="h-5 w-5 text-primary-foreground" />
              </div>
              <span className="text-xl font-bold">Gateway Admin</span>
            </div>
            <CardTitle className="text-2xl">欢迎回来</CardTitle>
            <CardDescription>
              登录您的账户以访问管理控制台
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="rounded-lg bg-destructive/10 px-4 py-3 text-sm text-destructive">
                  {error}
                </div>
              )}

              <div className="space-y-2">
                <Label htmlFor="email">邮箱</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="admin@gateway.local"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoComplete="email"
                  className="h-11"
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="password">密码</Label>
                  <Button variant="link" className="h-auto p-0 text-sm text-muted-foreground">
                    忘记密码？
                  </Button>
                </div>
                <Input
                  id="password"
                  type="password"
                  placeholder="输入密码"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoComplete="current-password"
                  className="h-11"
                />
              </div>

              <div className="flex items-center gap-2">
                <Checkbox 
                  id="remember" 
                  checked={rememberMe}
                  onCheckedChange={(checked) => setRememberMe(checked as boolean)}
                />
                <Label htmlFor="remember" className="text-sm font-normal cursor-pointer">
                  记住我
                </Label>
              </div>

              <Button type="submit" className="w-full h-11" disabled={isLoading}>
                {isLoading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    登录中...
                  </>
                ) : (
                  '登录'
                )}
              </Button>

              <p className="text-center text-sm text-muted-foreground">
                演示账号：admin@gateway.local / admin
              </p>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
