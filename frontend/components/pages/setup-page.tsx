'use client'

import { useEffect, useState } from 'react'
import { Check, Database, Save, Server, Shield, Bell, Monitor, Moon, Settings, Sun } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { gatewayApi, useGatewaySWR, type SystemConfig } from '@/lib/gateway-api'
import { useTheme } from '@/components/providers/theme-provider'
import { cn } from '@/lib/utils'

export function SetupPage() {
  const { data } = useGatewaySWR<SystemConfig>('/api/v1/system/config')
  const [saved, setSaved] = useState(false)
  const [form, setForm] = useState<SystemConfig | null>(null)
  const { theme, setTheme, compactMode, setCompactMode } = useTheme()

  useEffect(() => {
    if (data) setForm(data)
  }, [data])

  async function handleSave() {
    if (!form) return
    await gatewayApi.updateSystemConfig(form)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  if (!form) {
    return <div className="text-sm text-muted-foreground">加载配置中...</div>
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">设置</h2>
        <p className="text-muted-foreground">管理 Gateway 实例的核心系统配置</p>
      </div>

      <Tabs defaultValue="general" className="space-y-6">
        <TabsList className="inline-flex h-auto gap-1 rounded-xl bg-muted p-1">
          <TabsTrigger value="general" className="gap-2 rounded-lg px-4 py-2"><Settings className="h-4 w-4" />常规</TabsTrigger>
          <TabsTrigger value="server" className="gap-2 rounded-lg px-4 py-2"><Server className="h-4 w-4" />服务器</TabsTrigger>
          <TabsTrigger value="security" className="gap-2 rounded-lg px-4 py-2"><Shield className="h-4 w-4" />安全</TabsTrigger>
          <TabsTrigger value="notifications" className="gap-2 rounded-lg px-4 py-2"><Bell className="h-4 w-4" />通知</TabsTrigger>
          <TabsTrigger value="advanced" className="gap-2 rounded-lg px-4 py-2"><Database className="h-4 w-4" />高级</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>基本信息</CardTitle>
              <CardDescription>Gateway 实例的基本配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="instance-name">实例名称</Label>
                  <Input id="instance-name" defaultValue="My Gateway" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="timezone">时区</Label>
                  <Select defaultValue="Asia/Shanghai">
                    <SelectTrigger id="timezone"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Asia/Shanghai">Asia/Shanghai (UTC+8)</SelectItem>
                      <SelectItem value="America/New_York">America/New_York (UTC-5)</SelectItem>
                      <SelectItem value="Europe/London">Europe/London (UTC+0)</SelectItem>
                      <SelectItem value="Asia/Tokyo">Asia/Tokyo (UTC+9)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="instance-desc">描述</Label>
                <Textarea id="instance-desc" defaultValue="MCP Gateway 管理控制台" rows={3} />
              </div>
              <div className="space-y-3">
                <Label>主题模式</Label>
                <div className="grid grid-cols-3 gap-3">
                  {[
                    { id: 'light', label: '浅色', icon: Sun },
                    { id: 'dark', label: '深色', icon: Moon },
                    { id: 'system', label: '跟随系统', icon: Monitor },
                  ].map((item) => (
                    <button
                      key={item.id}
                      onClick={() => setTheme(item.id as 'light' | 'dark' | 'system')}
                      className={cn(
                        'flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:bg-muted/50',
                        theme === item.id ? 'border-primary bg-primary/5' : 'border-transparent bg-muted/30'
                      )}
                    >
                      <div className={cn('flex h-10 w-10 items-center justify-center rounded-lg', theme === item.id ? 'bg-primary text-primary-foreground' : 'bg-muted')}>
                        <item.icon className="h-5 w-5" />
                      </div>
                      <span className="text-sm font-medium">{item.label}</span>
                    </button>
                  ))}
                </div>
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">紧凑模式</Label>
                  <p className="text-sm text-muted-foreground">减少界面间距，显示更多内容</p>
                </div>
                <Switch checked={compactMode} onCheckedChange={setCompactMode} />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="server" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>服务器配置</CardTitle>
              <CardDescription>Gateway 服务器的运行配置</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-6 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="bind">绑定地址</Label>
                <Input id="bind" value={form.bind} onChange={(e) => setForm((v) => v ? { ...v, bind: e.target.value } : v)} />
              </div>
              <div className="space-y-2">
                <Label>网关协议</Label>
                <Select value={form.gateway_protocol} onValueChange={(value: 'sse' | 'streamhttp') => setForm((v) => v ? { ...v, gateway_protocol: value } : v)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="sse">SSE</SelectItem>
                    <SelectItem value="streamhttp">Streamable HTTP</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="gc">Session GC 间隔（秒）</Label>
                <Input id="gc" type="number" value={form.session_gc_interval_seconds} onChange={(e) => setForm((v) => v ? { ...v, session_gc_interval_seconds: Number(e.target.value) } : v)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="timeout">Session 超时（秒）</Label>
                <Input id="timeout" type="number" value={form.proxy_session_timeout_seconds} onChange={(e) => setForm((v) => v ? { ...v, proxy_session_timeout_seconds: Number(e.target.value) } : v)} />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>认证设置</CardTitle>
              <CardDescription>API 认证和授权配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">启用 API 密钥认证</Label>
                  <p className="text-sm text-muted-foreground">要求所有请求携带有效的 API 密钥</p>
                </div>
                <Switch checked={form.auth.enabled} onCheckedChange={(checked) => setForm((v) => v ? { ...v, auth: { ...v.auth, enabled: checked } } : v)} />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="notifications" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>通知偏好</CardTitle>
              <CardDescription>为后续通知能力预留的页面配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">服务异常提醒</Label>
                  <p className="text-sm text-muted-foreground">MCP 启动失败或异常停止时提醒</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">会话阈值提醒</Label>
                  <p className="text-sm text-muted-foreground">会话数过高时提醒</p>
                </div>
                <Switch />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="advanced" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>高级参数</CardTitle>
              <CardDescription>更细粒度的运行时参数</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="retry">MCP 重试次数</Label>
                <Input id="retry" type="number" value={form.mcp_retry_count} onChange={(e) => setForm((v) => v ? { ...v, mcp_retry_count: Number(e.target.value) } : v)} />
              </div>
              <div className="rounded-xl border bg-muted/30 p-4 text-sm text-muted-foreground">
                当前模式：{form.auth.mode}，注册能力：{form.auth.allow_register ? '开启' : '关闭'}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Button onClick={handleSave}>
        {saved ? <Check className="mr-2 h-4 w-4" /> : <Save className="mr-2 h-4 w-4" />}
        {saved ? '已保存' : '保存配置'}
      </Button>
    </div>
  )
}
