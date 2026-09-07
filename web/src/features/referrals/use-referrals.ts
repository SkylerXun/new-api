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
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'

import {
  getReferralCode,
  getReferralSettings,
  getReferralUser,
  transferReferralRewards,
} from './api'
import type { ReferralUserData } from './types'

function generateReferralLink(code: string): string {
  if (typeof window === 'undefined') return ''
  return `${window.location.origin}/sign-up?aff=${code}`
}

export function useReferrals() {
  const [user, setUser] = useState<ReferralUserData | null>(null)
  const [referralLink, setReferralLink] = useState('')
  const [complianceConfirmed, setComplianceConfirmed] = useState(true)
  const [loading, setLoading] = useState(true)
  const [transferring, setTransferring] = useState(false)

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const [userResponse, codeResponse, settingsResponse] = await Promise.all([
        getReferralUser(),
        getReferralCode(),
        getReferralSettings().catch(() => null),
      ])

      if (userResponse.success && userResponse.data) {
        setUser(userResponse.data)
      }
      if (codeResponse.success && codeResponse.data) {
        setReferralLink(generateReferralLink(codeResponse.data))
      }
      setComplianceConfirmed(
        settingsResponse?.data?.payment_compliance_confirmed !== false
      )
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to load referral program:', error)
    } finally {
      setLoading(false)
    }
  }, [])

  const transferRewards = useCallback(
    async (quota: number): Promise<boolean> => {
      try {
        setTransferring(true)
        const response = await transferReferralRewards({ quota })
        if (!response.success) {
          toast.error(response.message || i18next.t('Transfer failed'))
          return false
        }

        toast.success(response.message || i18next.t('Transfer successful'))
        await refresh()
        return true
      } catch {
        toast.error(i18next.t('Transfer failed'))
        return false
      } finally {
        setTransferring(false)
      }
    },
    [refresh]
  )

  useEffect(() => {
    void refresh()
  }, [refresh])

  return {
    user,
    referralLink,
    complianceConfirmed,
    loading,
    transferring,
    transferRewards,
  }
}
