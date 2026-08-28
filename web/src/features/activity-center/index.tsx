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
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query'
import { Gift, Loader2, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getSelf } from '@/lib/api'
import { formatQuota } from '@/lib/format'
import { useAuthStore, type AuthUser } from '@/stores/auth-store'

import {
  activityAttentionQueryKey,
  claimUserActivity,
  getUserActivities,
  userActivitiesQueryKey,
} from './api'
import { ActivityCard } from './components/activity-card'
import type { UserActivity } from './types'

async function refreshCurrentUser() {
  const response = await getSelf()
  if (!response?.success || !response.data) return

  const auth = useAuthStore.getState().auth
  const currentUser = auth.user
  const refreshedUser = response.data as AuthUser
  auth.setUser(
    currentUser ? { ...currentUser, ...refreshedUser } : refreshedUser
  )
}

export function ActivityCenter() {
  const { t } = useTranslation()
  const [view, setView] = useState<'ongoing' | 'participated'>('ongoing')
  const queryClient = useQueryClient()
  const query = useInfiniteQuery({
    queryKey: [...userActivitiesQueryKey, view],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const response = await getUserActivities(view, pageParam)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load activities'))
      }
      return response.data
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor,
    staleTime: 30_000,
  })
  const claimMutation = useMutation({
    mutationFn: async (activity: UserActivity) => {
      const response = await claimUserActivity(activity.id)
      if (!response.success) {
        throw new Error(response.message || t('Failed to claim activity'))
      }
      return response.data
    },
    onSuccess: async (data) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: userActivitiesQueryKey }),
        queryClient.invalidateQueries({ queryKey: activityAttentionQueryKey }),
        refreshCurrentUser(),
      ])
      toast.success(
        data?.granted
          ? t('Activity credit received: {{quota}}', {
              quota: formatQuota(data.reward_quota),
            })
          : t('Activity already claimed')
      )
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Failed to claim activity')
      )
    },
  })

  const handleClaim = (activity: UserActivity) => {
    if (!claimMutation.isPending) claimMutation.mutate(activity)
  }
  const activities = query.data?.pages.flatMap((page) => page.activities) ?? []
  const serverTime = query.data?.pages[0]?.server_time ?? 0

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Activity Center')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-5xl flex-col gap-4'>
          <Tabs
            value={view}
            onValueChange={(value) => {
              if (value === 'ongoing' || value === 'participated') {
                setView(value)
              }
            }}
          >
            <TabsList className='grid w-full max-w-md grid-cols-2'>
              <TabsTrigger value='ongoing'>{t('Ongoing')}</TabsTrigger>
              <TabsTrigger value='participated'>
                {t('Participation history')}
              </TabsTrigger>
            </TabsList>
          </Tabs>
          {query.isLoading ? (
            <Card data-card-hover='false' className='gap-3 p-5'>
              <Skeleton className='h-6 w-44' />
              <Skeleton className='h-4 w-full max-w-xl' />
              <Skeleton className='h-20 w-full max-w-lg' />
            </Card>
          ) : null}

          {query.isError ? (
            <Card
              data-card-hover='false'
              className='flex flex-col items-center gap-3 p-8 text-center'
            >
              <Gift className='text-muted-foreground size-8' />
              <p className='text-sm font-medium'>
                {t('Failed to load activities')}
              </p>
              <Button
                variant='outline'
                size='sm'
                onClick={() => query.refetch()}
              >
                <RefreshCw />
                {t('Retry')}
              </Button>
            </Card>
          ) : null}

          {activities.length === 0 && query.isSuccess ? (
            <Card
              data-card-hover='false'
              className='flex flex-col items-center gap-2 p-10 text-center'
            >
              <Gift className='text-muted-foreground size-8' />
              <p className='text-sm font-medium'>{t('No activities yet')}</p>
            </Card>
          ) : null}

          {activities.map((activity) => (
            <ActivityCard
              key={activity.id}
              activity={activity}
              serverTime={serverTime}
              claiming={
                claimMutation.isPending &&
                claimMutation.variables?.id === activity.id
              }
              onClaim={handleClaim}
            />
          ))}
          {query.hasNextPage ? (
            <Button
              variant='outline'
              className='self-center'
              disabled={query.isFetchingNextPage}
              onClick={() => void query.fetchNextPage()}
            >
              {query.isFetchingNextPage ? (
                <Loader2 className='animate-spin' />
              ) : null}
              {t('Load more')}
            </Button>
          ) : null}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
