/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { Crown, CalendarClock, Package } from 'lucide-react'
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { GroupBadge } from '@/components/group-badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { useSystemConfig } from '@/hooks/use-system-config'
import { formatQuota } from '@/lib/format'
import { DEFAULT_CURRENCY_CONFIG } from '@/stores/system-config-store'

import {
  paySubscriptionStripe,
  paySubscriptionCreem,
  paySubscriptionEpay,
  paySubscriptionHupijiao,
  paySubscriptionWaffoPancake,
  paySubscriptionBalance,
  getSubscriptionHupijiaoPaymentStatus,
} from '../../api'
import { formatDuration, formatResetPeriod } from '../../lib'
import type { PlanRecord } from '../../types'

interface PaymentMethod {
  type: string
  name?: string
}

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  plan: PlanRecord | null
  enableStripe?: boolean
  enableCreem?: boolean
  enableWaffoPancake?: boolean
  enableOnlineTopUp?: boolean
  onlinePaymentProvider?: 'epay' | 'hupijiao'
  epayMethods?: PaymentMethod[]
  purchaseLimit?: number
  purchaseCount?: number
  userQuota?: number
  onPurchaseSuccess?: () => void | Promise<void>
}

const PENDING_SUBSCRIPTION_PAYMENT_KEY = 'new-api:pending-hupijiao-subscription'
const PENDING_SUBSCRIPTION_PAYMENT_MAX_AGE = 30 * 60 * 1000

interface PendingSubscriptionPayment {
  tradeNo: string
  qrcodeUrl: string
  redirectUrl: string
  paymentMethod: string
  actualAmount: number
  planId: number
  createdAt: number
}

function readPendingSubscriptionPayment(): PendingSubscriptionPayment | null {
  if (typeof window === 'undefined') return null
  try {
    const parsed = JSON.parse(
      window.sessionStorage.getItem(PENDING_SUBSCRIPTION_PAYMENT_KEY) || 'null'
    ) as PendingSubscriptionPayment | null
    if (
      !parsed?.tradeNo ||
      Date.now() - parsed.createdAt > PENDING_SUBSCRIPTION_PAYMENT_MAX_AGE
    ) {
      window.sessionStorage.removeItem(PENDING_SUBSCRIPTION_PAYMENT_KEY)
      return null
    }
    return parsed
  } catch {
    window.sessionStorage.removeItem(PENDING_SUBSCRIPTION_PAYMENT_KEY)
    return null
  }
}

function getPaymentRequestError(error: unknown, fallback: string): string {
  if (!error || typeof error !== 'object') return fallback
  const candidate = error as {
    message?: string
    response?: { data?: { message?: string } }
  }
  return candidate.response?.data?.message || candidate.message || fallback
}

function isPaymentSuccess(response: { success?: boolean; message?: string }) {
  return response.success === true || response.message === 'success'
}

export function SubscriptionPurchaseDialog(props: Props) {
  const { t } = useTranslation()
  const { currency } = useSystemConfig()
  const { onOpenChange, onPurchaseSuccess } = props
  const [paying, setPaying] = useState(false)
  const [selectedEpayMethod, setSelectedEpayMethod] = useState('')
  const [pendingHupijiaoPayment, setPendingHupijiaoPayment] =
    useState<PendingSubscriptionPayment | null>(readPendingSubscriptionPayment)

  useEffect(() => {
    if (props.open && props.epayMethods && props.epayMethods.length > 0) {
      setSelectedEpayMethod(
        props.epayMethods.find((method) => method.type === 'alipay')?.type ||
          props.epayMethods[0].type
      )
    } else if (!props.open) {
      setSelectedEpayMethod('')
    }
  }, [props.open, props.epayMethods])

  useEffect(() => {
    const tradeNo = pendingHupijiaoPayment?.tradeNo
    if (!tradeNo) return
    let active = true
    let finished = false
    let timer: number | undefined

    const clearPending = () => {
      setPendingHupijiaoPayment(null)
      window.sessionStorage.removeItem(PENDING_SUBSCRIPTION_PAYMENT_KEY)
    }
    const poll = async () => {
      try {
        if (
          Date.now() - pendingHupijiaoPayment.createdAt >
          PENDING_SUBSCRIPTION_PAYMENT_MAX_AGE
        ) {
          finished = true
          clearPending()
          toast.error(t('Payment order expired or failed.'))
          return
        }
        const response = await getSubscriptionHupijiaoPaymentStatus(tradeNo)
        if (!active) return
        if (!response.success || !response.data) {
          finished = true
          clearPending()
          toast.error(response.message || t('Payment order expired or failed.'))
          return
        }
        if (response.data.status === 'success') {
          finished = true
          clearPending()
          toast.success(t('Payment successful. Subscription activated.'))
          await onPurchaseSuccess?.()
          onOpenChange(false)
          return
        }
        if (['failed', 'expired'].includes(response.data.status)) {
          finished = true
          clearPending()
          toast.error(t('Payment order expired or failed.'))
        }
      } catch {
        // Keep polling through temporary network failures.
      } finally {
        if (active && !finished) timer = window.setTimeout(poll, 2000)
      }
    }

    timer = window.setTimeout(poll, 1200)
    return () => {
      active = false
      if (timer) window.clearTimeout(timer)
    }
  }, [
    pendingHupijiaoPayment?.tradeNo,
    pendingHupijiaoPayment?.createdAt,
    onOpenChange,
    onPurchaseSuccess,
    t,
  ])

  const plan = props.plan?.plan
  if (!plan) return null

  const hasStripe = props.enableStripe && !!plan.stripe_price_id
  const hasCreem = props.enableCreem && !!plan.creem_product_id
  const hasWaffoPancake =
    props.enableWaffoPancake && !!plan.waffo_pancake_product_id
  const hasEpay =
    props.enableOnlineTopUp && (props.epayMethods || []).length > 0
  const hasAnyPayment = hasStripe || hasCreem || hasWaffoPancake || hasEpay
  const hasHupijiao = hasEpay && props.onlinePaymentProvider === 'hupijiao'
  const selectedEpayMethodLabel =
    (props.epayMethods || []).find((m) => m.type === selectedEpayMethod)
      ?.name ||
    selectedEpayMethod ||
    t('Select payment method')
  const totalAmount = Number(plan.total_amount || 0)
  const price = Number(plan.price_amount || 0).toFixed(2)
  const hupijiaoPrice = (
    Number(plan.price_amount || 0) * Number(plan.hupijiao_discount_rate || 1)
  ).toFixed(2)
  // Subscription checkout is settled in RMB by the configured online gateway.
  const subscriptionSymbol = '¥'
  const quotaPerUnit =
    currency?.quotaPerUnit && currency.quotaPerUnit > 0
      ? currency.quotaPerUnit
      : DEFAULT_CURRENCY_CONFIG.quotaPerUnit
  const balanceCost = Math.max(
    0,
    Math.ceil(Number(plan.price_amount || 0) * quotaPerUnit)
  )
  const userQuota = Math.max(0, Number(props.userQuota || 0))
  const allowBalancePay = plan.allow_balance_pay !== false
  const insufficientBalance = userQuota < balanceCost
  const limitReached =
    (props.purchaseLimit || 0) > 0 &&
    (props.purchaseCount || 0) >= (props.purchaseLimit || 0)

  const handlePayStripe = async () => {
    setPaying(true)
    try {
      const res = await paySubscriptionStripe({ plan_id: plan.id })
      if (isPaymentSuccess(res) && res.data?.pay_link) {
        window.open(res.data.pay_link, '_blank')
        toast.success(t('Payment page opened'))
        props.onOpenChange(false)
      } else {
        toast.error(
          res.message && res.message !== 'success'
            ? res.message
            : t('Payment request failed')
        )
      }
    } catch (error) {
      toast.error(getPaymentRequestError(error, t('Payment request failed')))
    } finally {
      setPaying(false)
    }
  }

  const handlePayCreem = async () => {
    setPaying(true)
    try {
      const res = await paySubscriptionCreem({ plan_id: plan.id })
      if (isPaymentSuccess(res) && res.data?.checkout_url) {
        window.open(res.data.checkout_url, '_blank')
        toast.success(t('Payment page opened'))
        props.onOpenChange(false)
      } else {
        toast.error(
          res.message && res.message !== 'success'
            ? res.message
            : t('Payment request failed')
        )
      }
    } catch {
      toast.error(t('Payment request failed'))
    } finally {
      setPaying(false)
    }
  }

  // In-tab redirect (not window.open) — user-gesture context is lost
  // across the await, so a popup would be blocked. Same as the wallet hook.
  const handlePayWaffoPancake = async () => {
    setPaying(true)
    try {
      const res = await paySubscriptionWaffoPancake({ plan_id: plan.id })
      if (isPaymentSuccess(res) && res.data?.checkout_url) {
        toast.success(t('Redirecting to payment page...'))
        window.location.href = res.data.checkout_url
      } else {
        toast.error(
          res.message && res.message !== 'success'
            ? res.message
            : t('Payment request failed')
        )
      }
    } catch {
      toast.error(t('Payment request failed'))
    } finally {
      setPaying(false)
    }
  }

  const isSafari =
    typeof navigator !== 'undefined' &&
    /^((?!chrome|android).)*safari/i.test(navigator.userAgent)

  const handlePayEpay = async () => {
    if (!selectedEpayMethod) {
      toast.error(t('Please select a payment method'))
      return
    }
    setPaying(true)
    try {
      const res = await (hasHupijiao
        ? paySubscriptionHupijiao({
            plan_id: plan.id,
            payment_method: selectedEpayMethod,
          })
        : paySubscriptionEpay({
            plan_id: plan.id,
            payment_method: selectedEpayMethod,
          }))
      const redirectUrl = res.data?.redirect_url
      const qrcodeUrl = res.data?.qrcode_url
      const tradeNo = res.data?.trade_no
      if (isPaymentSuccess(res) && (redirectUrl || qrcodeUrl)) {
        if (hasHupijiao && !tradeNo) {
          toast.error(t('Payment request failed'))
          return
        }
        if (hasHupijiao && tradeNo) {
          const pending = {
            tradeNo,
            qrcodeUrl: qrcodeUrl || '',
            redirectUrl: redirectUrl || '',
            paymentMethod: selectedEpayMethod,
            actualAmount: Number(res.data?.actual_amount || hupijiaoPrice),
            planId: plan.id,
            createdAt: Date.now(),
          }
          setPendingHupijiaoPayment(pending)
          window.sessionStorage.setItem(
            PENDING_SUBSCRIPTION_PAYMENT_KEY,
            JSON.stringify(pending)
          )
        }
        const mobile = /Mobile|Android|iPhone/i.test(navigator.userAgent)
        if (mobile || !qrcodeUrl) {
          if (!redirectUrl) {
            toast.error(t('Payment QR code was not returned'))
            return
          }
          window.location.href = redirectUrl
          props.onOpenChange(false)
        } else {
          // The pending payment state keeps the QR image available even if
          // the dialog is closed and reopened before the callback arrives.
        }
        return
      }
      if (isPaymentSuccess(res) && res.url) {
        const form = document.createElement('form')
        form.action = res.url
        form.method = 'POST'
        if (!isSafari) {
          form.target = '_blank'
        }
        Object.entries(res.data || {}).forEach(([key, value]) => {
          const input = document.createElement('input')
          input.type = 'hidden'
          input.name = key
          input.value = String(value)
          form.appendChild(input)
        })
        document.body.appendChild(form)
        form.submit()
        document.body.removeChild(form)
        toast.success(t('Payment initiated'))
        props.onOpenChange(false)
      } else {
        toast.error(
          res.message && res.message !== 'success'
            ? res.message
            : t('Payment request failed')
        )
      }
    } catch (error) {
      toast.error(getPaymentRequestError(error, t('Payment request failed')))
    } finally {
      setPaying(false)
    }
  }

  const handlePayBalance = async () => {
    if (!allowBalancePay) {
      toast.error(t('This plan does not allow balance redemption'))
      return
    }
    setPaying(true)
    try {
      const res = await paySubscriptionBalance({ plan_id: plan.id })
      if (res.success) {
        toast.success(t('Subscription purchased successfully'))
        void props.onPurchaseSuccess?.()
        props.onOpenChange(false)
      } else {
        toast.error(
          res.message && res.message !== 'success'
            ? res.message
            : t('Payment request failed')
        )
      }
    } catch {
      toast.error(t('Payment request failed'))
    } finally {
      setPaying(false)
    }
  }

  const visiblePendingPayment =
    pendingHupijiaoPayment?.planId === plan.id ? pendingHupijiaoPayment : null
  const visiblePendingPaymentMethodLabel =
    (props.epayMethods || []).find(
      (method) => method.type === visiblePendingPayment?.paymentMethod
    )?.name || selectedEpayMethodLabel

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={
        <>
          <Crown className='h-5 w-5' />
          {t('Purchase Subscription')}
        </>
      }
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'
      titleClassName='flex items-center gap-2'
      contentHeight='auto'
      bodyClassName='space-y-4'
    >
      <div className='space-y-3 sm:space-y-4'>
        {visiblePendingPayment?.qrcodeUrl && (
          <div className='flex flex-col items-center gap-3 rounded-md border p-4'>
            <div className='text-center'>
              <div className='text-muted-foreground text-sm'>
                {t('Amount Due')}
              </div>
              <div className='text-3xl font-semibold'>
                ¥
                {Number(
                  visiblePendingPayment.actualAmount || hupijiaoPrice
                ).toFixed(2)}
              </div>
            </div>
            <img
              src={visiblePendingPayment.qrcodeUrl}
              alt={t('Scan to pay')}
              className='h-[220px] w-[220px] max-w-full object-contain'
            />
            <p className='text-muted-foreground text-center text-xs'>
              {visiblePendingPaymentMethodLabel}
            </p>
            {visiblePendingPayment.redirectUrl && (
              <Button
                type='button'
                variant='outline'
                onClick={() => {
                  window.location.href = visiblePendingPayment.redirectUrl
                }}
              >
                {t('Open payment page')}
              </Button>
            )}
          </div>
        )}
        <div className='bg-muted/50 space-y-2.5 rounded-lg border p-3 sm:space-y-3 sm:p-4'>
          <div className='flex justify-between'>
            <span className='text-muted-foreground text-sm'>
              {t('Plan Name')}
            </span>
            <span className='max-w-[200px] truncate text-sm font-medium'>
              {plan.title}
            </span>
          </div>
          <div className='flex items-center justify-between'>
            <span className='text-muted-foreground text-sm'>
              {t('Validity Period')}
            </span>
            <span className='flex items-center gap-1 text-sm'>
              <CalendarClock className='h-3.5 w-3.5' />
              {formatDuration(plan, t)}
            </span>
          </div>
          {formatResetPeriod(plan, t) !== t('No Reset') && (
            <div className='flex justify-between'>
              <span className='text-muted-foreground text-sm'>
                {t('Reset Period')}
              </span>
              <span className='text-sm'>{formatResetPeriod(plan, t)}</span>
            </div>
          )}
          <div className='flex items-center justify-between'>
            <span className='text-muted-foreground text-sm'>
              {t('Plan Quota')}
            </span>
            <span className='flex items-center gap-1 text-sm font-bold'>
              <Package className='h-3.5 w-3.5' />
              {totalAmount > 0 ? formatQuota(totalAmount) : t('Unlimited')}
            </span>
          </div>
          {plan.upgrade_group && (
            <div className='flex items-center justify-between'>
              <span className='text-muted-foreground text-sm'>
                {t('Upgrade Group')}
              </span>
              <GroupBadge group={plan.upgrade_group} />
            </div>
          )}
          <Separator />
          <div className='flex items-center justify-between'>
            <span className='text-sm font-medium'>{t('Amount Due')}</span>
            <span className='text-primary text-lg font-bold'>
              {hasHupijiao && Number(plan.hupijiao_discount_rate || 1) < 1 ? (
                <>
                  <span className='text-muted-foreground mr-2 text-sm line-through'>
                    {subscriptionSymbol}
                    {price}
                  </span>
                  {subscriptionSymbol}
                  {hupijiaoPrice}
                </>
              ) : (
                `${subscriptionSymbol}${price}`
              )}
            </span>
          </div>
          {hasHupijiao && Number(plan.hupijiao_discount_rate || 1) < 1 && (
            <div className='text-muted-foreground flex items-center justify-between text-xs'>
              <span>{t('Discount')}</span>
              <span>
                {(Number(plan.hupijiao_discount_rate || 1) * 10).toFixed(1)}
                {t('折')} · {t('Savings')} ¥
                {(
                  Number(plan.price_amount || 0) - Number(hupijiaoPrice)
                ).toFixed(2)}
              </span>
            </div>
          )}
        </div>

        {limitReached && (
          <Alert variant='destructive'>
            <AlertDescription>
              {t('Purchase limit reached')} ({props.purchaseCount}/
              {props.purchaseLimit})
            </AlertDescription>
          </Alert>
        )}

        {allowBalancePay && (
          <div className='flex flex-col gap-2 rounded-md border p-3'>
            <div className='flex items-center justify-between gap-2 text-xs'>
              <span className='text-muted-foreground'>{t('Required')}</span>
              <span>{formatQuota(balanceCost)}</span>
            </div>
            <div className='flex items-center justify-between gap-2 text-xs'>
              <span className='text-muted-foreground'>{t('Available')}</span>
              <span>{formatQuota(userQuota)}</span>
            </div>
            {insufficientBalance && (
              <Alert variant='destructive'>
                <AlertDescription>{t('Insufficient balance')}</AlertDescription>
              </Alert>
            )}
            <Button
              variant='outline'
              onClick={handlePayBalance}
              disabled={paying || limitReached || insufficientBalance}
            >
              {t('Pay with Balance')}
            </Button>
          </div>
        )}

        {hasAnyPayment && (
          <div className='space-y-3'>
            {(!hasHupijiao || (props.epayMethods || []).length > 1) && (
              <p className='text-muted-foreground text-xs'>
                {t('Select payment method')}
              </p>
            )}
            {(hasStripe || hasCreem || hasWaffoPancake) && (
              <div className='grid grid-cols-2 gap-2 sm:flex'>
                {hasStripe && (
                  <Button
                    variant='outline'
                    className='flex-1'
                    onClick={handlePayStripe}
                    disabled={paying || limitReached}
                  >
                    Stripe
                  </Button>
                )}
                {hasCreem && (
                  <Button
                    variant='outline'
                    className='flex-1'
                    onClick={handlePayCreem}
                    disabled={paying || limitReached}
                  >
                    Creem
                  </Button>
                )}
                {hasWaffoPancake && (
                  <Button
                    variant='outline'
                    className='flex-1'
                    onClick={handlePayWaffoPancake}
                    disabled={paying || limitReached}
                  >
                    Waffo Pancake
                  </Button>
                )}
              </div>
            )}
            {hasEpay && (
              <div
                className={
                  hasHupijiao && (props.epayMethods || []).length === 1
                    ? 'grid'
                    : 'grid grid-cols-[minmax(0,1fr)_auto] gap-2'
                }
              >
                {(!hasHupijiao || (props.epayMethods || []).length > 1) && (
                  <Select
                    items={(props.epayMethods || []).map((m) => ({
                      value: m.type,
                      label: m.name || m.type,
                    }))}
                    value={selectedEpayMethod}
                    onValueChange={(v) =>
                      v !== null && setSelectedEpayMethod(v)
                    }
                    disabled={limitReached}
                  >
                    <SelectTrigger className='flex-1'>
                      <SelectValue>{selectedEpayMethodLabel}</SelectValue>
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        {(props.epayMethods || []).map((m) => (
                          <SelectItem key={m.type} value={m.type}>
                            {m.name || m.type}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                )}
                <Button
                  onClick={handlePayEpay}
                  disabled={paying || !selectedEpayMethod || limitReached}
                  className={
                    hasHupijiao && (props.epayMethods || []).length === 1
                      ? 'h-11 bg-sky-600 font-semibold text-white hover:bg-sky-700'
                      : undefined
                  }
                >
                  {hasHupijiao && (props.epayMethods || []).length === 1
                    ? t('Pay with {{method}}', {
                        method: selectedEpayMethodLabel,
                      })
                    : t('Pay')}
                </Button>
              </div>
            )}
          </div>
        )}
      </div>
    </Dialog>
  )
}
