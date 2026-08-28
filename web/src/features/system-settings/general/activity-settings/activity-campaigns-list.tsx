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
import { Clock3, List, Loader2, RefreshCw, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { formatTimestampToDate } from '@/lib/format'

import type {
  ActivityCampaign,
  ActivityCampaignStatus,
  AllUsersActivityGrantTask,
} from '../../types'
import { getAllUsersGrantProgress } from './lib'

type ActivityCampaignsListProps = {
  campaigns?: ActivityCampaign[]
  isLoading: boolean
  isError: boolean
  isTaskProgressError: boolean
  taskById: Record<string, AllUsersActivityGrantTask>
  closingActivityKey?: string
  onClose: (campaign: ActivityCampaign) => void
  onViewGrants: (campaign: ActivityCampaign) => void
  onRetry: () => void
  onRetryTaskProgress: () => void
}

function statusLabel(status: ActivityCampaignStatus): string {
  switch (status) {
    case 'active':
      return 'Active'
    case 'queued':
      return 'Queued'
    case 'running':
      return 'Running'
    case 'completed':
      return 'Completed'
    case 'failed':
      return 'Failed'
    case 'closed':
      return 'Closed'
  }
}

function statusVariant(status: ActivityCampaignStatus) {
  switch (status) {
    case 'active':
      return 'default' as const
    case 'queued':
    case 'running':
      return 'warning' as const
    case 'completed':
      return 'outline' as const
    case 'failed':
      return 'destructive' as const
    case 'closed':
      return 'secondary' as const
  }
}

function ImmediateCampaignProgress(props: {
  campaign: ActivityCampaign
  task?: AllUsersActivityGrantTask
}) {
  const { t } = useTranslation()
  const state = props.task?.state
  const result = props.task?.result
  const processed = result?.processed ?? state?.processed ?? 0
  const total = result?.total ?? state?.total ?? props.campaign.recipient_count
  const granted = result?.granted ?? state?.granted ?? 0
  const skipped = result?.skipped ?? state?.skipped ?? 0

  if (!props.task) {
    if (
      props.campaign.status !== 'queued' &&
      props.campaign.status !== 'running'
    ) {
      return null
    }
    return (
      <p className='text-muted-foreground flex items-center gap-1.5 text-xs'>
        <Clock3 className='size-3.5' aria-hidden='true' />
        {t('Waiting for the grant task to report progress.')}
      </p>
    )
  }

  return (
    <div className='space-y-2' aria-live='polite'>
      <div className='flex flex-wrap items-center justify-between gap-2 text-xs'>
        <span className='text-muted-foreground'>
          {t('{{processed}} of {{total}} users processed.', {
            processed,
            total,
          })}
        </span>
        <span className='text-muted-foreground tabular-nums'>
          {getAllUsersGrantProgress(props.task)}%
        </span>
      </div>
      <Progress
        value={getAllUsersGrantProgress(props.task)}
        aria-label={t('Campaign progress')}
      />
      <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs'>
        <span>{t('{{granted}} credited', { granted })}</span>
        <span>{t('{{skipped}} skipped', { skipped })}</span>
      </div>
      {props.task.status === 'failed' && props.task.error ? (
        <p className='text-destructive text-xs'>{props.task.error}</p>
      ) : null}
    </div>
  )
}

export function ActivityCampaignsList(props: ActivityCampaignsListProps) {
  const { t } = useTranslation()

  if (props.isLoading) {
    return (
      <div className='space-y-3' aria-label={t('Loading activity campaigns')}>
        <Skeleton className='h-28 w-full' />
        <Skeleton className='h-28 w-full' />
      </div>
    )
  }

  if (props.isError) {
    return (
      <div className='flex flex-wrap items-center gap-2'>
        <p className='text-destructive text-sm'>
          {t('Failed to load activity campaigns')}
        </p>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={props.onRetry}
        >
          <RefreshCw />
          {t('Retry')}
        </Button>
      </div>
    )
  }

  if (!props.campaigns?.length) {
    return (
      <div className='text-muted-foreground rounded-md border border-dashed px-4 py-8 text-center text-sm'>
        {t('No activity campaigns have been created yet.')}
      </div>
    )
  }

  return (
    <div className='space-y-3'>
      {props.isTaskProgressError ? (
        <div className='flex flex-wrap items-center gap-2'>
          <p className='text-destructive text-xs'>
            {t('Failed to refresh campaign progress')}
          </p>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            onClick={props.onRetryTaskProgress}
          >
            <RefreshCw />
            {t('Retry')}
          </Button>
        </div>
      ) : null}

      <div className='divide-y rounded-md border'>
        {props.campaigns.map((campaign) => {
          const task = campaign.task_id
            ? props.taskById[campaign.task_id]
            : undefined
          const isClosing = props.closingActivityKey === campaign.activity_key
          const isClosable =
            campaign.type === 'claimable' && campaign.status === 'active'

          return (
            <article key={campaign.activity_key} className='space-y-4 p-4'>
              <div className='flex flex-wrap items-start justify-between gap-3'>
                <div className='min-w-0 space-y-1'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <h5 className='min-w-0 text-sm font-medium break-words'>
                      {campaign.title}
                    </h5>
                    <Badge variant={statusVariant(campaign.status)}>
                      {t(statusLabel(campaign.status))}
                    </Badge>
                    <Badge variant='outline'>
                      {campaign.type === 'claimable'
                        ? t('Claimable')
                        : t('Immediate credit')}
                    </Badge>
                    <Badge variant='outline'>
                      {campaign.audience_type === 'selected'
                        ? t('Selected users')
                        : t('All users')}
                    </Badge>
                  </div>
                  {campaign.description ? (
                    <p className='text-muted-foreground max-w-3xl text-sm whitespace-pre-wrap'>
                      {campaign.description}
                    </p>
                  ) : null}
                </div>
                <div className='flex shrink-0 flex-wrap items-center gap-2'>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => props.onViewGrants(campaign)}
                  >
                    <List aria-hidden='true' />
                    {campaign.type === 'immediate'
                      ? t('View credit details')
                      : t('View claim details')}
                  </Button>
                  {isClosable ? (
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      disabled={isClosing}
                      onClick={() => props.onClose(campaign)}
                    >
                      {isClosing ? <Loader2 className='animate-spin' /> : <X />}
                      {t('Close activity')}
                    </Button>
                  ) : null}
                </div>
              </div>

              <dl className='grid gap-x-5 gap-y-3 text-xs sm:grid-cols-2 lg:grid-cols-5'>
                <div>
                  <dt className='text-muted-foreground'>{t('Amount')}</dt>
                  <dd className='mt-0.5 font-mono font-medium'>
                    ${campaign.amount_usd}
                  </dd>
                </div>
                <div>
                  <dt className='text-muted-foreground'>
                    {t('Activity audience')}
                  </dt>
                  <dd className='mt-0.5 tabular-nums'>
                    {t('{{count}} users', { count: campaign.recipient_count })}
                  </dd>
                </div>
                <div>
                  <dt className='text-muted-foreground'>
                    {campaign.type === 'immediate'
                      ? t('Credited users')
                      : t('Claimed users')}
                  </dt>
                  <dd className='mt-0.5 tabular-nums'>
                    {t('{{count}} users', {
                      count: campaign.granted_count ?? 0,
                    })}
                  </dd>
                </div>
                <div>
                  <dt className='text-muted-foreground'>
                    {campaign.type === 'claimable'
                      ? t('Claim deadline')
                      : t('Created')}
                  </dt>
                  <dd className='mt-0.5 tabular-nums'>
                    {formatTimestampToDate(
                      campaign.type === 'claimable'
                        ? campaign.ends_at
                        : campaign.created_at
                    )}
                  </dd>
                </div>
                <div>
                  <dt className='text-muted-foreground'>
                    {t('Quota per user')}
                  </dt>
                  <dd className='mt-0.5 font-mono tabular-nums'>
                    {campaign.quota.toLocaleString()}
                  </dd>
                  <p className='text-muted-foreground mt-1'>
                    {t('Fixed when published; no account balance is frozen.')}
                  </p>
                </div>
              </dl>

              {campaign.type === 'immediate' ? (
                <ImmediateCampaignProgress campaign={campaign} task={task} />
              ) : null}
              {campaign.failure_reason ? (
                <p className='text-destructive text-xs'>
                  {campaign.failure_reason}
                </p>
              ) : null}
            </article>
          )
        })}
      </div>
    </div>
  )
}
