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
import { Plus, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import { safeJsonParseWithValidation } from '../utils/json-parser'

type HupijiaoPackage = {
  id: string
  title: string
  original_amount: number
  quota: number
  discount_rate: number
  enabled: boolean
}

type HupijiaoPackagesVisualEditorProps = {
  value: string
  onChange: (value: string) => void
}

const emptyPackage = (): HupijiaoPackage => ({
  id: `package-${Date.now()}`,
  title: '',
  original_amount: 10,
  quota: 1000,
  discount_rate: 1,
  enabled: true,
})

export function HupijiaoPackagesVisualEditor({
  value,
  onChange,
}: HupijiaoPackagesVisualEditorProps) {
  const { t } = useTranslation()
  const [newPackage, setNewPackage] = useState<HupijiaoPackage>(emptyPackage)
  const packages = useMemo(
    () =>
      safeJsonParseWithValidation<unknown[]>(value, {
        fallback: [],
        validator: (data): data is unknown[] => Array.isArray(data),
        validatorMessage: t('Package configuration must be a JSON array'),
        context: 'Hupijiao packages',
      }).filter((item): item is HupijiaoPackage => {
        if (!item || typeof item !== 'object') return false
        const p = item as Record<string, unknown>
        return typeof p.id === 'string' && typeof p.title === 'string'
      }),
    [value, t]
  )

  const update = (next: HupijiaoPackage[]) =>
    onChange(JSON.stringify(next, null, 2))

  const addPackage = () => {
    if (
      !newPackage.title.trim() ||
      newPackage.original_amount <= 0 ||
      newPackage.quota <= 0
    )
      return
    update([...packages, { ...newPackage, title: newPackage.title.trim() }])
    setNewPackage(emptyPackage())
  }

  return (
    <div className='space-y-4'>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Configure RMB recharge packages for users. The discounted payment amount is calculated automatically.'
        )}
      </p>
      {packages.length > 0 && (
        <div className='space-y-3'>
          {packages.map((pkg, index) => (
            <div key={`${pkg.id}-${index}`} className='rounded-md border p-4'>
              <div className='grid gap-3 md:grid-cols-6'>
                <div className='md:col-span-2'>
                  <Label>{t('Package name')}</Label>
                  <Input
                    value={pkg.title}
                    onChange={(e) => {
                      const next = [...packages]
                      next[index] = { ...pkg, title: e.target.value }
                      update(next)
                    }}
                  />
                </div>
                <div>
                  <Label>{t('Original price (CNY)')}</Label>
                  <Input
                    type='number'
                    min='0.01'
                    step='0.01'
                    value={pkg.original_amount}
                    onChange={(e) => {
                      const next = [...packages]
                      next[index] = {
                        ...pkg,
                        original_amount: Number(e.target.value),
                      }
                      update(next)
                    }}
                  />
                </div>
                <div>
                  <Label>{t('Credited quota')}</Label>
                  <Input
                    type='number'
                    min='1'
                    step='1'
                    value={pkg.quota}
                    onChange={(e) => {
                      const next = [...packages]
                      next[index] = { ...pkg, quota: Number(e.target.value) }
                      update(next)
                    }}
                  />
                </div>
                <div>
                  <Label>{t('Discount (enter 0.8 for 20% off)')}</Label>
                  <Input
                    type='number'
                    min='0.01'
                    max='1'
                    step='0.01'
                    value={pkg.discount_rate}
                    onChange={(e) => {
                      const next = [...packages]
                      next[index] = {
                        ...pkg,
                        discount_rate: Number(e.target.value),
                      }
                      update(next)
                    }}
                  />
                </div>
                <div className='flex items-end justify-between gap-2'>
                  <div className='flex items-center gap-2 pb-2'>
                    <Switch
                      checked={pkg.enabled}
                      onCheckedChange={(enabled) => {
                        const next = [...packages]
                        next[index] = { ...pkg, enabled }
                        update(next)
                      }}
                    />
                    <Label>{t('Enabled')}</Label>
                  </div>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    title={t('Delete package')}
                    aria-label={t('Delete package')}
                    onClick={() =>
                      update(packages.filter((_, i) => i !== index))
                    }
                  >
                    <Trash2 className='h-4 w-4' />
                  </Button>
                </div>
              </div>
              <p className='text-muted-foreground mt-2 text-xs'>
                ID: {pkg.id} · {t('User pays')} ¥
                {(
                  Number(pkg.original_amount) * Number(pkg.discount_rate)
                ).toFixed(2)}
              </p>
            </div>
          ))}
        </div>
      )}
      <div className='rounded-md border border-dashed p-4'>
        <div className='grid gap-3 md:grid-cols-5'>
          <div className='md:col-span-2'>
            <Label>{t('New package name')}</Label>
            <Input
              placeholder={t('e.g. 10 CNY package')}
              value={newPackage.title}
              onChange={(e) =>
                setNewPackage({ ...newPackage, title: e.target.value })
              }
            />
          </div>
          <div>
            <Label>{t('Original price (CNY)')}</Label>
            <Input
              type='number'
              min='0.01'
              step='0.01'
              value={newPackage.original_amount}
              onChange={(e) =>
                setNewPackage({
                  ...newPackage,
                  original_amount: Number(e.target.value),
                })
              }
            />
          </div>
          <div>
            <Label>{t('Credited quota')}</Label>
            <Input
              type='number'
              min='1'
              step='1'
              value={newPackage.quota}
              onChange={(e) =>
                setNewPackage({ ...newPackage, quota: Number(e.target.value) })
              }
            />
          </div>
          <div>
            <Label>{t('Discount (enter 0.8 for 20% off)')}</Label>
            <Input
              type='number'
              min='0.01'
              max='1'
              step='0.01'
              value={newPackage.discount_rate}
              onChange={(e) =>
                setNewPackage({
                  ...newPackage,
                  discount_rate: Number(e.target.value),
                })
              }
            />
          </div>
        </div>
        <Button
          type='button'
          className='mt-3'
          onClick={addPackage}
          disabled={!newPackage.title.trim()}
        >
          <Plus className='mr-2 h-4 w-4' />
          {t('Add package')}
        </Button>
      </div>
    </div>
  )
}
