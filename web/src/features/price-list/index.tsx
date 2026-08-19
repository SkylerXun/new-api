import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Search } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { PageTransition } from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useIsAdmin } from '@/hooks/use-admin'

import { getPriceList } from './api'
import { PriceListTable } from './components/price-list-table'
import { filterPriceListModels } from './lib'

export function PriceList() {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
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

  if (priceListQuery.isError || !data || !data.success) {
    return (
      <main className='mx-auto w-full max-w-[1600px] space-y-3 px-4 py-6 sm:px-6'>
        <div>
          <p className='text-sm font-medium'>{t('无法加载模型价格清单')}</p>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('请确认侧边栏中的模型价格清单已启用，然后重试。')}
          </p>
        </div>
        <Button
          size='sm'
          variant='outline'
          onClick={() => void priceListQuery.refetch()}
        >
          {t('重试')}
        </Button>
      </main>
    )
  }

  const groupSections = visibleGroups.flatMap(([group, ratio]) => {
    const models = filterPriceListModels(data.data, group, vendorID, query)
    if (models.length === 0) return []
    return [{ group, ratio, models, info: data.group_info[group] }]
  })

  return (
    <PageTransition className='mx-auto w-full max-w-[1600px] px-3 py-6 sm:px-6'>
      <header className='border-b pb-5'>
        <div className='flex flex-wrap items-start justify-between gap-4'>
          <div>
            <h1 className='text-2xl font-semibold'>{t('模型价格清单')}</h1>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t('按分组对比实际价格与官方模型价格。')}
            </p>
          </div>
          {isAdmin && (
            <Button
              size='sm'
              variant='outline'
              render={
                <Link to='/models/$section' params={{ section: 'metadata' }} />
              }
            >
              {t('编辑模型官方价格')}
            </Button>
          )}
        </div>
        <div className='mt-5 grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto]'>
          <label className='relative block max-w-md'>
            <Search
              aria-hidden='true'
              className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2'
            />
            <Input
              aria-label={t('搜索模型名称')}
              className='pl-9'
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t('搜索模型名称')}
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
              {t('重置筛选')}
            </Button>
          </div>
        </div>
      </header>

      <FilterRow
        label={t('供应商')}
        options={[
          { value: 'all', label: t('全部') },
          ...data.vendors.map((vendor) => ({
            value: String(vendor.id),
            label: vendor.name,
          })),
        ]}
        value={vendorID}
        onChange={setVendorID}
      />
      <FilterRow
        label={t('分组')}
        options={[
          { value: 'all', label: t('全部') },
          ...groups.map(([name]) => ({
            value: name,
            label: data.group_info[name]?.ratio_label || name,
          })),
        ]}
        value={groupFilter}
        onChange={setGroupFilter}
      />
      <FilterRow
        label={t('倍率')}
        options={[
          { value: 'all', label: t('全部') },
          ...ratios.map((ratio) => ({
            value: String(ratio),
            label: `${ratio}x`,
          })),
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
                <span className='text-muted-foreground text-xs'>
                  {info.desc}
                </span>
              )}
            </div>
            <PriceListTable groupRatio={ratio} models={models} />
          </section>
        ))}
        {groupSections.length === 0 && (
          <p className='text-muted-foreground py-12 text-center text-sm'>
            {t('没有符合当前筛选条件的分组。')}
          </p>
        )}
      </main>
    </PageTransition>
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
    <div className='mx-auto w-full max-w-[1600px] space-y-5 px-3 py-6 sm:px-6'>
      <Skeleton className='h-32 w-full' />
      <Skeleton className='h-96 w-full' />
    </div>
  )
}
