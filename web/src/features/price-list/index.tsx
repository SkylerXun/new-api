import { Search } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'

import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'

import { getPriceList } from './api'
import { PriceListTable } from './components/price-list-table'
import { filterPriceListModels } from './lib'

export function PriceList() {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const [vendorID, setVendorID] = useState('all')
  const [groupFilter, setGroupFilter] = useState('all')
  const [ratioFilter, setRatioFilter] = useState('all')
  const priceListQuery = useQuery({
    queryKey: ['price-list'],
    queryFn: getPriceList,
    staleTime: 5 * 60 * 1000,
  })

  const data = priceListQuery.data
  const groups = useMemo(
    () =>
      Object.entries(data?.group_ratio || {})
        .filter(([name]) => name !== '' && name !== 'auto')
        .sort(([a], [b]) => a.localeCompare(b)),
    [data?.group_ratio]
  )
  const ratios = useMemo(
    () => [...new Set(groups.map(([, ratio]) => ratio))].sort((a, b) => a - b),
    [groups]
  )
  const visibleGroups = useMemo(
    () =>
      groups.filter(([group, ratio]) => {
        if (groupFilter !== 'all' && group !== groupFilter) return false
        return ratioFilter === 'all' || ratio === Number(ratioFilter)
      }),
    [groupFilter, groups, ratioFilter]
  )

  if (priceListQuery.isLoading) {
    return <PriceListLoading />
  }

  if (!data || !data.success) {
    return (
      <PublicLayout showMainContainer={false}>
        <main className='mx-auto w-full max-w-[1600px] px-4 pt-24 pb-10 sm:px-6'>
          <p className='text-muted-foreground text-sm'>
            {t('Unable to load price list')}
          </p>
        </main>
      </PublicLayout>
    )
  }

  const groupSections = visibleGroups.flatMap(([group, ratio]) => {
    const models = filterPriceListModels(data.data, group, vendorID, query)
    if (models.length === 0) return []
    return [{ group, ratio, models, info: data.group_info[group] }]
  })

  return (
    <PublicLayout showMainContainer={false}>
      <PageTransition className='mx-auto w-full max-w-[1600px] px-3 pt-20 pb-10 sm:px-6 sm:pt-24'>
        <header className='border-b pb-5'>
          <h1 className='text-2xl font-semibold'>{t('Price List')}</h1>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Compare actual and official model pricing by group.')}
          </p>
          <div className='mt-5 grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto]'>
            <label className='relative block max-w-md'>
              <Search
                aria-hidden='true'
                className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2'
              />
              <Input
                aria-label={t('Search model name')}
                className='pl-9'
                onChange={(event) => setQuery(event.target.value)}
                placeholder={t('Search model name')}
                value={query}
              />
            </label>
            <div className='flex flex-wrap gap-2'>
              <Button
                onClick={() => {
                  setQuery('')
                  setVendorID('all')
                  setGroupFilter('all')
                  setRatioFilter('all')
                }}
                size='sm'
                type='button'
                variant='outline'
              >
                {t('Reset filters')}
              </Button>
            </div>
          </div>
        </header>

        <FilterRow
          label={t('Vendor')}
          options={[
            { value: 'all', label: t('All') },
            ...data.vendors.map((vendor) => ({ value: String(vendor.id), label: vendor.name })),
          ]}
          value={vendorID}
          onChange={setVendorID}
        />
        <FilterRow
          label={t('Group')}
          options={[
            { value: 'all', label: t('All') },
            ...groups.map(([name]) => ({
              value: name,
              label: data.group_info[name]?.ratio_label || name,
            })),
          ]}
          value={groupFilter}
          onChange={setGroupFilter}
        />
        <FilterRow
          label={t('Multiplier')}
          options={[
            { value: 'all', label: t('All') },
            ...ratios.map((ratio) => ({ value: String(ratio), label: `${ratio}x` })),
          ]}
          value={ratioFilter}
          onChange={setRatioFilter}
        />

        <main className='mt-6 space-y-5'>
          {groupSections.map(({ group, ratio, models, info }) => (
              <section className='border' key={group}>
                <div className='flex min-h-14 flex-wrap items-center gap-x-3 gap-y-1 border-b px-4 py-3'>
                  <h2 className='text-sm font-semibold'>
                    {info?.ratio_label || group}
                  </h2>
                  <span className='bg-emerald-100 px-2 py-0.5 font-mono text-xs font-medium text-emerald-800 dark:bg-emerald-950 dark:text-emerald-200'>
                    {ratio}x
                  </span>
                  {info?.desc && info.desc !== info.ratio_label && (
                    <span className='text-muted-foreground text-xs'>{info.desc}</span>
                  )}
                </div>
                <PriceListTable groupRatio={ratio} models={models} />
              </section>
            ))}
          {groupSections.length === 0 && (
            <p className='text-muted-foreground py-12 text-center text-sm'>
              {t('No groups match the current filters.')}
            </p>
          )}
        </main>
      </PageTransition>
    </PublicLayout>
  )
}

function FilterRow(props: {
  label: string
  options: Array<{ value: string; label: string }>
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className='flex gap-3 border-b py-3'>
      <span className='text-muted-foreground w-20 shrink-0 pt-1 text-xs font-medium'>
        {props.label}
      </span>
      <div className='flex min-w-0 flex-wrap gap-2'>
        {props.options.map((option) => (
          <Button
            key={option.value}
            onClick={() => props.onChange(option.value)}
            size='sm'
            type='button'
            variant={option.value === props.value ? 'default' : 'outline'}
          >
            {option.label}
          </Button>
        ))}
      </div>
    </div>
  )
}

function PriceListLoading() {
  return (
    <PublicLayout showMainContainer={false}>
      <div className='mx-auto w-full max-w-[1600px] space-y-5 px-3 pt-20 sm:px-6 sm:pt-24'>
        <Skeleton className='h-32 w-full' />
        <Skeleton className='h-96 w-full' />
      </div>
    </PublicLayout>
  )
}
