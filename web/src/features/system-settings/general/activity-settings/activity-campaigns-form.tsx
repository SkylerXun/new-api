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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Gift, HandCoins, Loader2, Send, Search, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'

import {
  closeActivityCampaign,
  createActivityCampaign,
  getActivityCampaigns,
  getSystemTask,
} from '../../api'
import type {
  ActivityCampaign,
  ActivityCampaignType,
  AllUsersActivityGrantTask,
  CreateActivityCampaignRequest,
} from '../../types'
import { ActivityCampaignsList } from './activity-campaigns-list'
import {
  activityCampaignSchema,
  parseActivityCampaignEndAt,
  type ActivityCampaignFormValues,
} from './lib'

const activityCampaignsQueryKey = ['activity-campaigns'] as const
const emptyActivityCampaigns: ActivityCampaign[] = []

const defaultActivityCampaignFormValues: ActivityCampaignFormValues = {
  type: 'claimable',
  title: '',
  description: '',
  amountUSD: '',
  endsAt: '',
  audienceType: 'all',
}

function isActiveImmediateCampaign(campaign: ActivityCampaign): boolean {
  return (
    campaign.type === 'immediate' &&
    (campaign.status === 'queued' || campaign.status === 'running')
  )
}

export function ActivityCampaignsForm() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmation, setConfirmation] =
    useState<ActivityCampaignFormValues | null>(null)
  const [campaignToClose, setCampaignToClose] =
    useState<ActivityCampaign | null>(null)
  const [selectedUsers, setSelectedUsers] = useState<User[]>([])
  const [userSearch, setUserSearch] = useState('')
  const [userResults, setUserResults] = useState<User[]>([])
  const [searchingUsers, setSearchingUsers] = useState(false)
  const form = useForm<ActivityCampaignFormValues>({
    resolver: zodResolver(activityCampaignSchema),
    defaultValues: defaultActivityCampaignFormValues,
  })
  const campaignType = form.watch('type')
  const audienceType = form.watch('audienceType')

  const campaignsQuery = useQuery({
    queryKey: activityCampaignsQueryKey,
    queryFn: getActivityCampaigns,
    refetchInterval: (query) =>
      query.state.data?.some(isActiveImmediateCampaign) ? 1000 : false,
  })
  const campaigns = campaignsQuery.data ?? emptyActivityCampaigns
  const hasActiveImmediateCampaign = campaigns.some(isActiveImmediateCampaign)
  const taskIds = useMemo(
    () =>
      campaigns
        .filter(
          (campaign) => isActiveImmediateCampaign(campaign) && campaign.task_id
        )
        .map((campaign) => campaign.task_id as string),
    [campaigns]
  )

  const campaignTasksQuery = useQuery({
    queryKey: ['activity-campaign-tasks', taskIds],
    enabled: taskIds.length > 0,
    queryFn: async (): Promise<Record<string, AllUsersActivityGrantTask>> => {
      const taskEntries = await Promise.all(
        taskIds.map(async (taskId) => {
          const response =
            await getSystemTask<AllUsersActivityGrantTask>(taskId)
          if (!response.success || !response.data) {
            throw new Error(response.message || 'Failed to load campaign task')
          }
          return [taskId, response.data] as const
        })
      )
      return Object.fromEntries(taskEntries)
    },
    refetchInterval: hasActiveImmediateCampaign ? 1000 : false,
  })

  const createMutation = useMutation({
    mutationFn: async (values: ActivityCampaignFormValues) => {
      const request: CreateActivityCampaignRequest = {
        type: values.type,
        title: values.title,
        description: values.description || undefined,
        amount_usd: values.amountUSD,
        audience_type: values.audienceType,
        recipient_user_ids:
          values.audienceType === 'selected'
            ? selectedUsers.map((user) => user.id)
            : undefined,
      }
      if (values.type === 'claimable') {
        const endsAt = parseActivityCampaignEndAt(values.endsAt)
        if (!endsAt) throw new Error(t('A valid claim deadline is required.'))
        request.ends_at = endsAt
      }

      const response = await createActivityCampaign(request)
      if (!response.success || !response.data?.campaign) {
        throw new Error(
          response.message || t('Failed to create activity campaign')
        )
      }
      return response.data.campaign
    },
    onSuccess: (campaign) => {
      queryClient.setQueryData<ActivityCampaign[]>(
        activityCampaignsQueryKey,
        (current) => [
          campaign,
          ...(current ?? []).filter(
            (item) => item.activity_key !== campaign.activity_key
          ),
        ]
      )
      void queryClient.invalidateQueries({
        queryKey: activityCampaignsQueryKey,
      })
      setConfirmation(null)
      form.reset(defaultActivityCampaignFormValues)
      setSelectedUsers([])
      toast.success(
        campaign.type === 'immediate'
          ? t('Immediate activity campaign queued.')
          : t('Claimable activity campaign created.')
      )
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to create activity campaign')
      )
    },
  })

  const closeMutation = useMutation({
    mutationFn: closeActivityCampaign,
    onSuccess: (campaign) => {
      queryClient.setQueryData<ActivityCampaign[]>(
        activityCampaignsQueryKey,
        (current) =>
          current?.map((item) =>
            item.activity_key === campaign.activity_key ? campaign : item
          ) ?? []
      )
      setCampaignToClose(null)
      toast.success(t('Activity campaign closed.'))
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to close activity campaign')
      )
    },
  })

  const formDisabled = createMutation.isPending || confirmation !== null
  const immediateDisabled = hasActiveImmediateCampaign || formDisabled
  const createDisabled =
    formDisabled ||
    (campaignType === 'immediate' && hasActiveImmediateCampaign) ||
    (audienceType === 'selected' && selectedUsers.length === 0)

  const openConfirmation = form.handleSubmit((values) => {
    if (values.audienceType === 'selected' && selectedUsers.length === 0) {
      form.setError('audienceType', {
        type: 'validate',
        message: t('Select at least one user.'),
      })
      return
    }
    if (values.type === 'claimable') {
      const endsAt = parseActivityCampaignEndAt(values.endsAt)
      if (!endsAt || endsAt * 1000 <= Date.now()) {
        form.setError('endsAt', {
          type: 'validate',
          message: t('The claim deadline must be in the future.'),
        })
        return
      }
    }
    setConfirmation(values)
  })

  const handleCampaignTypeChange = (values: string[]) => {
    const nextType = values.find((value) => value !== campaignType)
    if (nextType !== 'claimable' && nextType !== 'immediate') return

    form.setValue('type', nextType as ActivityCampaignType, {
      shouldDirty: true,
      shouldValidate: true,
    })
    if (nextType === 'immediate') form.clearErrors('endsAt')
  }

  const handleAudienceTypeChange = (values: string[]) => {
    const nextType = values.find((value) => value !== audienceType)
    if (nextType !== 'all' && nextType !== 'selected') return
    form.setValue('audienceType', nextType, {
      shouldDirty: true,
      shouldValidate: true,
    })
    if (nextType === 'selected') {
      form.setValue('type', 'claimable', { shouldDirty: true })
    }
  }

  const searchForUsers = async () => {
    if (!userSearch.trim()) return
    setSearchingUsers(true)
    try {
      const response = await searchUsers({
        keyword: userSearch.trim(),
        status: '1',
        page_size: 20,
      })
      setUserResults(response.data?.items ?? [])
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to search users')
      )
    } finally {
      setSearchingUsers(false)
    }
  }

  const toggleSelectedUser = (user: User) => {
    form.clearErrors('audienceType')
    setSelectedUsers((current) =>
      current.some((item) => item.id === user.id)
        ? current.filter((item) => item.id !== user.id)
        : [...current, user]
    )
  }

  return (
    <div className='space-y-5'>
      <div>
        <h4 className='text-sm font-medium'>{t('Activity campaigns')}</h4>
        <p className='text-muted-foreground text-sm'>
          {t('Publish a claimable activity or issue a frozen USD credit.')}
        </p>
      </div>

      <Form {...form}>
        <form className='grid gap-4 lg:grid-cols-2' onSubmit={openConfirmation}>
          <FormField
            control={form.control}
            name='title'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Activity title')}</FormLabel>
                <FormControl>
                  <Input
                    maxLength={128}
                    placeholder={t('Summer credit')}
                    {...field}
                    disabled={formDisabled}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='audienceType'
            render={({ field }) => (
              <FormItem className='lg:col-span-2'>
                <FormLabel>{t('Audience')}</FormLabel>
                <ToggleGroup
                  value={[field.value]}
                  onValueChange={handleAudienceTypeChange}
                  variant='outline'
                  spacing={2}
                  className='grid w-full grid-cols-2 gap-2'
                >
                  <ToggleGroupItem value='all' className='h-auto min-h-12'>
                    {t('All users')}
                  </ToggleGroupItem>
                  <ToggleGroupItem value='selected' className='h-auto min-h-12'>
                    {t('Selected users')}
                  </ToggleGroupItem>
                </ToggleGroup>
                {audienceType === 'selected' ? (
                  <div className='space-y-2 rounded-md border p-3'>
                    <div className='flex gap-2'>
                      <Input
                        value={userSearch}
                        onChange={(event) => setUserSearch(event.target.value)}
                        placeholder={t('Search users')}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter') {
                            event.preventDefault()
                            void searchForUsers()
                          }
                        }}
                      />
                      <Button
                        type='button'
                        variant='outline'
                        onClick={() => void searchForUsers()}
                        disabled={searchingUsers}
                      >
                        <Search />
                        {t('Search')}
                      </Button>
                    </div>
                    {userResults.map((user) => (
                      <button
                        type='button'
                        key={user.id}
                        className='hover:bg-muted flex w-full items-center justify-between rounded px-2 py-1 text-left text-sm'
                        onClick={() => toggleSelectedUser(user)}
                      >
                        <span>
                          {user.username}{' '}
                          {user.display_name &&
                          user.display_name !== user.username
                            ? `(${user.display_name})`
                            : ''}
                        </span>
                        <span aria-hidden='true'>
                          {selectedUsers.some((item) => item.id === user.id) ? (
                            <Check className='size-4' />
                          ) : (
                            ''
                          )}
                        </span>
                      </button>
                    ))}
                    <div className='flex flex-wrap gap-1'>
                      {selectedUsers.map((user) => (
                        <span
                          key={user.id}
                          className='bg-muted inline-flex items-center gap-1 rounded px-2 py-1 text-xs'
                        >
                          {user.username}
                          <button
                            type='button'
                            aria-label={t('Remove')}
                            onClick={() => toggleSelectedUser(user)}
                          >
                            <X className='size-3' />
                          </button>
                        </span>
                      ))}
                    </div>
                    <p className='text-muted-foreground text-xs'>
                      {t('{{count}} users selected', {
                        count: selectedUsers.length,
                      })}
                    </p>
                  </div>
                ) : null}
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='amountUSD'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Amount (USD)')}</FormLabel>
                <FormControl>
                  <InputGroup>
                    <InputGroupAddon>$</InputGroupAddon>
                    <InputGroupInput
                      type='number'
                      min='0.01'
                      step='0.01'
                      inputMode='decimal'
                      {...field}
                      disabled={formDisabled}
                    />
                  </InputGroup>
                </FormControl>
                <FormDescription>
                  {t(
                    'The internal quota is frozen when the campaign is created.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='description'
            render={({ field }) => (
              <FormItem className='lg:col-span-2'>
                <FormLabel>{t('Description')}</FormLabel>
                <FormControl>
                  <Textarea
                    maxLength={4000}
                    placeholder={t('Optional details shown to users')}
                    {...field}
                    disabled={formDisabled}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='type'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Delivery mode')}</FormLabel>
                <ToggleGroup
                  value={[field.value]}
                  onValueChange={handleCampaignTypeChange}
                  aria-label={t('Delivery mode')}
                  variant='outline'
                  spacing={2}
                  className='grid w-full grid-cols-2 gap-2'
                >
                  <ToggleGroupItem
                    value='claimable'
                    disabled={formDisabled}
                    className='h-auto min-h-12 w-full gap-2 px-3 py-2'
                  >
                    <HandCoins aria-hidden='true' />
                    {t('Claimable')}
                  </ToggleGroupItem>
                  <ToggleGroupItem
                    value='immediate'
                    disabled={immediateDisabled || audienceType === 'selected'}
                    className='h-auto min-h-12 w-full gap-2 px-3 py-2'
                  >
                    <Send aria-hidden='true' />
                    {t('Immediate credit')}
                  </ToggleGroupItem>
                </ToggleGroup>
                <FormDescription>
                  {campaignType === 'claimable'
                    ? t('Users receive the credit after they claim it.')
                    : t(
                        'Credit is issued to the frozen user audience immediately.'
                      )}
                </FormDescription>
                {hasActiveImmediateCampaign ? (
                  <p className='text-muted-foreground text-xs'>
                    {t('Another immediate campaign is currently in progress.')}
                  </p>
                ) : null}
                <FormMessage />
              </FormItem>
            )}
          />

          {campaignType === 'claimable' ? (
            <FormField
              control={form.control}
              name='endsAt'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Claim deadline')}</FormLabel>
                  <FormControl>
                    <Input
                      type='datetime-local'
                      {...field}
                      disabled={formDisabled}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Required. Users cannot claim this activity after the deadline.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : (
            <div className='flex items-end justify-end'>
              <Button type='submit' disabled={createDisabled}>
                <Gift />
                {t('Create activity')}
              </Button>
            </div>
          )}

          {campaignType === 'claimable' ? (
            <div className='flex items-end justify-end lg:col-span-2'>
              <Button type='submit' disabled={createDisabled}>
                <Gift />
                {t('Create activity')}
              </Button>
            </div>
          ) : null}
        </form>
      </Form>

      <div className='border-t pt-5'>
        <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
          <div>
            <h4 className='text-sm font-medium'>{t('Campaign history')}</h4>
            <p className='text-muted-foreground text-sm'>
              {t('Published campaigns keep their frozen audience and quota.')}
            </p>
          </div>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => void campaignsQuery.refetch()}
            disabled={campaignsQuery.isFetching}
          >
            {campaignsQuery.isFetching ? (
              <Loader2 className='animate-spin' />
            ) : (
              <Gift />
            )}
            {t('Refresh')}
          </Button>
        </div>
        <ActivityCampaignsList
          campaigns={campaignsQuery.data}
          isLoading={campaignsQuery.isLoading}
          isError={campaignsQuery.isError}
          isTaskProgressError={campaignTasksQuery.isError}
          taskById={campaignTasksQuery.data ?? {}}
          closingActivityKey={
            closeMutation.isPending ? campaignToClose?.activity_key : undefined
          }
          onClose={setCampaignToClose}
          onRetry={() => void campaignsQuery.refetch()}
          onRetryTaskProgress={() => void campaignTasksQuery.refetch()}
        />
      </div>

      <ConfirmDialog
        open={confirmation !== null}
        onOpenChange={(open) => {
          if (!open && !createMutation.isPending) setConfirmation(null)
        }}
        title={t('Publish activity campaign')}
        desc={t(
          'The campaign amount and audience are frozen after it is published.'
        )}
        confirmText={
          createMutation.isPending ? t('Publishing...') : t('Publish campaign')
        }
        handleConfirm={() => {
          if (confirmation) createMutation.mutate(confirmation)
        }}
        isLoading={createMutation.isPending}
      >
        <dl className='grid gap-2 text-sm'>
          <div className='flex justify-between gap-3'>
            <dt className='text-muted-foreground'>{t('Activity title')}</dt>
            <dd className='max-w-52 text-right font-medium break-words'>
              {confirmation?.title}
            </dd>
          </div>
          <div className='flex justify-between gap-3'>
            <dt className='text-muted-foreground'>{t('Amount')}</dt>
            <dd className='font-mono'>${confirmation?.amountUSD}</dd>
          </div>
          <div className='flex justify-between gap-3'>
            <dt className='text-muted-foreground'>{t('Delivery mode')}</dt>
            <dd>
              {confirmation?.type === 'immediate'
                ? t('Immediate credit')
                : t('Claimable')}
            </dd>
          </div>
          <div className='flex justify-between gap-3'>
            <dt className='text-muted-foreground'>{t('Audience')}</dt>
            <dd>
              {confirmation?.audienceType === 'selected'
                ? t('{{count}} selected users', { count: selectedUsers.length })
                : t('All users')}
            </dd>
          </div>
          {confirmation?.type === 'claimable' ? (
            <div className='flex justify-between gap-3'>
              <dt className='text-muted-foreground'>{t('Claim deadline')}</dt>
              <dd className='text-right'>{confirmation.endsAt}</dd>
            </div>
          ) : null}
        </dl>
      </ConfirmDialog>

      <ConfirmDialog
        open={campaignToClose !== null}
        onOpenChange={(open) => {
          if (!open && !closeMutation.isPending) setCampaignToClose(null)
        }}
        title={t('Close activity campaign')}
        desc={t(
          'Close "{{title}}" now? Users who have not claimed it will no longer be able to do so.',
          { title: campaignToClose?.title ?? '' }
        )}
        confirmText={
          closeMutation.isPending ? t('Closing...') : t('Close activity')
        }
        destructive
        handleConfirm={() => {
          if (campaignToClose) {
            closeMutation.mutate(campaignToClose.activity_key)
          }
        }}
        isLoading={closeMutation.isPending}
      />
    </div>
  )
}
