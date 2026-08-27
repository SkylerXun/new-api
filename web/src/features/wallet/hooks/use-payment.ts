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
import i18next from 'i18next'
import { useState, useCallback, useEffect } from 'react'
import { toast } from 'sonner'

import {
  calculateAmount,
  calculateStripeAmount,
  calculateWaffoAmount,
  calculateWaffoPancakeAmount,
  requestPayment,
  requestStripePayment,
  getHupijiaoPaymentStatus,
  isApiSuccess,
} from '../api'
import {
  isStripePayment,
  isWaffoPayment,
  isWaffoPancakePayment,
  submitPaymentForm,
} from '../lib'
import type { AmountRequest, AmountResponse } from '../types'

// ============================================================================
// Payment Hook
// ============================================================================

type AmountCalculator = (request: AmountRequest) => Promise<AmountResponse>

export interface PaymentAmountCalculators {
  regular: AmountCalculator
  stripe: AmountCalculator
  waffo: AmountCalculator
  waffoPancake: AmountCalculator
}

const defaultPaymentAmountCalculators: PaymentAmountCalculators = {
  regular: calculateAmount,
  stripe: calculateStripeAmount,
  waffo: calculateWaffoAmount,
  waffoPancake: calculateWaffoPancakeAmount,
}

const PENDING_HUPIJIAO_PAYMENT_KEY = 'new-api:pending-hupijiao-topup'
const PENDING_PAYMENT_MAX_AGE = 30 * 60 * 1000

interface PendingHupijiaoPayment {
  tradeNo: string
  qrcodeUrl: string
  redirectUrl: string
  actualAmount: number
  paymentMethod: string
  createdAt: number
}

function readPendingHupijiaoPayment(): PendingHupijiaoPayment | null {
  if (typeof window === 'undefined') return null
  try {
    const parsed = JSON.parse(
      window.sessionStorage.getItem(PENDING_HUPIJIAO_PAYMENT_KEY) || 'null'
    ) as PendingHupijiaoPayment | null
    if (
      !parsed?.tradeNo ||
      Date.now() - Number(parsed.createdAt || 0) > PENDING_PAYMENT_MAX_AGE
    ) {
      window.sessionStorage.removeItem(PENDING_HUPIJIAO_PAYMENT_KEY)
      return null
    }
    return parsed
  } catch {
    window.sessionStorage.removeItem(PENDING_HUPIJIAO_PAYMENT_KEY)
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

export async function requestPaymentAmount(
  topupAmount: number,
  paymentType: string,
  calculators: PaymentAmountCalculators = defaultPaymentAmountCalculators
): Promise<number> {
  let calculator = calculators.regular
  if (isStripePayment(paymentType)) {
    calculator = calculators.stripe
  } else if (isWaffoPayment(paymentType)) {
    calculator = calculators.waffo
  } else if (isWaffoPancakePayment(paymentType)) {
    calculator = calculators.waffoPancake
  }

  const response = await calculator({ amount: topupAmount })
  if (!isApiSuccess(response) || !response.data) {
    return 0
  }

  return Number.parseFloat(response.data)
}

export function usePayment() {
  const [initialPendingPayment] = useState(readPendingHupijiaoPayment)
  const [amount, setAmount] = useState<number>(0)
  const [calculating, setCalculating] = useState(false)
  const [processing, setProcessing] = useState(false)
  const [qrcodeUrl, setQrcodeUrl] = useState(
    initialPendingPayment?.qrcodeUrl || ''
  )
  const [redirectUrl, setRedirectUrl] = useState(
    initialPendingPayment?.redirectUrl || ''
  )
  const [pendingTradeNo, setPendingTradeNo] = useState(
    initialPendingPayment?.tradeNo || ''
  )
  const [pendingCreatedAt, setPendingCreatedAt] = useState(
    initialPendingPayment?.createdAt || 0
  )
  const [pendingActualAmount, setPendingActualAmount] = useState(
    Number(initialPendingPayment?.actualAmount || 0)
  )
  const [pendingPaymentMethod, setPendingPaymentMethod] = useState(
    initialPendingPayment?.paymentMethod || ''
  )
  const [qrcodeOpen, setQrcodeOpen] = useState(Boolean(initialPendingPayment))
  const [paymentCompletedAt, setPaymentCompletedAt] = useState(0)

  const clearPendingPayment = useCallback(() => {
    setPendingTradeNo('')
    setQrcodeUrl('')
    setRedirectUrl('')
    setPendingCreatedAt(0)
    setPendingActualAmount(0)
    setPendingPaymentMethod('')
    window.sessionStorage.removeItem(PENDING_HUPIJIAO_PAYMENT_KEY)
  }, [])

  useEffect(() => {
    if (!pendingTradeNo) return
    let active = true
    let finished = false
    let timer: number | undefined

    const poll = async () => {
      try {
        if (
          pendingCreatedAt > 0 &&
          Date.now() - pendingCreatedAt > PENDING_PAYMENT_MAX_AGE
        ) {
          finished = true
          setQrcodeOpen(false)
          clearPendingPayment()
          toast.error(i18next.t('Payment order expired or failed.'))
          return
        }
        const response = await getHupijiaoPaymentStatus(pendingTradeNo)
        if (!active) return
        if (!response.success || !response.data) {
          finished = true
          setQrcodeOpen(false)
          clearPendingPayment()
          toast.error(
            response.message || i18next.t('Payment order expired or failed.')
          )
          return
        }
        if (response.data.status === 'success') {
          finished = true
          setQrcodeOpen(false)
          clearPendingPayment()
          setPaymentCompletedAt(Date.now())
          toast.success(i18next.t('Payment successful. Balance updated.'))
          return
        }
        if (['failed', 'expired'].includes(response.data.status)) {
          finished = true
          setQrcodeOpen(false)
          clearPendingPayment()
          toast.error(i18next.t('Payment order expired or failed.'))
        }
      } catch {
        // Temporary polling failures should not close a valid payment page.
      } finally {
        if (active && !finished) {
          timer = window.setTimeout(poll, 2000)
        }
      }
    }

    timer = window.setTimeout(poll, 1200)
    return () => {
      active = false
      if (timer) window.clearTimeout(timer)
    }
  }, [clearPendingPayment, pendingCreatedAt, pendingTradeNo])

  // Calculate payment amount
  const calculatePaymentAmount = useCallback(
    async (topupAmount: number, paymentType: string) => {
      try {
        setCalculating(true)
        const calculatedAmount = await requestPaymentAmount(
          topupAmount,
          paymentType
        )
        setAmount(calculatedAmount)
        return calculatedAmount
      } catch {
        setAmount(0)
        return 0
      } finally {
        setCalculating(false)
      }
    },
    []
  )

  // Process payment
  const processPayment = useCallback(
    async (topupAmount: number, paymentType: string, packageId?: string) => {
      try {
        setProcessing(true)

        const isStripe = isStripePayment(paymentType)
        const isMobile = /Mobile|Android|iPhone/i.test(navigator.userAgent)
        const amount = Math.floor(topupAmount)

        const response = isStripe
          ? await requestStripePayment({
              amount,
              payment_method: 'stripe',
            })
          : await requestPayment({
              ...(packageId ? { package_id: packageId } : { amount }),
              payment_method: paymentType,
              device: isMobile ? 'mobile' : 'pc',
            })

        if (!isApiSuccess(response)) {
          toast.error(response.message || i18next.t('Payment request failed'))
          return false
        }

        // Handle Stripe payment
        if (isStripe && response.data?.pay_link) {
          window.open(response.data.pay_link as string, '_blank')
          toast.success(i18next.t('Redirecting to payment page...'))
          return true
        }

        // Handle non-Stripe payment
        const paymentData = response.data as Record<string, unknown> | undefined
        if (!isStripe && isMobile && paymentData?.redirect_url) {
          const tradeNo = String(paymentData.trade_no || '')
          if (tradeNo) {
            const pending = {
              tradeNo,
              qrcodeUrl: String(paymentData.qrcode_url || ''),
              redirectUrl: String(paymentData.redirect_url || ''),
              actualAmount: Number(paymentData.actual_amount || 0),
              paymentMethod: paymentType,
              createdAt: Date.now(),
            }
            window.sessionStorage.setItem(
              PENDING_HUPIJIAO_PAYMENT_KEY,
              JSON.stringify(pending)
            )
          }
          window.location.href = paymentData.redirect_url as string
          toast.success(i18next.t('Redirecting to payment page...'))
          return true
        }
        if (!isStripe && paymentData?.qrcode_url) {
          const nextQRCodeUrl = String(paymentData.qrcode_url)
          const nextRedirectUrl = String(paymentData.redirect_url || '')
          const tradeNo = String(paymentData.trade_no || '')
          setQrcodeUrl(nextQRCodeUrl)
          setRedirectUrl(nextRedirectUrl)
          setPendingTradeNo(tradeNo)
          setPendingCreatedAt(Date.now())
          setPendingActualAmount(Number(paymentData.actual_amount || 0))
          setPendingPaymentMethod(paymentType)
          setQrcodeOpen(true)
          if (tradeNo) {
            window.sessionStorage.setItem(
              PENDING_HUPIJIAO_PAYMENT_KEY,
              JSON.stringify({
                tradeNo,
                qrcodeUrl: nextQRCodeUrl,
                redirectUrl: nextRedirectUrl,
                actualAmount: Number(paymentData.actual_amount || 0),
                paymentMethod: paymentType,
                createdAt: Date.now(),
              } satisfies PendingHupijiaoPayment)
            )
          }
          return true
        }
        if (!isStripe && response.data) {
          const url = (response as unknown as { url?: string }).url
          if (url) {
            submitPaymentForm(url, response.data)
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
        }

        return false
      } catch (error) {
        toast.error(
          getPaymentRequestError(error, i18next.t('Payment request failed'))
        )
        return false
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  return {
    amount,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
    setAmount,
    qrcodeUrl,
    redirectUrl,
    qrcodeOpen,
    pendingTradeNo,
    paymentCompletedAt,
    pendingActualAmount,
    pendingPaymentMethod,
    closeQRCode: () => setQrcodeOpen(false),
    resumeQRCode: () => setQrcodeOpen(true),
  }
}
