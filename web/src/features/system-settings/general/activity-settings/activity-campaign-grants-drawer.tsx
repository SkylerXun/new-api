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
import { useInfiniteQuery } from '@tanstack/react-query'
import { Loader2, RefreshCw, UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { formatTimestampToDate } from '@/lib/format'

import { getActivityCampaignGrants } from '../../api'
import type { ActivityCampaign } from '../../types'

type ActivityCampaignGrantsDrawerProps = {
  campaign: ActivityCampaign | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ActivityCampaignGrantsDrawer(
  props: ActivityCampaignGrantsDrawerProps
) {
  const { t } = useTranslation()
  const query = useInfiniteQuery({
    queryKey: ['activity-campaign-grants', props.campaign?.activity_key],
    enabled: props.open && props.campaign !== null,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => {
      if (!props.campaign) {
        throw new Error(t('Activity campaign is unavailable'))
      }
      return getActivityCampaignGrants(props.campaign.activity_key, pageParam)
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor,
  })
  const details = query.data?.pages.flatMap((page) => page.items) ?? []
  const isImmediate = props.campaign?.type === 'immediate'

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className='w-full sm:max-w-xl'>
        <SheetHeader className='border-b pr-12'>
          <SheetTitle>
            {isImmediate ? t('Credit details') : t('Claim details')}
          </SheetTitle>
          <SheetDescription className='break-words'>
            {props.campaign?.title ?? ''}
          </SheetDescription>
        </SheetHeader>

        <div className='min-h-0 flex-1 overflow-y-auto px-4 pb-4'>
          {props.campaign ? (
            <dl className='bg-muted/40 mb-4 grid grid-cols-2 gap-3 rounded-md p-3 text-sm'>
              <div>
                <dt className='text-muted-foreground'>
                  {t('Activity audience')}
                </dt>
                <dd className='mt-0.5 font-medium tabular-nums'>
                  {t('{{count}} users', {
                    count: props.campaign.recipient_count,
                  })}
                </dd>
              </div>
              <div>
                <dt className='text-muted-foreground'>
                  {isImmediate ? t('Credited users') : t('Claimed users')}
                </dt>
                <dd className='mt-0.5 font-medium tabular-nums'>
                  {t('{{count}} users', {
                    count: props.campaign.granted_count ?? 0,
                  })}
                </dd>
              </div>
            </dl>
          ) : null}

          {query.isLoading ? (
            <div
              className='space-y-2'
              aria-label={t('Loading activity details')}
            >
              <Skeleton className='h-20 w-full' />
              <Skeleton className='h-20 w-full' />
              <Skeleton className='h-20 w-full' />
            </div>
          ) : null}

          {query.isError ? (
            <div className='flex min-h-40 flex-col items-center justify-center gap-3 text-center'>
              <p className='text-destructive text-sm'>
                {t('Failed to load activity recipients')}
              </p>
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={() => void query.refetch()}
              >
                <RefreshCw aria-hidden='true' />
                {t('Retry')}
              </Button>
            </div>
          ) : null}

          {query.isSuccess && details.length === 0 ? (
            <div className='text-muted-foreground flex min-h-40 flex-col items-center justify-center gap-2 text-center'>
              <UserRound className='size-8' aria-hidden='true' />
              <p className='text-sm'>
                {isImmediate
                  ? t('No users have been credited yet.')
                  : t('No users have claimed this activity yet.')}
              </p>
            </div>
          ) : null}

          {details.length > 0 ? (
            <div className='divide-y rounded-md border'>
              {details.map((detail) => {
                const name =
                  detail.display_name ||
                  detail.username ||
                  t('User #{{id}}', { id: detail.user_id })
                const account = detail.username
                  ? `@${detail.username} · ID ${detail.user_id}`
                  : `ID ${detail.user_id}`
                return (
                  <article
                    key={detail.id}
                    className='flex flex-col gap-2 p-3 sm:flex-row sm:items-center sm:justify-between'
                  >
                    <div className='min-w-0'>
                      <p className='truncate font-medium'>{name}</p>
                      <p className='text-muted-foreground mt-0.5 truncate text-xs'>
                        {account}
                      </p>
                    </div>
                    <div className='shrink-0 text-left text-xs sm:text-right'>
                      <p className='font-mono font-medium tabular-nums'>
                        +{detail.quota.toLocaleString()}
                      </p>
                      <p className='text-muted-foreground mt-0.5 tabular-nums'>
                        {formatTimestampToDate(detail.granted_at)}
                      </p>
                    </div>
                  </article>
                )
              })}
            </div>
          ) : null}

          {query.hasNextPage ? (
            <Button
              type='button'
              variant='outline'
              className='mx-auto mt-4 flex'
              disabled={query.isFetchingNextPage}
              onClick={() => void query.fetchNextPage()}
            >
              {query.isFetchingNextPage ? (
                <Loader2 className='animate-spin' aria-hidden='true' />
              ) : null}
              {t('Load more')}
            </Button>
          ) : null}
        </div>
      </SheetContent>
    </Sheet>
  )
}
