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
import type { Table } from '@tanstack/react-table'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { CopyButton } from '@/components/copy-button'
import { DataTableBulkActions as BulkActionsToolbar } from '@/components/data-table'
import { Button } from '@/components/ui/button'

import { assignRedemptionCategory, getRedemptionCategories } from '../api'
import type { Redemption, RedemptionCategory } from '../types'
import { useRedemptions } from './redemptions-provider'

type DataTableBulkActionsProps<TData> = {
  table: Table<TData>
}

export function DataTableBulkActions<TData>({
  table,
}: DataTableBulkActionsProps<TData>) {
  const { t } = useTranslation()
  const { triggerRefresh } = useRedemptions()
  const selectedRows = table.getSelectedRowModel().rows
  const [categories, setCategories] = useState<RedemptionCategory[]>([])
  const [categoryId, setCategoryId] = useState(0)
  const [assigning, setAssigning] = useState(false)

  useEffect(() => {
    void getRedemptionCategories(false).then((result) => {
      if (result.success) setCategories(result.data || [])
    })
  }, [])

  const contentToCopy = useMemo(() => {
    const selectedCodes = selectedRows.map((row) => {
      const redemption = row.original as Redemption
      return `${redemption.name}\t${redemption.key}`
    })
    return selectedCodes.join('\n')
  }, [selectedRows])

  const pendingIds = selectedRows
    .map((row) => row.original as Redemption)
    .filter((redemption) => !redemption.category_priced_at)
    .map((redemption) => redemption.id)

  const assignCategory = async () => {
    if (!categoryId || !pendingIds.length) return
    setAssigning(true)
    try {
      const result = await assignRedemptionCategory(pendingIds, categoryId)
      if (!result.success) throw new Error(result.message)
      toast.success(
        t('Assigned categories to {{count}} legacy redemption codes', {
          count: result.data?.assigned || pendingIds.length,
        })
      )
      table.resetRowSelection()
      triggerRefresh()
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('Failed to assign redemption category')
      )
    } finally {
      setAssigning(false)
    }
  }

  return (
    <BulkActionsToolbar table={table} entityName={t('redemption code')}>
      <CopyButton
        value={contentToCopy}
        variant='outline'
        size='icon'
        className='size-8'
        tooltip={t('Copy selected codes')}
        successTooltip={t('Codes copied!')}
        aria-label={t('Copy selected codes')}
      />
      <select
        value={categoryId || ''}
        onChange={(event) => setCategoryId(Number(event.target.value))}
        className='border-input bg-background h-8 max-w-48 rounded-md border px-2 text-xs'
        aria-label={t('Category for legacy codes')}
      >
        <option value=''>{t('Select category for pending pricing')}</option>
        {categories.map((category) => (
          <option key={category.id} value={category.id}>
            {category.name} · ¥{(category.price_cents / 100).toFixed(2)}
          </option>
        ))}
      </select>
      <Button
        size='sm'
        variant='outline'
        className='h-8'
        disabled={!categoryId || !pendingIds.length || assigning}
        onClick={assignCategory}
      >
        {t('Price selected')} ({pendingIds.length})
      </Button>
    </BulkActionsToolbar>
  )
}
