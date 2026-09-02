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

import { Link } from '@tanstack/react-router'
import { CheckCircle2, Clock3, Gift, Loader2, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { formatQuota, formatTimestampToDate } from '@/lib/format'

import { useActivityCountdown } from '../hooks/use-activity-countdown'
import { splitCountdown } from '../lib/countdown'
import type { UserActivity } from '../types'

type ActivityCardProps = {
  activity: UserActivity
  serverTime: number
  claiming?: boolean
  onClaim?: (activity: UserActivity) => void
}

const statusVariant = {
  active: 'warning',
  claimable: 'warning',
  claimed: 'secondary',
  credited: 'secondary',
  expired: 'outline',
  closed: 'outline',
  unavailable: 'outline',
} as const

export function ActivityCard(props: ActivityCardProps) {
  const { t } = useTranslation()
  const remaining = useActivityCountdown(
    props.activity.ends_at,
    props.serverTime
  )
  const countdown = splitCountdown(remaining)
  const locallyExpired =
    (props.activity.status === 'active' ||
      props.activity.status === 'claimable') &&
    remaining === 0
  const status = locallyExpired ? 'expired' : props.activity.status
  const canShowCountdown = status === 'active' || status === 'claimable'
  const activityTitle =
    props.activity.type === 'new_user_topup_bonus'
      ? t('New user recharge bonus')
      : props.activity.title
  const activityDescription =
    props.activity.type === 'new_user_topup_bonus'
      ? t(
          'Top up within {{days}} days of registration for an extra {{percent}}% credit each time.',
          {
            days: Math.max(
              0,
              Math.round(
                (props.activity.ends_at - props.activity.starts_at) / 86_400
              )
            ),
            percent: props.activity.bonus_percent,
          }
        )
      : props.activity.description
  const labels = [
    { value: countdown.days, label: t('Days') },
    { value: countdown.hours, label: t('Hours') },
    { value: countdown.minutes, label: t('Minutes') },
    { value: countdown.seconds, label: t('Seconds') },
  ]

  return (
    <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
      <div className='flex flex-col gap-4 border-b p-4 sm:flex-row sm:items-start sm:justify-between sm:p-5'>
        <div className='flex min-w-0 items-start gap-3'>
          <IconBadge tone='warning' size='lg'>
            <Gift />
          </IconBadge>
          <div className='min-w-0'>
            <div className='flex flex-wrap items-center gap-2'>
              <h3 className='text-base font-semibold sm:text-lg'>
                {activityTitle}
              </h3>
              <Badge variant={statusVariant[status]}>{t(status)}</Badge>
            </div>
            <p className='text-muted-foreground mt-1 text-sm leading-6'>
              {activityDescription}
            </p>
          </div>
        </div>

        {canShowCountdown && props.activity.action?.type === 'navigate' ? (
          <Button
            className='w-full shrink-0 sm:w-auto'
            render={<Link to='/purchase' />}
          >
            <WalletCards />
            {t('Recharge now')}
          </Button>
        ) : null}
        {canShowCountdown && props.activity.action?.type === 'claim' ? (
          <Button
            className='w-full shrink-0 sm:w-auto'
            disabled={props.claiming}
            onClick={() => props.onClaim?.(props.activity)}
          >
            {props.claiming ? <Loader2 className='animate-spin' /> : <Gift />}
            {t('Claim now')}
          </Button>
        ) : null}
      </div>

      {canShowCountdown ? (
        <div className='p-4 sm:p-5'>
          <div className='text-muted-foreground mb-3 flex items-center gap-2 text-xs font-medium'>
            <Clock3 className='size-4' />
            {t('Time remaining')}
          </div>
          <div className='grid grid-cols-4 gap-2 sm:max-w-lg sm:gap-3'>
            {labels.map((item) => (
              <div
                key={item.label}
                className='bg-muted/60 min-w-0 rounded-md px-2 py-3 text-center'
              >
                <div className='text-xl font-semibold tabular-nums sm:text-2xl'>
                  {String(item.value).padStart(2, '0')}
                </div>
                <div className='text-muted-foreground mt-1 truncate text-[10px] sm:text-xs'>
                  {item.label}
                </div>
              </div>
            ))}
          </div>
        </div>
      ) : null}

      {props.activity.reward_quota ? (
        <div className='flex items-center gap-3 p-4 text-sm sm:p-5'>
          <CheckCircle2 className='size-5 text-emerald-600 dark:text-emerald-400' />
          <span>
            {props.activity.type === 'new_user_topup_bonus'
              ? t('Cumulative bonus received')
              : t('Credit received')}
          </span>
          <span className='font-semibold'>
            +{formatQuota(props.activity.reward_quota)}
          </span>
        </div>
      ) : null}

      {props.activity.granted_at ? (
        <div className='text-muted-foreground border-t px-4 py-3 text-xs sm:px-5'>
          {t('Granted at')}: {formatTimestampToDate(props.activity.granted_at)}
        </div>
      ) : null}
    </Card>
  )
}
