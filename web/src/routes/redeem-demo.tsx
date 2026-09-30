import { createFileRoute } from '@tanstack/react-router'
import { useMemo, useState } from 'react'

export const Route = createFileRoute('/redeem-demo')({ component: RedeemDemo })

const MONTHLY_LIMIT = 500
const CATEGORIES = [
  { id: 'cash', label: '现金兑换码', amount: 100000000 },
  { id: 'gift', label: '礼品兑换码', amount: 1000 },
  { id: 'bonus', label: '奖金兑换码', amount: 5000 },
]

function RedeemDemo() {
  const [category, setCategory] = useState(CATEGORIES[0].id)
  const [redeemed, setRedeemed] = useState(0)
  const [message, setMessage] = useState<{ title: string; body: string } | null>(null)
  const remaining = Math.max(0, MONTHLY_LIMIT - redeemed)
  const refreshDate = useMemo(() => {
    const date = new Date()
    date.setMonth(date.getMonth() + 1, 1)
    return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
  }, [])

  function redeem() {
    const selected = CATEGORIES.find((item) => item.id === category) ?? CATEGORIES[0]
    const available = Math.min(selected.amount, remaining)
    setRedeemed((value) => value + available)
    setMessage({
      title: available < selected.amount ? '本次兑换已按额度处理' : '兑换成功',
      body: `${selected.label}面额 $${selected.amount.toLocaleString()}，本次扣除 $${available.toFixed(2)}。下次刷新时间：${refreshDate} 00:00。`,
    })
  }

  return (
    <main className="min-h-svh bg-[#f5f7fb] px-4 py-10 text-slate-900">
      <div className="mx-auto max-w-2xl">
        <div className="mb-6 flex items-center justify-between">
          <div>
            <p className="text-sm font-semibold tracking-[0.2em] text-indigo-600">VAULT · REDEEM</p>
            <h1 className="mt-2 text-3xl font-bold tracking-tight">兑换中心</h1>
          </div>
          <span className="rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-700">演示模式</span>
        </div>

        <section className="rounded-3xl bg-white p-6 shadow-xl shadow-slate-200/60 sm:p-8">
          <div className="rounded-2xl bg-gradient-to-br from-indigo-600 to-violet-600 p-6 text-white">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-indigo-100">可兑换余额</p>
                <p className="mt-2 text-4xl font-bold tabular-nums">$100,000,000.00</p>
              </div>
              <span className="rounded-lg bg-white/15 px-2 py-1 text-xs">USD</span>
            </div>
            <div className="mt-6 grid grid-cols-2 gap-4 border-t border-white/20 pt-4 text-sm">
              <div><p className="text-indigo-100">本月最高可兑换</p><p className="mt-1 text-xl font-semibold">$500.00</p></div>
              <div><p className="text-indigo-100">本月剩余额度</p><p className="mt-1 text-xl font-semibold">${remaining.toFixed(2)}</p></div>
            </div>
          </div>

          <div className="mt-6">
            <label htmlFor="category" className="text-sm font-medium text-slate-700">选择兑换码类别</label>
            <div className="mt-2 flex gap-3">
              <select id="category" value={category} onChange={(event) => setCategory(event.target.value)} className="flex-1 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-lg outline-none focus:border-indigo-500">{CATEGORIES.map((item) => <option key={item.id} value={item.id}>{item.label} · ${item.amount.toLocaleString()}</option>)}</select>
              <button type="button" onClick={redeem} disabled={remaining === 0} className="rounded-xl bg-indigo-600 px-5 py-3 font-semibold text-white shadow-lg shadow-indigo-200 transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:bg-slate-300">立即兑换</button>
            </div>
            {message && <div role="alert" className="mt-3 rounded-xl border border-indigo-100 bg-indigo-50 px-4 py-3 text-sm text-indigo-700"><p className="font-semibold">{message.title}</p><p className="mt-1">{message.body}</p><button type="button" onClick={() => setMessage(null)} className="mt-2 text-xs underline">知道了</button></div>}
          </div>

          <div className="mt-6 flex items-center justify-between rounded-xl bg-slate-50 px-4 py-3 text-sm">
            <span className="text-slate-500">下次额度刷新</span>
            <span className="font-semibold text-slate-800">{refreshDate  } 00:00</span>
          </div>
          <p className="mt-5 text-center text-xs text-slate-400">本页面仅供朋友间娱乐演示，不构成真实兑换承诺。</p>
        </section>
      </div>
    </main>
  )
}
