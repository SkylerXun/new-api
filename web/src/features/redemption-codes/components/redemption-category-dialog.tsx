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
import { Loader2, Pencil, Plus } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import {
  createRedemptionCategory,
  getRedemptionCategories,
  updateRedemptionCategory,
  updateRedemptionCategoryStatus,
} from '../api'
import type { RedemptionCategory } from '../types'

export function RedemptionCategoryDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const [categories, setCategories] = useState<RedemptionCategory[]>([])
  const [editingId, setEditingId] = useState<number | null>(null)
  const [name, setName] = useState('')
  const [price, setPrice] = useState('0.00')
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const result = await getRedemptionCategories(true)
      if (!result.success) throw new Error(result.message)
      setCategories(result.data || [])
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('Failed to load redemption categories')
      )
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (open) void load()
  }, [load, open])

  const resetForm = () => {
    setEditingId(null)
    setName('')
    setPrice('0.00')
  }

  const save = async () => {
    const value = Number(price)
    if (!name.trim() || !Number.isFinite(value) || value < 0) {
      toast.error(t('Enter a valid category name and non-negative RMB price'))
      return
    }
    setLoading(true)
    try {
      const input = {
        name: name.trim(),
        price_cents: Math.round(value * 100),
      }
      const result = editingId
        ? await updateRedemptionCategory(editingId, input)
        : await createRedemptionCategory(input)
      if (!result.success) throw new Error(result.message)
      toast.success(t(editingId ? 'Category updated' : 'Category created'))
      resetForm()
      await load()
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('Failed to save redemption category')
      )
    } finally {
      setLoading(false)
    }
  }

  const toggle = async (category: RedemptionCategory) => {
    try {
      const result = await updateRedemptionCategoryStatus(
        category.id,
        !category.enabled
      )
      if (!result.success) throw new Error(result.message)
      await load()
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('Failed to update category status')
      )
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Redemption Category Management')}
      description={t(
        'Category prices are stored in RMB cents and snapshotted onto each redemption code.'
      )}
      contentClassName='sm:max-w-3xl'
      contentHeight='auto'
      bodyClassName='space-y-5'
    >
      <div className='grid gap-3 rounded-lg border p-4 sm:grid-cols-[1fr_180px_auto] sm:items-end'>
        <div className='space-y-2'>
          <Label htmlFor='category-name'>{t('Category name')}</Label>
          <Input
            id='category-name'
            maxLength={80}
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder={t('For example: Gift code or CNY 100 package')}
          />
        </div>
        <div className='space-y-2'>
          <Label htmlFor='category-price'>{t('RMB price')}</Label>
          <Input
            id='category-price'
            type='number'
            min='0'
            step='0.01'
            value={price}
            onChange={(event) => setPrice(event.target.value)}
          />
        </div>
        <div className='flex gap-2'>
          {editingId && (
            <Button variant='outline' onClick={resetForm}>
              {t('Cancel')}
            </Button>
          )}
          <Button onClick={save} disabled={loading} className='gap-1'>
            {loading ? (
              <Loader2 className='h-4 w-4 animate-spin' />
            ) : (
              <Plus className='h-4 w-4' />
            )}
            {t(editingId ? 'Save' : 'Create')}
          </Button>
        </div>
      </div>

      <div className='max-h-[46vh] space-y-2 overflow-y-auto'>
        {categories.map((category) => (
          <div
            key={category.id}
            className='flex flex-col gap-3 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between'
          >
            <div>
              <div className='font-medium'>{category.name}</div>
              <div className='text-muted-foreground text-sm'>
                ¥{(category.price_cents / 100).toFixed(2)} · #{category.id} ·{' '}
                {category.enabled ? t('Enabled') : t('Disabled')}
              </div>
            </div>
            <div className='flex gap-2'>
              <Button
                variant='outline'
                size='sm'
                onClick={() => {
                  setEditingId(category.id)
                  setName(category.name)
                  setPrice((category.price_cents / 100).toFixed(2))
                }}
                className='gap-1'
              >
                <Pencil className='h-4 w-4' />
                {t('Edit')}
              </Button>
              <Button
                variant='outline'
                size='sm'
                onClick={() => toggle(category)}
              >
                {category.enabled ? t('Disable') : t('Enable')}
              </Button>
            </div>
          </div>
        ))}
        {!loading && !categories.length && (
          <div className='text-muted-foreground p-8 text-center'>
            {t(
              'No redemption categories. Create a free category with ¥0.00 for gift codes.'
            )}
          </div>
        )}
      </div>
    </Dialog>
  )
}
