import { useQuery } from '@tanstack/react-query'
import {
  Check,
  Link2Off,
  Pause,
  Play,
  RefreshCw,
  ShieldCheck,
  X,
} from 'lucide-react'
import { useDeferredValue, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { searchUsers } from '@/features/users/api'

import type { SecuritySettings } from '../types'
import {
  addRiskExemption,
  approveRiskAccountReview,
  detachRiskAccount,
  getRiskAccountBackfill,
  getRiskAccountActions,
  getRiskAccountReviews,
  getRiskExemptions,
  pauseRiskAccountBackfill,
  rejectRiskAccountReview,
  removeRiskExemption,
  resumeRiskAccountBackfill,
  startRiskAccountBackfill,
} from './risk-api'

type RiskAccountSectionProps = { defaultValues?: SecuritySettings }

function optionalNumber(value: string) {
  const parsed = Number(value)
  return value.trim() && Number.isFinite(parsed) && parsed >= 0
    ? parsed
    : undefined
}

function optionalTimestamp(value: string) {
  if (!value) return undefined
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? Math.floor(timestamp / 1000) : undefined
}

function scanStatusLabel(status: string, t: (key: string) => string) {
  const labels: Record<string, string> = {
    pending: t('Pending'),
    running: t('Running'),
    pause_requested: t('Pause requested'),
    paused: t('Paused'),
    succeeded: t('Completed'),
    failed: t('Failed'),
  }
  return labels[status] || status
}

function reviewStatusLabel(status: string, t: (key: string) => string) {
  const labels: Record<string, string> = {
    pending: t('Pending review'),
    approved: t('Approved'),
    rejected: t('Rejected'),
    failed: t('Failed'),
  }
  return labels[status] || status
}

function actionLabel(action: string, t: (key: string) => string) {
  const labels: Record<string, string> = {
    disable_subaccount: t('Automatic subaccount ban'),
    unlink_restore: t('Manual relationship restore'),
  }
  return labels[action] || t('Other risk action')
}

function evidenceLabel(evidence: string, t: (key: string) => string) {
  const labels: Record<string, string> = {
    exact_identity: t('Matching account identity'),
    exact_oauth_identity: t('Matching OAuth identity'),
    same_ip_user_agent_24h: t('Same IP and browser within 24 hours'),
    inviter_reward_burst: t('Invitation reward burst'),
    strong_identity_or_device: t('Strong identity or device match'),
    'historical approval would demote existing main account': t('Existing main account requires merge review'),
  }
  return labels[evidence] || evidence || t('Strong association')
}

function reviewKindLabel(kind: string, t: (key: string) => string) {
  return kind === 'cluster_merge'
    ? t('Cluster merge review')
    : t('Historical account review')
}

export function RiskAccountSection(_props: RiskAccountSectionProps) {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const [selectedUser, setSelectedUser] = useState('')
  const [reason, setReason] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [minUserId, setMinUserId] = useState('')
  const [maxUserId, setMaxUserId] = useState('')
  const [createdFrom, setCreatedFrom] = useState('')
  const [createdTo, setCreatedTo] = useState('')
  const [detachUserId, setDetachUserId] = useState('')
  const [detachKeyword, setDetachKeyword] = useState('')
  const [detachReason, setDetachReason] = useState('')
  const [reviewStatus, setReviewStatus] = useState('pending')
  const [actionPage, setActionPage] = useState(1)
  const [busy, setBusy] = useState(false)
  const deferredDetachKeyword = useDeferredValue(detachKeyword)

  const usersQuery = useQuery({
    queryKey: ['risk-exemption-users', keyword],
    queryFn: () =>
      searchUsers({
        keyword,
        p: 1,
        page_size: 100,
        sort_by: 'username',
        sort_order: 'asc',
      }),
    staleTime: 30 * 1000,
  })
  const exemptionsQuery = useQuery({
    queryKey: ['risk-exemptions'],
    queryFn: getRiskExemptions,
    staleTime: 10 * 1000,
  })
  const backfillQuery = useQuery({
    queryKey: ['risk-backfill-latest'],
    queryFn: () => getRiskAccountBackfill(),
    refetchInterval: 2500,
  })
  const reviewsQuery = useQuery({
    queryKey: ['risk-account-reviews', reviewStatus],
    queryFn: () => getRiskAccountReviews(reviewStatus),
    refetchInterval: 5000,
  })
  const detachUsersQuery = useQuery({
    queryKey: ['risk-restore-users', deferredDetachKeyword],
    queryFn: () =>
      searchUsers({
        keyword: deferredDetachKeyword,
        p: 1,
        page_size: 50,
        sort_by: 'username',
        sort_order: 'asc',
      }),
    staleTime: 15 * 1000,
  })
  const actionsQuery = useQuery({
    queryKey: ['risk-account-actions', actionPage],
    queryFn: () => getRiskAccountActions({ page: actionPage, page_size: 20 }),
    refetchInterval: 5000,
  })

  const exemptions = exemptionsQuery.data?.data ?? []
  const activeExemptions = exemptions.filter(
    (row) => row.enabled && (!row.expires_at || row.expires_at > Math.floor(Date.now() / 1000))
  )
  const users = usersQuery.data?.data?.items ?? []
  const run = backfillQuery.data?.data
  const reviews = reviewsQuery.data?.data?.items ?? []
  const restoreUsers = [...(detachUsersQuery.data?.data?.items ?? [])].sort(
    (a, b) => Number(b.status === 2) - Number(a.status === 2) || a.username.localeCompare(b.username)
  )
  const actions = actionsQuery.data?.data?.items ?? []
  const actionTotal = actionsQuery.data?.data?.total ?? 0
  const selected = users.find((user) => String(user.id) === selectedUser)
  const scanActive =
    run?.status === 'pending' ||
    run?.status === 'running' ||
    run?.status === 'pause_requested'
  const progress = run?.total_users
    ? Math.min(100, Math.round((run.processed_users * 100) / run.total_users))
    : 0

  const refresh = () => {
    void exemptionsQuery.refetch()
  }

  const add = async () => {
    if (!selectedUser) return
    setBusy(true)
    try {
      const result = await addRiskExemption({
        user_id: Number(selectedUser),
        reason: reason.trim(),
        expires_at: optionalTimestamp(expiresAt),
      })
      if (!result.success) {
        throw new Error(result.message || t('Failed to save whitelist'))
      }
      toast.success(t('Account added to risk whitelist'))
      setSelectedUser('')
      setReason('')
      setExpiresAt('')
      refresh()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save whitelist')
      )
    } finally {
      setBusy(false)
    }
  }

  const remove = async (userId: number) => {
    setBusy(true)
    try {
      const result = await removeRiskExemption(userId)
      if (!result.success) {
        throw new Error(result.message || t('Failed to remove whitelist'))
      }
      toast.success(t('Account removed from risk whitelist'))
      refresh()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to remove whitelist')
      )
    } finally {
      setBusy(false)
    }
  }

  const startBackfill = async () => {
    setBusy(true)
    try {
      const result = await startRiskAccountBackfill({
        min_user_id: optionalNumber(minUserId),
        max_user_id: optionalNumber(maxUserId),
        created_from: optionalTimestamp(createdFrom),
        created_to: optionalTimestamp(createdTo),
        page_size: 200,
      })
      if (!result.success) {
        throw new Error(result.message || t('Historical scan failed'))
      }
      toast.success(
        result.data?.created
          ? t('Historical preview scan started')
          : t('A historical preview scan is already running')
      )
      void backfillQuery.refetch()
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Historical scan failed')
      )
    } finally {
      setBusy(false)
    }
  }

  const pauseBackfill = async () => {
    if (!run) return
    setBusy(true)
    try {
      const result = await pauseRiskAccountBackfill(run.run_id)
      if (!result.success) {
        throw new Error(result.message || t('Failed to pause historical scan'))
      }
      toast.success(t('Historical scan pause requested'))
      void backfillQuery.refetch()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to pause historical scan')
      )
    } finally {
      setBusy(false)
    }
  }

  const resumeBackfill = async () => {
    if (!run) return
    setBusy(true)
    try {
      const result = await resumeRiskAccountBackfill(run.run_id)
      if (!result.success) {
        throw new Error(result.message || t('Failed to resume historical scan'))
      }
      toast.success(t('Historical scan resumed'))
      void backfillQuery.refetch()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to resume historical scan')
      )
    } finally {
      setBusy(false)
    }
  }

  const resolveReview = async (reviewId: number, approve: boolean) => {
    setBusy(true)
    try {
      const result = approve
        ? await approveRiskAccountReview(reviewId)
        : await rejectRiskAccountReview(reviewId)
      if (!result.success) {
        throw new Error(result.message || t('Failed to resolve risk review'))
      }
      toast.success(
        approve ? t('Linked accounts enforced') : t('Risk review rejected')
      )
      void reviewsQuery.refetch()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to resolve risk review')
      )
    } finally {
      setBusy(false)
    }
  }

  const detach = async () => {
    const userId = optionalNumber(detachUserId)
    if (!userId) return
    setBusy(true)
    try {
      const result = await detachRiskAccount(userId, detachReason.trim())
      if (!result.success) {
        throw new Error(result.message || t('Failed to restore account'))
      }
      toast.success(t('Account relationship removed and account restored'))
      setDetachUserId('')
      setDetachKeyword('')
      setDetachReason('')
      void reviewsQuery.refetch()
      void actionsQuery.refetch()
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to restore account')
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card data-card-hover='false'>
      <CardHeader className='flex flex-row items-start gap-3'>
        <div className='bg-primary/10 text-primary rounded-lg p-2'>
          <ShieldCheck className='h-4 w-4' />
        </div>
        <div>
          <div className='font-semibold'>{t('Account link risk control')}</div>
          <div className='text-muted-foreground text-sm'>
            {t(
              'Administrators and whitelisted accounts are excluded from account linking.'
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-7'>
        <section className='space-y-3'>
          <div>
            <div className='text-sm font-medium'>{t('Risk whitelist')}</div>
            <div className='text-muted-foreground text-xs'>
              {t(
                'Adding an account removes its existing relationship and restores access.'
              )}
            </div>
          </div>
          <div className='grid gap-2 lg:grid-cols-[1fr_1fr_1fr_auto]'>
            <Input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('Search users')}
            />
            <Select
              value={selectedUser}
              onValueChange={(value) => value !== null && setSelectedUser(value)}
            >
              <SelectTrigger>
                <SelectValue
                  placeholder={
                    selected
                      ? `${selected.username} (${selected.id})`
                      : t('Select a user')
                  }
                />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {users
                    .filter((user) => user.role < 10)
                    .map((user) => (
                      <SelectItem key={user.id} value={String(user.id)}>
                        {user.username} ({user.id})
                      </SelectItem>
                    ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <Input
              type='datetime-local'
              value={expiresAt}
              onChange={(event) => setExpiresAt(event.target.value)}
              aria-label={t('Whitelist expiration')}
            />
            <Button
              type='button'
              disabled={!selectedUser || busy}
              onClick={add}
            >
              {t('Add whitelist')}
            </Button>
          </div>
          <Input
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder={t('Whitelist reason (optional)')}
            maxLength={255}
          />
          <div className='space-y-2'>
            {activeExemptions.map((row) => {
              const user = users.find((item) => item.id === row.user_id)
              return (
                <div
                  key={row.user_id}
                  className='flex items-center justify-between rounded-md border px-3 py-2 text-sm'
                >
                  <span>
                    {user
                      ? `${user.username} (${user.id})`
                      : `${t('User ID')} ${row.user_id}`}
                    {row.reason ? ` · ${row.reason}` : ''}
                  </span>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    disabled={busy}
                    onClick={() => remove(row.user_id)}
                    aria-label={t('Remove whitelist')}
                  >
                    <X className='h-4 w-4' />
                  </Button>
                </div>
              )
            })}
            {activeExemptions.length === 0 && (
              <div className='text-muted-foreground text-sm'>
                {t('No whitelisted accounts')}
              </div>
            )}
          </div>
        </section>

        <section className='space-y-3 border-t pt-6'>
          <div>
            <div className='text-sm font-medium'>
              {t('Historical account preview')}
            </div>
            <div className='text-muted-foreground text-xs'>
              {t(
                'The scan only creates a report. Accounts are disabled after an administrator approves a candidate group.'
              )}
            </div>
          </div>
          <div className='grid gap-2 sm:grid-cols-2 lg:grid-cols-4'>
            <div className='space-y-1'>
              <div className='text-muted-foreground text-xs'>{t('Minimum user ID')}</div>
              <Input type='number' min={0} value={minUserId} onChange={(event) => setMinUserId(event.target.value)} />
            </div>
            <div className='space-y-1'>
              <div className='text-muted-foreground text-xs'>{t('Maximum user ID')}</div>
              <Input type='number' min={0} value={maxUserId} onChange={(event) => setMaxUserId(event.target.value)} />
            </div>
            <div className='space-y-1'>
              <div className='text-muted-foreground text-xs'>{t('Registration start time')}</div>
              <Input type='datetime-local' value={createdFrom} onChange={(event) => setCreatedFrom(event.target.value)} />
            </div>
            <div className='space-y-1'>
              <div className='text-muted-foreground text-xs'>{t('Registration end time')}</div>
              <Input type='datetime-local' value={createdTo} onChange={(event) => setCreatedTo(event.target.value)} />
            </div>
          </div>
          <div className='text-muted-foreground text-xs'>
            {t('Registration date range limits the historical scan; leave blank for no limit.')}
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <Button
              type='button'
              variant='outline'
              disabled={busy || scanActive}
              onClick={startBackfill}
            >
              <RefreshCw className='mr-1 h-4 w-4' />
              {t('Start preview scan')}
            </Button>
            {scanActive && run?.status !== 'pause_requested' && (
              <Button
                type='button'
                variant='outline'
                disabled={busy}
                onClick={pauseBackfill}
              >
                <Pause className='mr-1 h-4 w-4' />
                {t('Pause scan')}
              </Button>
            )}
            {(run?.status === 'paused' || run?.status === 'failed') && (
              <Button
                type='button'
                variant='outline'
                disabled={busy}
                onClick={resumeBackfill}
              >
                <Play className='mr-1 h-4 w-4' />
                {t('Resume scan')}
              </Button>
            )}
            {run && (
              <Badge variant={run.status === 'failed' ? 'destructive' : 'secondary'}>
                {t('Scan status')}: {scanStatusLabel(run.status, t)}
              </Badge>
            )}
          </div>
          {run && (
            <div className='space-y-1'>
              <div className='bg-muted h-2 overflow-hidden rounded-full'>
                <div
                  className='bg-primary h-full transition-all'
                  style={{ width: `${progress}%` }}
                />
              </div>
              <div className='text-muted-foreground text-xs'>
                {run.processed_users}/{run.total_users} ·{' '}
                {t('{{count}} candidate groups', {
                  count: run.candidate_groups,
                })}
              </div>
              {run.error && (
                <div className='text-destructive text-xs'>{run.error}</div>
              )}
            </div>
          )}
        </section>

        <section className='space-y-3 border-t pt-6'>
          <div className='flex flex-wrap items-start justify-between gap-2'>
            <div>
              <div className='text-sm font-medium'>{t('Risk review reports')}</div>
            <div className='text-muted-foreground text-xs'>
              {t(
                'Approving keeps the earliest eligible account and disables the remaining accounts.'
              )}
            </div>
            </div>
            <Select value={reviewStatus} onValueChange={(value) => value && setReviewStatus(value)}>
              <SelectTrigger className='w-40'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='pending'>{t('Pending review')}</SelectItem>
                <SelectItem value='approved'>{t('Approved')}</SelectItem>
                <SelectItem value='rejected'>{t('Rejected')}</SelectItem>
                <SelectItem value='failed'>{t('Failed')}</SelectItem>
                <SelectItem value='all'>{t('All reports')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className='space-y-2'>
            {reviews.map((review) => (
              <div key={review.id} className='space-y-2 rounded-md border p-3'>
                <div className='flex flex-wrap items-center justify-between gap-2'>
                  <div className='text-sm font-medium'>
                    #{review.id} · {reviewKindLabel(review.kind, t)} ·{' '}
                    {reviewStatusLabel(review.status, t)} ·{' '}
                    {evidenceLabel(review.evidence, t)}
                  </div>
                  {review.status === 'pending' && (
                    <div className='flex gap-2'>
                      <Button
                        type='button'
                        size='sm'
                        disabled={busy}
                        onClick={() => resolveReview(review.id, true)}
                      >
                        <Check className='mr-1 h-4 w-4' />
                        {t('Approve and enforce')}
                      </Button>
                      <Button
                        type='button'
                        size='sm'
                        variant='outline'
                        disabled={busy}
                        onClick={() => resolveReview(review.id, false)}
                      >
                        <X className='mr-1 h-4 w-4' />
                        {t('Reject')}
                      </Button>
                    </div>
                  )}
                </div>
                <div className='flex flex-wrap gap-2'>
                  {review.members.map((member) => (
                    <Badge key={member.id} variant='secondary'>
                      {member.username} ({member.id}) ·{' '}
                      {member.relation === 'main'
                        ? t('Main account')
                        : t('Subaccount')}
                    </Badge>
                  ))}
                </div>
                {review.status !== 'pending' && (
                  <div className='text-muted-foreground text-xs'>
                    {review.resolved_by_username || (review.resolved_by > 0 ? `${t('User ID')} ${review.resolved_by}` : t('System automatic'))} ·{' '}
                    {review.resolved_at > 0 ? new Date(review.resolved_at * 1000).toLocaleString() : ''} ·{' '}
                    {review.resolution || t('No resolution note')}
                  </div>
                )}
              </div>
            ))}
            {reviews.length === 0 && (
              <div className='text-muted-foreground text-sm'>
                {reviewStatus === 'pending'
                  ? t('No pending risk reviews')
                  : t('No processed risk reports')}
              </div>
            )}
          </div>
        </section>

        <section className='space-y-3 border-t pt-6'>
          <div>
            <div className='text-sm font-medium'>{t('Automatic enforcement records')}</div>
            <div className='text-muted-foreground text-xs'>
              {t('These records show actions already taken by the system or an administrator.')}
            </div>
          </div>
          <div className='space-y-2'>
            {actions.map((item) => (
              <div key={item.id} className='space-y-1 rounded-md border p-3 text-sm'>
                <div className='flex flex-wrap items-center justify-between gap-2'>
                  <div className='font-medium'>{actionLabel(item.action, t)} · {item.username || `${t('User ID')} ${item.user_id}`}</div>
                  <Badge variant={item.source === 'system' ? 'destructive' : 'secondary'}>
                    {item.source === 'system' ? t('System automatic') : t('Administrator action')}
                  </Badge>
                </div>
                <div className='text-muted-foreground text-xs'>
                  {t('Main account')}: {item.main_username || `${t('User ID')} ${item.main_user_id}`} · {t('Rule')}: {evidenceLabel(item.rule, t)} · {new Date(item.created_at * 1000).toLocaleString()}
                </div>
              </div>
            ))}
            {actions.length === 0 && <div className='text-muted-foreground text-sm'>{t('No automatic enforcement records')}</div>}
            {actionTotal > 20 && (
              <div className='flex items-center justify-between pt-2 text-xs'>
                <span className='text-muted-foreground'>{t('{{count}} total records', { count: actionTotal })}</span>
                <div className='flex gap-2'>
                  <Button type='button' size='sm' variant='outline' disabled={actionPage <= 1} onClick={() => setActionPage((page) => page - 1)}>{t('Previous page')}</Button>
                  <Button type='button' size='sm' variant='outline' disabled={actionPage * 20 >= actionTotal} onClick={() => setActionPage((page) => page + 1)}>{t('Next page')}</Button>
                </div>
              </div>
            )}
          </div>
        </section>

        <section className='space-y-3 border-t pt-6'>
          <div>
            <div className='text-sm font-medium'>
              {t('Restore a false positive')}
            </div>
            <div className='text-muted-foreground text-xs'>
              {t(
                'Restoring a main account removes the entire cluster; restoring a subaccount removes only that account.'
              )}
            </div>
          </div>
          <div className='grid gap-2 sm:grid-cols-[1fr_1fr_auto]'>
            <div className='space-y-1'>
              <Input
                value={detachKeyword}
                onChange={(event) => setDetachKeyword(event.target.value)}
                placeholder={t('Search accounts to restore')}
              />
              <Select
                value={detachUserId}
                onValueChange={(value) => value && setDetachUserId(value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t('Select an account to restore')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {restoreUsers.map((user) => (
                      <SelectItem key={user.id} value={String(user.id)}>
                        {user.username} ({user.id}) ·{' '}
                        {user.status === 2 ? t('Disabled') : t('Enabled')}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <Input
              value={detachReason}
              onChange={(event) => setDetachReason(event.target.value)}
              placeholder={t('Restore reason (optional)')}
              maxLength={255}
            />
            <Button
              type='button'
              variant='outline'
              disabled={!optionalNumber(detachUserId) || busy}
              onClick={detach}
            >
              <Link2Off className='mr-1 h-4 w-4' />
              {t('Remove link and restore')}
            </Button>
          </div>
        </section>

      </CardContent>
    </Card>
  )
}
