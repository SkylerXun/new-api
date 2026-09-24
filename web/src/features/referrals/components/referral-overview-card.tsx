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
import {
  CircleCheckBig,
  Gift,
  Link2,
  Percent,
  UserPlus,
  Users,
  WalletCards,
} from 'lucide-react'
import { Trans, useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { formatQuota } from '@/lib/format'

import type { ReferralUserData } from '../types'

interface ReferralOverviewCardProps {
  user: ReferralUserData | null
  referralLink: string
  onTransfer: () => void
  complianceConfirmed: boolean
  rebateEnabled: boolean
  rebatePercent: number
  loading: boolean
}

const stepIcons = [Link2, UserPlus, Gift]
const loadingStepKeys = ['link', 'registration', 'rebate']

export function ReferralOverviewCard(props: ReferralOverviewCardProps) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <div className='overflow-hidden rounded-lg border'>
        <div className='grid gap-8 bg-[#f3f5ff] p-5 sm:p-8 @3xl/content:grid-cols-[minmax(0,1.7fr)_minmax(240px,0.8fr)]'>
          <div className='space-y-4'>
            <Skeleton className='h-4 w-28' />
            <Skeleton className='h-9 w-full max-w-xl' />
            <Skeleton className='h-16 w-full max-w-2xl' />
            <Skeleton className='h-20 w-full' />
          </div>
          <Skeleton className='min-h-64 rounded-lg' />
        </div>
        <div className='grid gap-6 p-6 md:grid-cols-3'>
          {loadingStepKeys.map((key) => (
            <Skeleton key={key} className='h-28 rounded-lg' />
          ))}
        </div>
      </div>
    )
  }

  const planActive =
    props.complianceConfirmed && props.rebateEnabled && props.rebatePercent > 0
  const hasRewards = (props.user?.aff_quota ?? 0) > 0
  const percent = new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 2,
  }).format(props.rebatePercent)
  const description =
    props.rebatePercent > 0
      ? t(
          'When a friend registers through your link and purchases quota, you earn {{percent}}% back. Rewards can be transferred to your balance at any time.',
          { percent }
        )
      : t(
          'When a friend registers through your link and purchases quota, you earn a rebate. Rewards can be transferred to your balance at any time.'
        )
  const steps = [
    {
      title: t('Get your referral link'),
      description: t(
        'Copy your unique referral link and share it with friends you trust.'
      ),
    },
    {
      title: t('Invite a friend to register'),
      description: t(
        'The referral is linked automatically after your friend completes registration.'
      ),
    },
    {
      title: t('Friend purchases, rebate arrives'),
      description: t(
        'When your friend purchases quota, your rebate is credited automatically.'
      ),
    },
  ]

  return (
    <div
      className='overflow-hidden rounded-lg border'
      data-slot='referral-page'
    >
      <section className='bg-[#f3f5ff] px-5 py-8 sm:px-8 sm:py-10 dark:bg-[#111827]'>
        <div
          className='grid items-stretch gap-8 @3xl/content:grid-cols-[minmax(0,1.7fr)_minmax(240px,0.8fr)] @3xl/content:gap-6 @5xl/content:gap-10'
          data-slot='referral-hero-grid'
        >
          <div className='flex min-w-0 flex-col justify-center'>
            <p className='text-primary mb-3 text-xs font-semibold'>
              {t('Referral rewards')}
            </p>
            <h1 className='max-w-3xl text-2xl leading-tight font-bold sm:text-3xl'>
              <Trans
                i18nKey='Share <product>General Tokens</product>, both sides enjoy <reward>quota rebates</reward>'
                components={{
                  product: <span className='text-primary' />,
                  reward: <span className='text-primary' />,
                }}
              />
            </h1>
            <p className='text-muted-foreground mt-3 max-w-3xl text-sm leading-6'>
              {description}
            </p>

            <div className='mt-6 rounded-lg border bg-white p-3 shadow-sm dark:bg-black/20'>
              <label
                htmlFor='referral-link'
                className='mb-2 block text-xs font-semibold'
              >
                {t('Your referral link')}
              </label>
              <div className='flex min-w-0 flex-col gap-2 sm:flex-row'>
                <Input
                  id='referral-link'
                  value={referralLinkOrFallback(props.referralLink, t)}
                  readOnly
                  className='bg-muted/30 h-10 min-w-0 flex-1 font-mono text-xs'
                />
                {props.referralLink ? (
                  <CopyButton
                    value={props.referralLink}
                    variant='default'
                    size='lg'
                    className='h-10 px-4'
                    tooltip={t('Copy referral link')}
                    successTooltip={t('Copied!')}
                    aria-label={t('Copy referral link')}
                  >
                    {t('Copy link')}
                  </CopyButton>
                ) : null}
              </div>
            </div>
          </div>

          <aside className='rounded-lg border bg-white p-5 shadow-sm dark:bg-black/20'>
            <div className='flex items-start justify-between gap-3 border-b pb-4'>
              <h2 className='text-sm font-semibold'>
                {t('Your referral overview')}
              </h2>
              <span
                className={
                  planActive
                    ? 'rounded-md bg-emerald-50 px-2 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
                    : 'bg-muted text-muted-foreground rounded-md px-2 py-1 text-xs font-medium'
                }
              >
                {planActive ? t('Rebate active') : t('Rebate paused')}
              </span>
            </div>

            <dl className='space-y-4 py-5'>
              <StatRow
                icon={Users}
                label={t('Successful referrals')}
                value={t('{{count}} people', {
                  count: props.user?.aff_count ?? 0,
                })}
              />
              <StatRow
                icon={Percent}
                label={t('Rebate rate')}
                value={`${percent}%`}
                accent={planActive}
              />
              <StatRow
                icon={WalletCards}
                label={t('Available rebate')}
                value={formatQuota(props.user?.aff_quota ?? 0)}
                accent={hasRewards}
              />
            </dl>

            <div className='border-t pt-4'>
              <p className='text-muted-foreground flex gap-2 text-xs leading-5'>
                <CircleCheckBig
                  className='mt-0.5 size-4 shrink-0 text-emerald-500'
                  aria-hidden='true'
                />
                {planActive
                  ? t(
                      'Rewards are credited automatically after your friend completes a quota purchase.'
                    )
                  : t('The rebate plan is not currently active.')}
              </p>
              {hasRewards ? (
                <Button
                  onClick={props.onTransfer}
                  disabled={!props.complianceConfirmed}
                  variant='outline'
                  className='mt-4 w-full'
                >
                  <WalletCards aria-hidden='true' />
                  {t('Transfer to Balance')}
                </Button>
              ) : null}
            </div>
          </aside>
        </div>
      </section>

      <section className='bg-background px-5 py-9 sm:px-8 sm:py-12'>
        <div className='text-center'>
          <p className='text-primary text-xs font-semibold'>
            {t('Simple to share, easy to earn')}
          </p>
          <h2 className='mt-2 text-xl font-bold'>
            {t('Three steps to earn rebates')}
          </h2>
        </div>
        <ol
          className='mt-8 grid gap-7 md:grid-cols-3 md:gap-8'
          data-slot='referral-steps'
        >
          {steps.map((step, index) => {
            const Icon = stepIcons[index]
            if (!Icon) return null
            return (
              <li key={step.title} className='min-w-0'>
                <div className='bg-primary/10 text-primary flex size-9 items-center justify-center rounded-lg'>
                  <Icon className='size-4' aria-hidden='true' />
                </div>
                <p className='text-primary mt-3 text-xs font-medium'>
                  {t('Step {{number}}', { number: index + 1 })}
                </p>
                <h3 className='mt-1 text-sm font-semibold'>{step.title}</h3>
                <p className='text-muted-foreground mt-1 text-xs leading-5'>
                  {step.description}
                </p>
              </li>
            )
          })}
        </ol>
      </section>
    </div>
  )
}

function referralLinkOrFallback(
  referralLink: string,
  translate: (key: string) => string
): string {
  return referralLink || translate('Referral link unavailable')
}

interface StatRowProps {
  icon: typeof Users
  label: string
  value: string
  accent?: boolean
}

function StatRow(props: StatRowProps) {
  const Icon = props.icon
  return (
    <div className='flex items-center gap-3'>
      <Icon
        className='text-muted-foreground size-4 shrink-0'
        aria-hidden='true'
      />
      <dt className='text-muted-foreground min-w-0 flex-1 text-xs'>
        {props.label}
      </dt>
      <dd
        className={
          props.accent
            ? 'text-primary text-sm font-bold tabular-nums'
            : 'text-sm font-bold tabular-nums'
        }
      >
        {props.value}
      </dd>
    </div>
  )
}
