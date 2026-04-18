"use client"

import Link from "next/link"
import { ArrowRight, Check, HelpCircle, Zap } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { useState } from "react"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

const plans = [
  {
    name: "Free",
    description: "适合个人开发者和小型项目",
    price: { monthly: 0, yearly: 0 },
    priceLabel: "永久免费",
    features: [
      { text: "每月 5,000 次 API 调用", included: true },
      { text: "最多 2 个工作空间", included: true },
      { text: "最多 5 个 MCP 集成", included: true },
      { text: "社区支持", included: true },
      { text: "基础监控", included: true },
      { text: "优先支持", included: false },
      { text: "自定义域名", included: false },
      { text: "SLA 保障", included: false },
    ],
    cta: "免费开始",
    popular: false,
  },
  {
    name: "Pro",
    description: "适合成长中的团队和专业开发者",
    price: { monthly: 29, yearly: 24 },
    priceLabel: "/月",
    features: [
      { text: "每月 100,000 次 API 调用", included: true },
      { text: "无限工作空间", included: true },
      { text: "无限 MCP 集成", included: true },
      { text: "邮件支持", included: true },
      { text: "高级监控与分析", included: true },
      { text: "优先支持", included: true },
      { text: "自定义域名", included: true },
      { text: "SLA 保障", included: false },
    ],
    cta: "升级到 Pro",
    popular: true,
  },
  {
    name: "Team",
    description: "适合需要协作的团队",
    price: { monthly: 79, yearly: 66 },
    priceLabel: "/月",
    features: [
      { text: "每月 500,000 次 API 调用", included: true },
      { text: "无限工作空间", included: true },
      { text: "无限 MCP 集成", included: true },
      { text: "专属客户经理", included: true },
      { text: "高级监控与分析", included: true },
      { text: "优先支持", included: true },
      { text: "自定义域名", included: true },
      { text: "99.9% SLA 保障", included: true },
    ],
    cta: "开始 Team 计划",
    popular: false,
  },
]

const enterprisePlan = {
  name: "Enterprise",
  description: "适合大型企业，需要更高安全性和定制化服务",
  features: [
    "无限 API 调用",
    "私有化部署",
    "SAML SSO 单点登录",
    "专属技术支持",
    "99.99% SLA 保障",
    "定制化培训",
  ],
}

const faqs = [
  {
    question: "什么是 API 调用次数？",
    answer: "API 调用次数是指您通过 MCP Gateway 发送的请求总数。每次工具调用、会话创建或数据查询都计为一次调用。"
  },
  {
    question: "可以随时升级或降级吗？",
    answer: "是的，您可以随时升级或降级您的计划。升级会立即生效，降级将在当前计费周期结束后生效。"
  },
  {
    question: "是否支持退款？",
    answer: "我们提供 14 天无理由退款保证。如果您对服务不满意，可以在购买后 14 天内申请全额退款。"
  },
  {
    question: "企业版有什么特别之处？",
    answer: "企业版提供私有化部署、专属技术支持、SAML SSO、定制化 SLA 等高级功能，满足大型企业的安全合规需求。"
  },
]

export default function PricingPage() {
  const [billingCycle, setBillingCycle] = useState<"monthly" | "yearly">("monthly")

  return (
    <TooltipProvider>
      <div className="bg-background">
        {/* Hero */}
        <section className="pt-20 pb-16">
          <div className="mx-auto max-w-7xl px-6 text-center">
            {/* Announcement */}
            <div className="flex justify-center">
              <div className="inline-flex items-center gap-2 rounded-full border border-border bg-muted/50 px-4 py-1.5 text-sm">
                <span className="rounded-full bg-accent px-2 py-0.5 text-xs font-medium text-accent-foreground">
                  新功能
                </span>
                <span className="text-muted-foreground">基于用量的灵活计费</span>
              </div>
            </div>

            <h1 className="mt-8 text-4xl font-bold tracking-tight md:text-5xl">
              方案与定价
            </h1>
            <p className="mx-auto mt-4 max-w-2xl text-lg text-muted-foreground">
              立即免费开始使用。升级以获得更多调用量、更强功能和团队协作支持。
            </p>

            {/* Billing Toggle */}
            <div className="mt-10 flex items-center justify-center gap-4">
              <button
                onClick={() => setBillingCycle("monthly")}
                className={cn(
                  "text-sm font-medium transition-colors",
                  billingCycle === "monthly" ? "text-foreground" : "text-muted-foreground"
                )}
              >
                月付
              </button>
              <button
                onClick={() => setBillingCycle(billingCycle === "monthly" ? "yearly" : "monthly")}
                className={cn(
                  "relative h-6 w-11 rounded-full transition-colors",
                  billingCycle === "yearly" ? "bg-accent" : "bg-muted"
                )}
              >
                <span
                  className={cn(
                    "absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow-sm transition-transform",
                    billingCycle === "yearly" && "translate-x-5"
                  )}
                />
              </button>
              <button
                onClick={() => setBillingCycle("yearly")}
                className={cn(
                  "flex items-center gap-2 text-sm font-medium transition-colors",
                  billingCycle === "yearly" ? "text-foreground" : "text-muted-foreground"
                )}
              >
                年付
                <span className="rounded-full bg-accent/10 px-2 py-0.5 text-xs font-medium text-accent">
                  省 17%
                </span>
              </button>
            </div>
          </div>
        </section>

        {/* Pricing Cards */}
        <section className="pb-16">
          <div className="mx-auto max-w-7xl px-6">
            <div className="grid gap-6 md:grid-cols-3">
              {plans.map((plan) => (
                <div
                  key={plan.name}
                  className={cn(
                    "relative flex flex-col rounded-2xl border p-8 transition-all",
                    plan.popular
                      ? "border-accent bg-card shadow-lg shadow-accent/10"
                      : "border-border bg-card hover:border-accent/50"
                  )}
                >
                  {plan.popular && (
                    <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                      <span className="inline-flex items-center gap-1 rounded-full bg-accent px-3 py-1 text-xs font-medium text-accent-foreground">
                        <Zap className="h-3 w-3" />
                        推荐
                      </span>
                    </div>
                  )}

                  <div>
                    <h3 className="text-xl font-semibold">{plan.name}</h3>
                    <p className="mt-2 text-sm text-muted-foreground">{plan.description}</p>
                  </div>

                  <div className="mt-6">
                    <div className="flex items-baseline gap-1">
                      {plan.price.monthly === 0 ? (
                        <span className="text-4xl font-bold">免费</span>
                      ) : (
                        <>
                          <span className="text-4xl font-bold">
                            ${billingCycle === "monthly" ? plan.price.monthly : plan.price.yearly}
                          </span>
                          <span className="text-muted-foreground">{plan.priceLabel}</span>
                        </>
                      )}
                    </div>
                    {plan.price.monthly > 0 && billingCycle === "yearly" && (
                      <p className="mt-1 text-sm text-muted-foreground">
                        按年计费，共 ${plan.price.yearly * 12}/年
                      </p>
                    )}
                  </div>

                  <ul className="mt-8 flex-1 space-y-3">
                    {plan.features.map((feature) => (
                      <li key={feature.text} className="flex items-start gap-3">
                        <div className={cn(
                          "mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full",
                          feature.included 
                            ? "bg-accent/10 text-accent" 
                            : "bg-muted text-muted-foreground"
                        )}>
                          <Check className="h-3 w-3" />
                        </div>
                        <span className={cn(
                          "text-sm",
                          !feature.included && "text-muted-foreground"
                        )}>
                          {feature.text}
                        </span>
                      </li>
                    ))}
                  </ul>

                  <div className="mt-8">
                    <Link href="/login">
                      <Button
                        className={cn(
                          "w-full",
                          plan.popular && "bg-accent text-accent-foreground hover:bg-accent/90"
                        )}
                        variant={plan.popular ? "default" : "outline"}
                      >
                        {plan.cta}
                      </Button>
                    </Link>
                  </div>
                </div>
              ))}
            </div>

            {/* Enterprise */}
            <div className="mt-8 rounded-2xl border border-border bg-card p-8">
              <div className="flex flex-col items-center justify-between gap-6 md:flex-row">
                <div className="text-center md:text-left">
                  <h3 className="text-xl font-semibold">{enterprisePlan.name}</h3>
                  <p className="mt-2 text-muted-foreground">{enterprisePlan.description}</p>
                  <div className="mt-4 flex flex-wrap justify-center gap-4 md:justify-start">
                    {enterprisePlan.features.map((feature) => (
                      <div key={feature} className="flex items-center gap-2 text-sm">
                        <Check className="h-4 w-4 text-accent" />
                        <span>{feature}</span>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="shrink-0">
                  <Button size="lg" variant="outline" className="gap-2">
                    联系销售
                    <ArrowRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* FAQ */}
        <section className="border-t border-border bg-muted/30 py-24">
          <div className="mx-auto max-w-3xl px-6">
            <h2 className="text-center text-3xl font-bold">常见问题</h2>
            <div className="mt-12 space-y-6">
              {faqs.map((faq) => (
                <div
                  key={faq.question}
                  className="rounded-xl border border-border bg-card p-6"
                >
                  <h3 className="flex items-center gap-2 font-semibold">
                    <HelpCircle className="h-5 w-5 text-accent" />
                    {faq.question}
                  </h3>
                  <p className="mt-3 text-muted-foreground">{faq.answer}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* CTA */}
        <section className="py-24">
          <div className="mx-auto max-w-7xl px-6 text-center">
            <h2 className="text-3xl font-bold">准备好开始了吗？</h2>
            <p className="mx-auto mt-4 max-w-xl text-muted-foreground">
              立即注册，获得免费额度，体验 MCP Gateway 的强大功能
            </p>
            <div className="mt-10">
              <Link href="/login">
                <Button size="lg" className="h-12 px-8 text-base gap-2">
                  免费开始
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </Link>
            </div>
          </div>
        </section>

      </div>
    </TooltipProvider>
  )
}
