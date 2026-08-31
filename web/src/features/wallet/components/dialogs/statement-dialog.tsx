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
import { Download, Eye, FileText, History, Loader2, Search } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useIsAdmin } from '@/hooks/use-admin'
import { useSystemConfig } from '@/hooks/use-system-config'

import {
  downloadStatementPdf,
  generateAdminStatement,
  generateCurrentStatement,
  getAdminStatementHistory,
  getAdminStatementMonthly,
  getPreviousStatement,
} from '../../api'
import type { ConsumptionStatement, StatementMonthlySummary } from '../../types'

interface StatementDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const adminPageSize = 20

function formatCents(cents: number) {
  return `¥${(cents / 100).toFixed(2)}`
}

function formatTime(timestamp: number) {
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(new Date(timestamp * 1000))
}

function currentMonth() {
  const parts = new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
  }).formatToParts(new Date())
  const year = parts.find((part) => part.type === 'year')?.value
  const month = parts.find((part) => part.type === 'month')?.value
  return `${year}-${month}`
}

function StatementPreview({
  statement,
  downloading,
  loadingPrevious,
  onDownload,
  onBack,
  onPrevious,
}: {
  statement: ConsumptionStatement
  downloading: boolean
  loadingPrevious: boolean
  onDownload: () => void
  onBack?: () => void
  onPrevious?: () => void
}) {
  const { t } = useTranslation()
  const { logo } = useSystemConfig()
  const snapshot = statement.snapshot
  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        {onBack ? (
          <Button variant='outline' size='sm' onClick={onBack}>
            {t('Back to statement center')}
          </Button>
        ) : (
          <div />
        )}
        <div className='flex flex-wrap justify-end gap-2'>
          {onPrevious && (
            <Button
              variant='outline'
              onClick={onPrevious}
              disabled={loadingPrevious}
              className='gap-2'
            >
              {loadingPrevious ? (
                <Loader2 className='h-4 w-4 animate-spin' />
              ) : (
                <History className='h-4 w-4' />
              )}
              {t('View previous month statement')}
            </Button>
          )}
          <Button onClick={onDownload} disabled={downloading} className='gap-2'>
            {downloading ? (
              <Loader2 className='h-4 w-4 animate-spin' />
            ) : (
              <Download className='h-4 w-4' />
            )}
            {t('Export PDF')}
          </Button>
        </div>
      </div>

      <div className='relative overflow-hidden rounded-xl border bg-white p-4 text-neutral-900 shadow-sm sm:p-7'>
        <div className='pointer-events-none absolute inset-0 flex -rotate-12 flex-col items-center justify-center gap-8 text-4xl font-bold text-neutral-900/[0.035] sm:text-6xl'>
          <span>非税务发票</span>
          <span>仅供消费对账</span>
        </div>
        <div className='relative space-y-6'>
          <div className='flex items-start justify-between gap-4 border-b-2 border-neutral-900 pb-4'>
            <div className='flex items-center gap-3'>
              <img
                src={logo || '/logo.png'}
                alt=''
                className='h-10 w-10 rounded-lg object-contain'
              />
              <div>
                <div className='font-semibold'>{snapshot.issuer.name}</div>
                <div className='text-xs text-neutral-500'>
                  {snapshot.issuer.website}
                </div>
              </div>
            </div>
            <div className='text-right'>
              <h3 className='text-xl font-bold'>
                {t('Consumption Statement')}
              </h3>
            </div>
          </div>

          <div className='grid gap-4 text-sm sm:grid-cols-3'>
            <div>
              <div className='mb-1 font-semibold'>{t('Issuer')}</div>
              <div>{snapshot.issuer.name}</div>
              <div>{snapshot.issuer.contact_email || '-'}</div>
              <div className='whitespace-pre-wrap'>
                {snapshot.issuer.address || '-'}
              </div>
            </div>
            <div>
              <div className='mb-1 font-semibold'>
                {t('Statement recipient')}
              </div>
              <div>
                {t('User ID')}: {snapshot.recipient.user_id}
              </div>
              <div>
                {t('Username')}: {snapshot.recipient.username}
              </div>
              <div>
                {t('Email')}: {snapshot.recipient.email || '-'}
              </div>
              {snapshot.recipient.billing_title && (
                <div>
                  {t('Reconciliation title')}:{' '}
                  {snapshot.recipient.billing_title}
                </div>
              )}
              {snapshot.recipient.billing_address && (
                <div className='whitespace-pre-wrap'>
                  {t('Contact address')}: {snapshot.recipient.billing_address}
                </div>
              )}
            </div>
            <div>
              <div className='mb-1 font-semibold'>
                {t('Statement information')}
              </div>
              <div>
                {t('Statement number')}: {statement.statement_no}
              </div>
              <div>
                {t('Period start')}: {formatTime(snapshot.period_start)}
              </div>
              <div>
                {t('Period end')}: {formatTime(snapshot.period_end)}
              </div>
              <div>
                {t('Generated at')}: {formatTime(snapshot.generated_at)}
              </div>
              <div>
                {t('Version type')}:{' '}
                {snapshot.is_final
                  ? t('System monthly final')
                  : t('Immutable snapshot')}
              </div>
            </div>
          </div>

          {snapshot.recipient.user_supplied && (
            <div className='rounded-md border border-amber-300 bg-amber-50 p-2 text-xs text-amber-900'>
              {t(
                'The reconciliation title and contact address are supplied by the user and are not invoice or tax information.'
              )}
            </div>
          )}

          <div>
            <h4 className='mb-2 font-semibold'>{t('Model token summary')}</h4>
            <div className='overflow-x-auto'>
              <table className='w-full min-w-[560px] border-collapse text-sm'>
                <thead className='bg-neutral-100'>
                  <tr>
                    <th className='border p-2 text-left'>{t('Model')}</th>
                    <th className='border p-2 text-right'>
                      {t('Input tokens')}
                    </th>
                    <th className='border p-2 text-right'>
                      {t('Output tokens')}
                    </th>
                    <th className='border p-2 text-right'>
                      {t('Billing records')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {snapshot.tokens.length ? (
                    snapshot.tokens.map((item) => (
                      <tr key={item.model_name}>
                        <td className='border p-2'>{item.model_name}</td>
                        <td className='border p-2 text-right'>
                          {item.input_tokens.toLocaleString()}
                        </td>
                        <td className='border p-2 text-right'>
                          {item.output_tokens.toLocaleString()}
                        </td>
                        <td className='border p-2 text-right'>
                          {item.record_count.toLocaleString()}
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td
                        colSpan={4}
                        className='border p-4 text-center text-neutral-500'
                      >
                        {t('No token usage in this period')}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          <div className='grid gap-4 lg:grid-cols-2'>
            <div>
              <h4 className='mb-2 font-semibold'>
                {t('Redemption bookkeeping')}
              </h4>
              <div className='space-y-1 text-sm'>
                {snapshot.redemptions.length ? (
                  snapshot.redemptions.map((item) => (
                    <div
                      key={item.record_id}
                      className='flex justify-between gap-3 border-b py-1.5'
                    >
                      <span>
                        #{item.record_id} · {item.category}
                      </span>
                      <span className='font-medium'>
                        {formatCents(item.amount_cents)}
                      </span>
                    </div>
                  ))
                ) : (
                  <div className='text-neutral-500'>
                    {t('No priced redemptions')}
                  </div>
                )}
              </div>
            </div>
            <div>
              <h4 className='mb-2 font-semibold'>{t('CNY wallet top-ups')}</h4>
              <div className='space-y-1 text-sm'>
                {snapshot.topups.length ? (
                  snapshot.topups.map((item) => (
                    <div
                      key={item.record_id}
                      className='flex justify-between gap-3 border-b py-1.5'
                    >
                      <span>
                        #{item.record_id} · {formatTime(item.completed_at)}
                      </span>
                      <span className='font-medium'>
                        {formatCents(item.amount_cents)}
                      </span>
                    </div>
                  ))
                ) : (
                  <div className='text-neutral-500'>
                    {t('No eligible CNY top-ups')}
                  </div>
                )}
              </div>
            </div>
            <div>
              <h4 className='mb-2 font-semibold'>{t('Subscription purchases')}</h4>
              <div className='space-y-1 text-sm'>
                {snapshot.subscriptions?.length ? (
                  snapshot.subscriptions.map((item) => (
                    <div key={item.record_id} className='flex justify-between gap-3 border-b py-1.5'>
                      <span>#{item.record_id} · {item.plan_title} · {formatTime(item.paid_at)}</span>
                      <span className='font-medium'>{formatCents(item.amount_cents)}</span>
                    </div>
                  ))
                ) : (
                  <div className='text-neutral-500'>{t('No subscription purchases')}</div>
                )}
              </div>
            </div>
          </div>

          {(snapshot.warnings.unpriced_redemptions > 0 ||
            snapshot.warnings.unknown_currency_topups > 0) && (
            <div className='rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900'>
              {t('Excluded with warning')}: {t('Unpriced redemptions')}{' '}
              {snapshot.warnings.unpriced_redemptions};{' '}
              {t('Unknown-currency top-ups')}{' '}
              {snapshot.warnings.unknown_currency_topups}.
            </div>
          )}

          <div className='ml-auto max-w-md space-y-2 text-sm'>
            <div className='flex justify-between border-b pb-1'>
              <span>{t('Redemption bookkeeping total')}</span>
              <strong>{formatCents(snapshot.redemption_total_cents)}</strong>
            </div>
            <div className='flex justify-between border-b pb-1'>
              <span>{t('CNY wallet top-up total')}</span>
              <strong>{formatCents(snapshot.topup_total_cents)}</strong>
            </div>
            <div className='flex justify-between border-b pb-1'>
              <span>{t('Subscription purchase total')}</span>
              <strong>{formatCents(snapshot.subscription_total_cents || 0)}</strong>
            </div>
            <div className='flex justify-between border-b-2 border-neutral-900 pb-2 text-base'>
              <span>{t('RMB bookkeeping total')}</span>
              <strong>{formatCents(snapshot.total_cents)}</strong>
            </div>
          </div>

          <div>
            <h4 className='mb-2 font-semibold'>{t('Compliance notice')}</h4>
            <ol className='list-decimal space-y-1 pl-5 text-xs leading-5 text-neutral-600'>
              {snapshot.disclaimers.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ol>
          </div>
          <div className='border-t pt-2 font-mono text-[10px] break-all text-neutral-500'>
            SHA-256: {statement.content_hash}
          </div>
        </div>
      </div>
    </div>
  )
}

export function StatementDialog({ open, onOpenChange }: StatementDialogProps) {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const [statement, setStatement] = useState<ConsumptionStatement | null>(null)
  const [billingTitle, setBillingTitle] = useState('')
  const [billingAddress, setBillingAddress] = useState('')
  const [loading, setLoading] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const [loadingPrevious, setLoadingPrevious] = useState(false)
  const [tab, setTab] = useState<'monthly' | 'history'>('monthly')
  const [month, setMonth] = useState(currentMonth)
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [monthly, setMonthly] = useState<StatementMonthlySummary[]>([])
  const [history, setHistory] = useState<ConsumptionStatement[]>([])

  const loadAdmin = useCallback(async () => {
    if (!isAdmin || !open) return
    setLoading(true)
    try {
      if (tab === 'monthly') {
        const response = await getAdminStatementMonthly({
          month,
          keyword,
          page,
          pageSize: adminPageSize,
        })
        if (!response.success) throw new Error(response.message)
        setMonthly(response.data?.items || [])
        setTotal(response.data?.total || 0)
      } else {
        const response = await getAdminStatementHistory({
          month,
          keyword,
          page,
          pageSize: adminPageSize,
        })
        if (!response.success) throw new Error(response.message)
        setHistory(response.data?.items || [])
        setTotal(response.data?.total || 0)
      }
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('Failed to load statement center')
      )
    } finally {
      setLoading(false)
    }
  }, [isAdmin, keyword, month, open, page, t, tab])

  useEffect(() => {
    if (open && isAdmin && !statement) void loadAdmin()
  }, [isAdmin, loadAdmin, open, statement])

  useEffect(() => {
    if (!open) {
      setStatement(null)
      setBillingTitle('')
      setBillingAddress('')
    }
  }, [open])

  const generateSelf = async () => {
    setLoading(true)
    try {
      const response = await generateCurrentStatement({
        billing_title: billingTitle.trim() || undefined,
        billing_address: billingAddress.trim() || undefined,
      })
      if (!response.success || !response.data) throw new Error(response.message)
      setStatement(response.data)
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('Failed to generate statement')
      )
    } finally {
      setLoading(false)
    }
  }

  const generateForUser = async (userId: number) => {
    setLoading(true)
    try {
      const response = await generateAdminStatement({ user_id: userId, month })
      if (!response.success || !response.data) throw new Error(response.message)
      setStatement(response.data)
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('Failed to generate statement')
      )
    } finally {
      setLoading(false)
    }
  }

  const viewPreviousStatement = async () => {
    setLoadingPrevious(true)
    try {
      const response = await getPreviousStatement()
      if (!response.success || !response.data) throw new Error(response.message)
      setStatement(response.data)
    } catch {
      toast.error(t('Previous month statement is not ready yet'))
    } finally {
      setLoadingPrevious(false)
    }
  }

  const downloadPdf = async () => {
    if (!statement) return
    setDownloading(true)
    try {
      const blob = await downloadStatementPdf(statement.id)
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = `${statement.statement_no}.pdf`
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(url)
    } catch {
      toast.error(t('Failed to export PDF'))
    } finally {
      setDownloading(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={isAdmin ? t('Statement Center') : t('Export Statement')}
      description={t(
        'Statements are immutable consumption reconciliation records, not tax invoices.'
      )}
      contentClassName='flex max-h-[calc(100dvh-2rem)] flex-col max-sm:w-screen max-sm:max-w-none max-sm:rounded-none sm:max-w-6xl'
      bodyClassName='min-h-0 overflow-y-auto'
      contentHeight='auto'
    >
      {statement && (
        <StatementPreview
          statement={statement}
          downloading={downloading}
          loadingPrevious={loadingPrevious}
          onDownload={downloadPdf}
          onBack={isAdmin ? () => setStatement(null) : undefined}
          onPrevious={
            !isAdmin && statement.source !== 'system_monthly'
              ? viewPreviousStatement
              : undefined
          }
        />
      )}
      {!statement && isAdmin && (
        <div className='space-y-4'>
          <div className='flex flex-wrap gap-2'>
            <Button
              variant={tab === 'monthly' ? 'default' : 'outline'}
              size='sm'
              onClick={() => {
                if (!month) setMonth(currentMonth())
                setPage(1)
                setTab('monthly')
              }}
            >
              {t('View by month')}
            </Button>
            <Button
              variant={tab === 'history' ? 'default' : 'outline'}
              size='sm'
              onClick={() => {
                setMonth('')
                setPage(1)
                setTab('history')
              }}
            >
              {t('Generated versions')}
            </Button>
          </div>
          <p className='text-muted-foreground text-xs'>
            {t(
              'System monthly statements are finalized at 00:00 on the first day of the next month in Asia/Shanghai.'
            )}
          </p>
          <div className='grid gap-2 sm:grid-cols-[170px_1fr_auto]'>
            <Input
              type='month'
              value={month}
              max={currentMonth()}
              onChange={(event) => {
                setMonth(event.target.value)
                setPage(1)
              }}
            />
            <Input
              value={keyword}
              onChange={(event) => {
                setKeyword(event.target.value)
                setPage(1)
              }}
              placeholder={t(
                'Search user ID, username, email or statement number'
              )}
            />
            <Button variant='outline' onClick={loadAdmin} className='gap-2'>
              <Search className='h-4 w-4' />
              {t('Search')}
            </Button>
          </div>
          {loading && (
            <div className='flex min-h-48 items-center justify-center'>
              <Loader2 className='h-6 w-6 animate-spin' />
            </div>
          )}
          {!loading && tab === 'monthly' && (
            <div className='overflow-x-auto rounded-lg border'>
              <table className='w-full min-w-[900px] text-sm'>
                <thead className='bg-muted/60'>
                  <tr>
                    <th className='p-3 text-left'>{t('User')}</th>
                    <th className='p-3 text-right'>{t('Input tokens')}</th>
                    <th className='p-3 text-right'>{t('Output tokens')}</th>
                    <th className='p-3 text-right'>{t('RMB total')}</th>
                    <th className='p-3 text-center'>{t('Warnings')}</th>
                    <th className='p-3 text-center'>{t('Versions')}</th>
                    <th className='p-3 text-right'>{t('Actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {monthly.map((item) => (
                    <tr key={item.user_id} className='border-t'>
                      <td className='p-3'>
                        <div className='font-medium'>
                          {item.username} · #{item.user_id}
                        </div>
                        <div className='text-muted-foreground text-xs'>
                          {item.email || '-'}
                        </div>
                      </td>
                      <td className='p-3 text-right'>
                        {item.input_tokens.toLocaleString()}
                      </td>
                      <td className='p-3 text-right'>
                        {item.output_tokens.toLocaleString()}
                      </td>
                      <td className='p-3 text-right font-medium'>
                        {formatCents(item.total_cents)}
                      </td>
                      <td className='p-3 text-center'>
                        {item.warnings.unpriced_redemptions +
                          item.warnings.unknown_currency_topups || '-'}
                      </td>
                      <td className='p-3 text-center'>
                        {item.version_count}
                        {item.has_system_final
                          ? ` · ${t('Monthly final')}`
                          : ''}
                      </td>
                      <td className='p-3 text-right'>
                        <Button
                          size='sm'
                          onClick={() => generateForUser(item.user_id)}
                          className='gap-1'
                        >
                          <FileText className='h-4 w-4' />
                          {t('Generate and preview')}
                        </Button>
                      </td>
                    </tr>
                  ))}
                  {!monthly.length && (
                    <tr>
                      <td
                        colSpan={7}
                        className='text-muted-foreground p-10 text-center'
                      >
                        {t('No users found')}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
          {!loading && tab === 'history' && (
            <div className='space-y-2'>
              {history.map((item) => {
                let sourceLabel = t('Admin generated')
                if (item.source === 'system_monthly') {
                  sourceLabel = t('System monthly final')
                } else if (item.source === 'user_export') {
                  sourceLabel = t('User export')
                }
                return (
                  <div
                    key={item.id}
                    className='flex flex-col gap-3 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between'
                  >
                    <div>
                      <div className='font-medium'>{item.statement_no}</div>
                      <div className='text-muted-foreground text-xs'>
                        {item.username} · #{item.user_id} ·{' '}
                        {formatTime(item.generated_at)} · {sourceLabel}
                      </div>
                    </div>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => setStatement(item)}
                      className='gap-1'
                    >
                      <Eye className='h-4 w-4' />
                      {t('Preview')}
                    </Button>
                  </div>
                )
              })}
              {!history.length && (
                <div className='text-muted-foreground p-10 text-center'>
                  {t('No generated statements')}
                </div>
              )}
            </div>
          )}
          {!loading && total > adminPageSize && (
            <div className='flex items-center justify-end gap-2'>
              <Button
                variant='outline'
                size='sm'
                disabled={page <= 1}
                onClick={() => setPage((value) => Math.max(1, value - 1))}
              >
                {t('Previous')}
              </Button>
              <span className='text-muted-foreground text-xs'>
                {page} / {Math.ceil(total / adminPageSize)}
              </span>
              <Button
                variant='outline'
                size='sm'
                disabled={page >= Math.ceil(total / adminPageSize)}
                onClick={() => setPage((value) => value + 1)}
              >
                {t('Next')}
              </Button>
            </div>
          )}
        </div>
      )}
      {!statement && !isAdmin && (
        <div className='space-y-5'>
          <div className='rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900'>
            {t(
              'Only the current Shanghai calendar month through the generation time can be exported. Each generation creates a separate immutable version.'
            )}
          </div>
          <div className='grid gap-4 sm:grid-cols-2'>
            <div className='space-y-2'>
              <Label htmlFor='statement-title'>
                {t('Reconciliation title (optional)')}
              </Label>
              <Input
                id='statement-title'
                maxLength={120}
                value={billingTitle}
                onChange={(event) => setBillingTitle(event.target.value)}
                placeholder={t('For example: company or department name')}
              />
              <p className='text-muted-foreground text-xs'>
                {t(
                  'This is user-supplied reconciliation information, not an invoice title.'
                )}
              </p>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='statement-address'>
                {t('Contact address (optional)')}
              </Label>
              <Input
                id='statement-address'
                maxLength={300}
                value={billingAddress}
                onChange={(event) => setBillingAddress(event.target.value)}
                placeholder={t('For example: Guangdong Province, Shenzhen...')}
              />
            </div>
          </div>
          <div className='flex justify-end'>
            <Button onClick={generateSelf} disabled={loading} className='gap-2'>
              {loading ? (
                <Loader2 className='h-4 w-4 animate-spin' />
              ) : (
                <FileText className='h-4 w-4' />
              )}
              {t('Generate and preview current month')}
            </Button>
          </div>
        </div>
      )}
    </Dialog>
  )
}
