"use client"

import { useState } from "react"
import Link from "next/link"
import { ArrowRight, Zap, Shield, Globe, Code, Lock, ChevronRight, Check, Play, ChevronDown } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const features = [
  {
    icon: Globe,
    title: "统一接入",
    description: "一个网关连接所有 MCP 服务，简化集成复杂度"
  },
  {
    icon: Shield,
    title: "企业级安全",
    description: "内置 API 密钥管理、访问控制和审计日志"
  },
  {
    icon: Zap,
    title: "高性能",
    description: "毫秒级响应，支持百万级并发连接"
  },
  {
    icon: Code,
    title: "开发友好",
    description: "完整的 SDK 支持，丰富的开发者工具"
  }
]

const stats = [
  { value: "99.99%", label: "服务可用性" },
  { value: "<10ms", label: "平均延迟" },
  { value: "1M+", label: "日活连接" },
  { value: "500+", label: "MCP 集成" }
]

const useCases = [
  {
    title: "AI 应用开发",
    description: "快速集成各类 AI 模型和工具，构建智能应用",
    gradient: "from-emerald-500/20 to-teal-500/20"
  },
  {
    title: "企业自动化",
    description: "连接企业内部系统，实现业务流程自动化",
    gradient: "from-blue-500/20 to-cyan-500/20"
  },
  {
    title: "数据分析",
    description: "统一数据源接入，支持实时数据分析与可视化",
    gradient: "from-violet-500/20 to-purple-500/20"
  }
]

export default function HomePage() {
  const [isCodeExpanded, setIsCodeExpanded] = useState(false)

  return (
    <div className="bg-background">
      {/* Hero Section */}
      <section className="relative overflow-hidden pt-20 pb-20">
        {/* Background gradient */}
        <div className="absolute inset-0 -z-10">
          <div className="absolute top-0 left-1/4 h-[500px] w-[500px] rounded-full bg-accent/20 blur-[120px]" />
          <div className="absolute bottom-0 right-1/4 h-[400px] w-[400px] rounded-full bg-accent/10 blur-[100px]" />
        </div>
        
        <div className="mx-auto max-w-7xl px-6">
          {/* Announcement badge */}
          <div className="flex justify-center">
            <Link 
              href="#" 
              className="group inline-flex items-center gap-2 rounded-full border border-border bg-background/50 px-4 py-1.5 text-sm backdrop-blur-sm transition-colors hover:border-accent"
            >
              <span className="rounded-full bg-accent px-2 py-0.5 text-xs font-medium text-accent-foreground">
                新版本
              </span>
              <span className="text-muted-foreground">Gateway v2.0 正式发布</span>
              <ChevronRight className="h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
            </Link>
          </div>

          {/* Hero content */}
          <div className="mt-10 text-center">
            <h1 className="mx-auto max-w-4xl text-balance text-5xl font-bold tracking-tight md:text-6xl lg:text-7xl">
              构建 AI 应用的
              <span className="relative">
                <span className="relative z-10 text-accent"> 统一网关</span>
              </span>
            </h1>
            <p className="mx-auto mt-6 max-w-2xl text-balance text-lg text-muted-foreground md:text-xl">
              MCP Gateway 为您提供安全、高效、可扩展的 MCP 协议网关服务。
              一站式管理所有 AI 工具集成，让开发更简单。
            </p>
            
            {/* CTA buttons */}
            <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
              <Link href="/login">
                <Button size="lg" className="h-12 px-8 text-base gap-2">
                  免费开始使用
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </Link>
              <Button variant="outline" size="lg" className="h-12 px-8 text-base gap-2">
                <Play className="h-4 w-4" />
                观看演示
              </Button>
            </div>
          </div>

          {/* Stats */}
          <div className="mt-20 grid grid-cols-2 gap-8 md:grid-cols-4">
            {stats.map((stat) => (
              <div key={stat.label} className="text-center">
                <div className="text-3xl font-bold md:text-4xl">{stat.value}</div>
                <div className="mt-1 text-sm text-muted-foreground">{stat.label}</div>
              </div>
            ))}
          </div>

          {/* Code preview */}
          <div className="mt-20">
            <div className="mx-auto max-w-3xl overflow-hidden rounded-xl border border-border bg-card shadow-2xl shadow-black/5">
              <div className="flex items-center gap-2 border-b border-border bg-muted/50 px-4 py-3">
                <div className="flex gap-1.5">
                  <div className="h-3 w-3 rounded-full bg-red-500/80" />
                  <div className="h-3 w-3 rounded-full bg-yellow-500/80" />
                  <div className="h-3 w-3 rounded-full bg-green-500/80" />
                </div>
                <span className="ml-2 text-xs text-muted-foreground">terminal</span>
              </div>
              <div 
                className={cn(
                  "overflow-hidden transition-all duration-300 ease-in-out",
                  isCodeExpanded ? "max-h-none" : "max-h-[225px]"
                )}
              >
                <div className="p-6">
                  <pre className="text-sm leading-relaxed">
                    <code className="text-muted-foreground">
                      <span className="text-muted-foreground/60"># 1. 初始化会话，绑定 Workspace 并获取 Session ID</span>{"\n"}
                      <span className="text-accent">$</span> curl -X POST https://gateway.example.com/stream \{"\n"}
                      {"  "}-H <span className="text-green-400">{'"Authorization: Bearer $API_KEY"'}</span> \{"\n"}
                      {"  "}-H <span className="text-green-400">{'"Content-Type: application/json"'}</span> \{"\n"}
                      {"  "}-d <span className="text-green-400">{`'{"jsonrpc":"2.0","id":1,"method":"initialize",`}</span>{"\n"}
                      {"      "}<span className="text-green-400">{`"params":{"protocolVersion":"2025-03-26",`}</span>{"\n"}
                      {"      "}<span className="text-green-400">{`"capabilities":{},`}</span>{"\n"}
                      {"      "}<span className="text-green-400">{`"clientInfo":{"name":"my-client","version":"1.0.0"}}}'`}</span>{"\n\n"}
                      <span className="text-muted-foreground/60"># 响应头包含 Mcp-Session-Id: 7782f2f9-563c-4379-b961-df06e49e54c0</span>{"\n\n"}
                      <span className="text-muted-foreground/60"># 2. 完成握手</span>{"\n"}
                      <span className="text-accent">$</span> curl -X POST https://gateway.example.com/stream \{"\n"}
                      {"  "}-H <span className="text-green-400">{'"Authorization: Bearer $API_KEY"'}</span> \{"\n"}
                      {"  "}-H <span className="text-green-400">{'"Content-Type: application/json"'}</span> \{"\n"}
                      {"  "}-H <span className="text-green-400">{'"Mcp-Session-Id: $SESSION_ID"'}</span> \{"\n"}
                      {"  "}-d <span className="text-green-400">{`'{"jsonrpc":"2.0","method":"notifications/initialized"}'`}</span>{"\n\n"}
                      <span className="text-muted-foreground/60"># 3. 调用 Workspace 下的 MCP 工具</span>{"\n"}
                      <span className="text-accent">$</span> curl -X POST https://gateway.example.com/stream \{"\n"}
                      {"  "}-H <span className="text-green-400">{'"Authorization: Bearer $API_KEY"'}</span> \{"\n"}
                      {"  "}-H <span className="text-green-400">{'"Content-Type: application/json"'}</span> \{"\n"}
                      {"  "}-H <span className="text-green-400">{'"Mcp-Session-Id: $SESSION_ID"'}</span> \{"\n"}
                      {"  "}-d <span className="text-green-400">{`'{"jsonrpc":"2.0","id":2,"method":"tools/call",`}</span>{"\n"}
                      {"      "}<span className="text-green-400">{`"params":{"name":"time_get_current_time",`}</span>{"\n"}
                      {"      "}<span className="text-green-400">{`"arguments":{"timezone":"Asia/Shanghai"}}}'`}</span>{"\n\n"}
                      <span className="text-muted-foreground/60">{"{"}</span>{"\n"}
                      {"  "}<span className="text-blue-400">{'"jsonrpc"'}</span>: <span className="text-green-400">{'"2.0"'}</span>,{"\n"}
                      {"  "}<span className="text-blue-400">{'"id"'}</span>: <span className="text-yellow-400">2</span>,{"\n"}
                      {"  "}<span className="text-blue-400">{'"result"'}</span>: {"{"}{"\n"}
                      {"    "}<span className="text-blue-400">{'"content"'}</span>: [{"\n"}
                      {"      {"}<span className="text-blue-400">{'"type"'}</span>: <span className="text-green-400">{'"text"'}</span>, <span className="text-blue-400">{'"text"'}</span>: <span className="text-green-400">{'"2025-04-19 12:34:56+08:00"'}</span>{"}"}{"\n"}
                      {"    "}]{"\n"}
                      {"  "}{"}"}{"\n"}
                      <span className="text-muted-foreground/60">{"}"}</span>
                    </code>
                  </pre>
                </div>
              </div>
              <button
                onClick={() => setIsCodeExpanded(!isCodeExpanded)}
                className="flex w-full items-center justify-center gap-2 border-t border-border bg-muted/50 px-4 py-3 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              >
                {isCodeExpanded ? "收起" : "查看全部"}
                <ChevronDown className={cn("h-4 w-4 transition-transform", isCodeExpanded && "rotate-180")} />
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="border-t border-border bg-muted/30 py-24">
        <div className="mx-auto max-w-7xl px-6">
          <div className="text-center">
            <h2 className="text-3xl font-bold md:text-4xl">为什么选择 MCP Gateway</h2>
            <p className="mx-auto mt-4 max-w-2xl text-muted-foreground">
              我们提供企业级的 MCP 协议网关解决方案，帮助您快速构建可靠的 AI 应用
            </p>
          </div>

          <div className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-4">
            {features.map((feature) => (
              <div
                key={feature.title}
                className="group rounded-xl border border-border bg-card p-6 transition-all hover:border-accent/50 hover:shadow-lg"
              >
                <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-accent/10 text-accent transition-colors group-hover:bg-accent group-hover:text-accent-foreground">
                  <feature.icon className="h-6 w-6" />
                </div>
                <h3 className="mt-4 text-lg font-semibold">{feature.title}</h3>
                <p className="mt-2 text-sm text-muted-foreground">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Use Cases Section */}
      <section id="use-cases" className="py-24">
        <div className="mx-auto max-w-7xl px-6">
          <div className="text-center">
            <h2 className="text-3xl font-bold md:text-4xl">使用场景</h2>
            <p className="mx-auto mt-4 max-w-2xl text-muted-foreground">
              无论您是初创团队还是大型企业，MCP Gateway 都能满足您的需求
            </p>
          </div>

          <div className="mt-16 grid gap-6 md:grid-cols-3">
            {useCases.map((useCase) => (
              <div
                key={useCase.title}
                className={cn(
                  "group relative overflow-hidden rounded-2xl border border-border p-8 transition-all hover:border-accent/50",
                  "bg-gradient-to-br",
                  useCase.gradient
                )}
              >
                <h3 className="text-xl font-semibold">{useCase.title}</h3>
                <p className="mt-3 text-muted-foreground">{useCase.description}</p>
                <Link 
                  href="#" 
                  className="mt-6 inline-flex items-center gap-1 text-sm font-medium text-accent transition-colors hover:text-accent/80"
                >
                  了解更多
                  <ArrowRight className="h-4 w-4" />
                </Link>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="border-t border-border bg-muted/30 py-24">
        <div className="mx-auto max-w-7xl px-6 text-center">
          <h2 className="text-3xl font-bold md:text-4xl">准备好开始了吗？</h2>
          <p className="mx-auto mt-4 max-w-xl text-muted-foreground">
            立即注册，获得免费额度，体验 MCP Gateway 的强大功能
          </p>
          <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
            <Link href="/login">
              <Button size="lg" className="h-12 px-8 text-base gap-2">
                免费开始
                <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
            <Link href="/pricing">
              <Button variant="outline" size="lg" className="h-12 px-8 text-base">
                查看定价
              </Button>
            </Link>
          </div>
        </div>
      </section>

    </div>
  )
}
