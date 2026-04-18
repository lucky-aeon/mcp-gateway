'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Settings,
  Server,
  Shield,
  Bell,
  Database,
  Save,
  Check,
  Moon,
  Sun,
  Monitor,
} from 'lucide-react'
import { useTheme } from '@/components/providers/theme-provider'
import { cn } from '@/lib/utils'

export function SetupPage() {
  const [saved, setSaved] = useState(false)
  const { theme, setTheme, compactMode, setCompactMode } = useTheme()

  const handleSave = () => {
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">设置</h2>
        <p className="text-muted-foreground">管理系统配置和偏好设置</p>
      </div>

      <Tabs defaultValue="general" className="space-y-6">
        <TabsList className="inline-flex h-auto gap-1 rounded-xl bg-muted p-1">
          <TabsTrigger 
            value="general" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <Settings className="h-4 w-4" />
            常规
          </TabsTrigger>
          <TabsTrigger 
            value="server" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <Server className="h-4 w-4" />
            服务器
          </TabsTrigger>
          <TabsTrigger 
            value="security" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <Shield className="h-4 w-4" />
            安全
          </TabsTrigger>
          <TabsTrigger 
            value="notifications" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <Bell className="h-4 w-4" />
            通知
          </TabsTrigger>
          <TabsTrigger 
            value="advanced" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <Database className="h-4 w-4" />
            高级
          </TabsTrigger>
        </TabsList>

        {/* General Settings */}
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
                    <SelectTrigger id="timezone">
                      <SelectValue />
                    </SelectTrigger>
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
                <Textarea
                  id="instance-desc"
                  defaultValue="MCP Gateway 管理控制台"
                  rows={3}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="language">语言</Label>
                <Select defaultValue="zh-CN">
                  <SelectTrigger id="language" className="w-full sm:w-[240px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="zh-CN">简体中文</SelectItem>
                    <SelectItem value="en-US">English (US)</SelectItem>
                    <SelectItem value="ja-JP">日本語</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>外观设置</CardTitle>
              <CardDescription>自定义界面外观</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Theme Selection */}
              <div className="space-y-3">
                <Label>主题模式</Label>
                <div className="grid grid-cols-3 gap-3">
                  <button
                    onClick={() => setTheme('light')}
                    className={cn(
                      'flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:bg-muted/50',
                      theme === 'light' ? 'border-primary bg-primary/5' : 'border-transparent bg-muted/30'
                    )}
                  >
                    <div className={cn(
                      'flex h-10 w-10 items-center justify-center rounded-lg',
                      theme === 'light' ? 'bg-primary text-primary-foreground' : 'bg-muted'
                    )}>
                      <Sun className="h-5 w-5" />
                    </div>
                    <span className="text-sm font-medium">浅色</span>
                  </button>
                  <button
                    onClick={() => setTheme('dark')}
                    className={cn(
                      'flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:bg-muted/50',
                      theme === 'dark' ? 'border-primary bg-primary/5' : 'border-transparent bg-muted/30'
                    )}
                  >
                    <div className={cn(
                      'flex h-10 w-10 items-center justify-center rounded-lg',
                      theme === 'dark' ? 'bg-primary text-primary-foreground' : 'bg-muted'
                    )}>
                      <Moon className="h-5 w-5" />
                    </div>
                    <span className="text-sm font-medium">深色</span>
                  </button>
                  <button
                    onClick={() => setTheme('system')}
                    className={cn(
                      'flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:bg-muted/50',
                      theme === 'system' ? 'border-primary bg-primary/5' : 'border-transparent bg-muted/30'
                    )}
                  >
                    <div className={cn(
                      'flex h-10 w-10 items-center justify-center rounded-lg',
                      theme === 'system' ? 'bg-primary text-primary-foreground' : 'bg-muted'
                    )}>
                      <Monitor className="h-5 w-5" />
                    </div>
                    <span className="text-sm font-medium">跟随系统</span>
                  </button>
                </div>
              </div>

              {/* Compact Mode */}
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">紧凑模式</Label>
                  <p className="text-sm text-muted-foreground">减少界面间距，显示更多内容</p>
                </div>
                <Switch 
                  checked={compactMode} 
                  onCheckedChange={setCompactMode}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Server Settings */}
        <TabsContent value="server" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>服务器配置</CardTitle>
              <CardDescription>Gateway 服务器的运行配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="server-port">监听端口</Label>
                  <Input id="server-port" type="number" defaultValue="8080" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="server-host">绑定地址</Label>
                  <Input id="server-host" defaultValue="0.0.0.0" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="max-connections">最大连接数</Label>
                  <Input id="max-connections" type="number" defaultValue="1000" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="request-timeout">请求超时（秒）</Label>
                  <Input id="request-timeout" type="number" defaultValue="30" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>日志配置</CardTitle>
              <CardDescription>日志记录和存储设置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="log-level">日志级别</Label>
                  <Select defaultValue="info">
                    <SelectTrigger id="log-level">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="debug">Debug</SelectItem>
                      <SelectItem value="info">Info</SelectItem>
                      <SelectItem value="warn">Warning</SelectItem>
                      <SelectItem value="error">Error</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="log-retention">日志保留天数</Label>
                  <Input id="log-retention" type="number" defaultValue="30" />
                </div>
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">启用访问日志</Label>
                  <p className="text-sm text-muted-foreground">记录所有 API 访问请求</p>
                </div>
                <Switch defaultChecked />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Security Settings */}
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
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">启用速率限制</Label>
                  <p className="text-sm text-muted-foreground">限制每个客户端的请求频率</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="space-y-2">
                <Label htmlFor="rate-limit">速率限制（请求/分钟）</Label>
                <Input id="rate-limit" type="number" defaultValue="60" className="w-full sm:w-[240px]" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>CORS 设置</CardTitle>
              <CardDescription>跨域资源共享配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">启用 CORS</Label>
                  <p className="text-sm text-muted-foreground">允许跨域请求</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="space-y-2">
                <Label htmlFor="cors-origins">允许的来源</Label>
                <Textarea
                  id="cors-origins"
                  placeholder="每行一个，例如：https://example.com"
                  rows={3}
                  defaultValue="*"
                />
                <p className="text-xs text-muted-foreground">使用 * 允许所有来源</p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Notification Settings */}
        <TabsContent value="notifications" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>通知偏好</CardTitle>
              <CardDescription>配置系统通知</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">邮件通知</Label>
                  <p className="text-sm text-muted-foreground">接收重要事件的邮件通知</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">Webhook 通知</Label>
                  <p className="text-sm text-muted-foreground">将事件发送到自定义 Webhook</p>
                </div>
                <Switch />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>告警规则</CardTitle>
              <CardDescription>配置系统告警触发条件</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">MCP 错误告警</Label>
                  <p className="text-sm text-muted-foreground">当 MCP 发生错误时发送告警</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">高延迟告警</Label>
                  <p className="text-sm text-muted-foreground">当响应延迟超过阈值时告警</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="space-y-2">
                <Label htmlFor="latency-threshold">延迟阈值（毫秒）</Label>
                <Input id="latency-threshold" type="number" defaultValue="5000" className="w-full sm:w-[240px]" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Advanced Settings */}
        <TabsContent value="advanced" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>数据存储</CardTitle>
              <CardDescription>数据库和缓存配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-2">
                <Label htmlFor="db-url">数据库连接</Label>
                <Input
                  id="db-url"
                  type="password"
                  defaultValue="postgresql://..."
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="redis-url">Redis 连接（可选）</Label>
                <Input id="redis-url" placeholder="redis://localhost:6379" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>MCP 运行时</CardTitle>
              <CardDescription>MCP 服务器运行配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="mcp-timeout">MCP 超时（秒）</Label>
                  <Input id="mcp-timeout" type="number" defaultValue="30" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="mcp-retries">重试次数</Label>
                  <Input id="mcp-retries" type="number" defaultValue="3" />
                </div>
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">启用沙箱模式</Label>
                  <p className="text-sm text-muted-foreground">在隔离环境中运行 MCP</p>
                </div>
                <Switch defaultChecked />
              </div>
            </CardContent>
          </Card>

          <Card className="border-destructive/50">
            <CardHeader>
              <CardTitle className="text-destructive">危险操作</CardTitle>
              <CardDescription>这些操作可能导致数据丢失，请谨慎操作</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-xl border border-destructive/20 bg-destructive/5 p-4">
                <div className="space-y-0.5">
                  <p className="font-medium">清除所有日志</p>
                  <p className="text-sm text-muted-foreground">删除所有系统日志记录</p>
                </div>
                <Button variant="destructive" size="sm">清除</Button>
              </div>
              <div className="flex items-center justify-between rounded-xl border border-destructive/20 bg-destructive/5 p-4">
                <div className="space-y-0.5">
                  <p className="font-medium">重置所有设置</p>
                  <p className="text-sm text-muted-foreground">将所有设置恢复为默认值</p>
                </div>
                <Button variant="destructive" size="sm">重置</Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Save Button */}
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={saved} className="min-w-[120px]">
          {saved ? (
            <>
              <Check className="mr-2 h-4 w-4" />
              已保存
            </>
          ) : (
            <>
              <Save className="mr-2 h-4 w-4" />
              保存更改
            </>
          )}
        </Button>
      </div>
    </div>
  )
}
