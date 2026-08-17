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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Gift, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { getSelf } from '@/lib/api'
import { formatQuota } from '@/lib/format'
import { useAuthStore, type AuthUser } from '@/stores/auth-store'

import { claimUserActivity, getUserActivities } from './api'
import { ActivityCard } from './components/activity-card'
import type { UserActivity } from './types'

const activityQueryKey = ['activities', 'self'] as const

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
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: activityQueryKey,
    queryFn: async () => {
      const response = await getUserActivities()
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load activities'))
      }
      return response.data
    },
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
        queryClient.invalidateQueries({ queryKey: activityQueryKey }),
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

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Activity Center')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-5xl flex-col gap-4'>
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

          {query.data?.activities.length === 0 ? (
            <Card
              data-card-hover='false'
              className='flex flex-col items-center gap-2 p-10 text-center'
            >
              <Gift className='text-muted-foreground size-8' />
              <p className='text-sm font-medium'>{t('No activities yet')}</p>
            </Card>
          ) : null}

          {query.data?.activities.map((activity) => (
            <ActivityCard
              key={activity.id}
              activity={activity}
              serverTime={query.data.server_time}
              claiming={
                claimMutation.isPending &&
                claimMutation.variables?.id === activity.id
              }
              onClaim={handleClaim}
            />
          ))}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
