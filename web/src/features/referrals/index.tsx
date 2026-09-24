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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'

import { ReferralOverviewCard } from './components/referral-overview-card'
import { TransferDialog } from './components/transfer-dialog'
import { useReferrals } from './use-referrals'

export function Referrals() {
  const { t } = useTranslation()
  const [transferDialogOpen, setTransferDialogOpen] = useState(false)
  const {
    user,
    referralLink,
    complianceConfirmed,
    rebateEnabled,
    rebatePercent,
    loading,
    transferring,
    transferRewards,
  } = useReferrals()

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Referral Program')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='mx-auto w-full max-w-7xl'>
            <ReferralOverviewCard
              user={user}
              referralLink={referralLink}
              onTransfer={() => setTransferDialogOpen(true)}
              complianceConfirmed={complianceConfirmed}
              rebateEnabled={rebateEnabled}
              rebatePercent={rebatePercent}
              loading={loading}
            />
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <TransferDialog
        open={transferDialogOpen}
        onOpenChange={setTransferDialogOpen}
        onConfirm={transferRewards}
        availableQuota={user?.aff_quota ?? 0}
        transferring={transferring}
      />
    </>
  )
}
