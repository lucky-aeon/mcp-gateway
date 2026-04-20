'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Loader2, KeyRound, Server, Shield, Zap } from 'lucide-react'
import { gatewayApi, GatewayApiError, hasGatewayAuth, saveGatewayApiKey, saveGatewayRefreshToken } from '@/lib/gateway-api'
import { useAppStore } from '@/lib/store'

export default function LoginPage() {
  const router = useRouter()
  const { setCurrentUser } = useAppStore()
  const [isLoading, setIsLoading] = useState(false)
  const [metaLoading, setMetaLoading] = useState(true)
  const [mode, setMode] = useState<'single-key' | 'saas'>('single-key')
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login')
  const [allowRegister, setAllowRegister] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [rememberMe, setRememberMe] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false

    async function bootstrap() {
      try {
        const meta = await gatewayApi.getMeta()
        if (cancelled) return
        setMode(meta.mode === 'saas' ? 'saas' : 'single-key')
        setAllowRegister(meta.allow_register || false)

        if (hasGatewayAuth()) {
          try {
            await gatewayApi.getMe()
            router.replace('/dashboard')
            return
          } catch {
            // 忽略旧 token / api key，留在登录页重新输入
          }
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : '无法获取登录模式')
        }
      } finally {
        if (!cancelled) setMetaLoading(false)
      }
    }

    bootstrap()
    return () => {
      cancelled = true
    }
  }, [router])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      if (authMode === 'register') {
        await gatewayApi.register({ email, password, display_name: displayName || undefined })
        // 注册成功后切换到登录模式
        setAuthMode('login')
        setError('')
        setIsLoading(false)
        return
      }

      let result
      if (mode === 'single-key') {
        result = await gatewayApi.login({ api_key: apiKey })
        saveGatewayApiKey(result.token || apiKey, rememberMe)
      } else {
        result = await gatewayApi.login({ email, password })
        saveGatewayApiKey(result.token, rememberMe)
        if (result.refresh_token) {
          saveGatewayRefreshToken(result.refresh_token, rememberMe)
        }
      }
      // 设置用户信息到 store
      setCurrentUser(result.user)
      router.push('/dashboard')
    } catch (e) {
      if (e instanceof GatewayApiError) {
        setError(e.message)
      } else {
        setError(authMode === 'register' ? '注册失败，请稍后重试' : '登录失败，请稍后重试')
      }
    } finally {
      setIsLoading(false)
    }
  }

  const toggleAuthMode = () => {
    setAuthMode(authMode === 'login' ? 'register' : 'login')
    setError('')
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
            <CardTitle className="text-2xl">
              {mode === 'single-key'
                ? '接入控制台'
                : authMode === 'register'
                  ? '创建账户'
                  : '欢迎回来'}
            </CardTitle>
            <CardDescription>
              {mode === 'single-key'
                ? '当前实例为 single-key 模式，请输入 API Key 访问管理控制台'
                : authMode === 'register'
                  ? '注册新账户以访问管理控制台'
                  : '登录您的账户以访问管理控制台'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {metaLoading ? (
              <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                正在获取登录模式...
              </div>
            ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="rounded-lg bg-destructive/10 px-4 py-3 text-sm text-destructive">
                  {error}
                </div>
              )}

              {mode === 'single-key' ? (
                <div className="space-y-2">
                  <Label htmlFor="api-key">API Key</Label>
                  <div className="relative">
                    <KeyRound className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                    <Input
                      id="api-key"
                      type="password"
                      placeholder="输入 Gateway API Key"
                      value={apiKey}
                      onChange={(e) => setApiKey(e.target.value)}
                      required
                      autoComplete="off"
                      className="h-11 pl-9"
                    />
                  </div>
                </div>
              ) : (
                <>
                  {authMode === 'register' && (
                    <div className="space-y-2">
                      <Label htmlFor="display-name">显示名称</Label>
                      <Input
                        id="display-name"
                        type="text"
                        placeholder="输入显示名称"
                        value={displayName}
                        onChange={(e) => setDisplayName(e.target.value)}
                        autoComplete="name"
                        className="h-11"
                      />
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
                      autoComplete={authMode === 'register' ? 'email' : 'email'}
                      className="h-11"
                    />
                  </div>

                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <Label htmlFor="password">密码</Label>
                      {authMode === 'login' && (
                        <Button variant="link" className="h-auto p-0 text-sm text-muted-foreground">
                          忘记密码？
                        </Button>
                      )}
                    </div>
                    <Input
                      id="password"
                      type="password"
                      placeholder={authMode === 'register' ? '设置密码' : '输入密码'}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      required
                      autoComplete={authMode === 'register' ? 'new-password' : 'current-password'}
                      className="h-11"
                    />
                  </div>
                </>
              )}

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
                    {authMode === 'register' ? '注册中...' : '登录中...'}
                  </>
                ) : (
                  authMode === 'register' ? '注册' : '登录'
                )}
              </Button>

              {mode === 'saas' && allowRegister && (
                <p className="text-center text-sm">
                  {authMode === 'login' ? (
                    <>
                      还没有账户？{' '}
                      <Button variant="link" className="h-auto p-0 text-sm" onClick={toggleAuthMode}>
                        立即注册
                      </Button>
                    </>
                  ) : (
                    <>
                      已有账户？{' '}
                      <Button variant="link" className="h-auto p-0 text-sm" onClick={toggleAuthMode}>
                        立即登录
                      </Button>
                    </>
                  )}
                </p>
              )}

              {mode === 'saas' && !allowRegister && authMode === 'login' ? (
                <p className="text-center text-sm text-muted-foreground">
                  演示账号：admin@gateway.local / admin123456
                </p>
              ) : mode === 'single-key' ? (
                <p className="text-center text-sm text-muted-foreground">
                  API Key 由 Gateway 管理员分发，登录成功后才会请求受保护接口
                </p>
              ) : null}
            </form>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
