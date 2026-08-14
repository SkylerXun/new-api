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
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { AlertCircle, BookOpen, Loader2 } from 'lucide-react'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import { getPublishedGuides } from './api'
import type { Guide } from './types'

const EMPTY_GUIDES: Guide[] = []

type GuidesProps = {
  selectedSlug?: string
}

export function Guides({ selectedSlug }: GuidesProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const query = useQuery({
    queryKey: ['guides'],
    queryFn: getPublishedGuides,
  })
  const guides = query.data?.data ?? EMPTY_GUIDES
  const firstGuide = guides.at(0)
  const selectedGuide = guides.find((guide) => guide.slug === selectedSlug)

  useEffect(() => {
    if (!selectedSlug && firstGuide) {
      void navigate({
        to: '/guides/$slug',
        params: { slug: firstGuide.slug },
        replace: true,
      })
    }
  }, [firstGuide, navigate, selectedSlug])

  const renderContent = () => {
    if (query.isLoading) {
      return (
        <StateMessage
          icon={<Loader2 className='h-7 w-7 animate-spin' />}
          title={t('Loading guides...')}
        />
      )
    }

    if (query.isError) {
      return (
        <StateMessage
          icon={<AlertCircle className='h-7 w-7' />}
          title={t('Failed to load guides')}
          description={t('Please check your connection and try again.')}
          action={
            <Button size='sm' variant='outline' onClick={() => query.refetch()}>
              {t('Retry')}
            </Button>
          }
        />
      )
    }

    if (!firstGuide) {
      return (
        <StateMessage
          icon={<BookOpen className='h-7 w-7' />}
          title={t('No published guides')}
          description={t('Published guides will appear here.')}
        />
      )
    }

    if (selectedSlug && !selectedGuide) {
      return (
        <StateMessage
          icon={<AlertCircle className='h-7 w-7' />}
          title={t('Guide not found')}
          description={t('This guide may have been removed or hidden.')}
          action={
            <Button
              size='sm'
              onClick={() =>
                void navigate({
                  to: '/guides/$slug',
                  params: { slug: firstGuide.slug },
                })
              }
            >
              {t('Open first guide')}
            </Button>
          }
        />
      )
    }

    return (
      <div className='space-y-4'>
        <div className='md:hidden'>
          <Select
            value={selectedGuide?.slug ?? firstGuide.slug}
            onValueChange={(slug) => {
              if (slug) {
                void navigate({ to: '/guides/$slug', params: { slug } })
              }
            }}
          >
            <SelectTrigger className='w-full'>
              <SelectValue placeholder={t('Select a guide')} />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {guides.map((guide) => (
                  <SelectItem key={guide.id} value={guide.slug}>
                    {guide.title}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className='grid min-h-[32rem] min-w-0 gap-8 md:grid-cols-[16rem_minmax(0,1fr)]'>
          <nav
            className='hidden border-r pr-4 md:block'
            aria-label={t('Guide list')}
          >
            <div className='sticky top-4 space-y-1'>
              {guides.map((guide) => {
                const active = guide.slug === selectedGuide?.slug
                return (
                  <Link
                    key={guide.id}
                    to='/guides/$slug'
                    params={{ slug: guide.slug }}
                    aria-current={active ? 'page' : undefined}
                    className={cn(
                      'block rounded-md px-3 py-2.5 text-sm transition-colors',
                      active
                        ? 'bg-accent text-accent-foreground font-medium'
                        : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
                    )}
                  >
                    <span className='block'>{guide.title}</span>
                    {guide.summary && (
                      <span className='mt-1 line-clamp-2 block text-xs font-normal opacity-75'>
                        {guide.summary}
                      </span>
                    )}
                  </Link>
                )
              })}
            </div>
          </nav>

          {selectedGuide && (
            <article className='min-w-0 pb-12'>
              <header className='mb-7 border-b pb-5'>
                <h1 className='text-2xl font-semibold tracking-normal sm:text-3xl'>
                  {selectedGuide.title}
                </h1>
                {selectedGuide.summary && (
                  <p className='text-muted-foreground mt-2 leading-7'>
                    {selectedGuide.summary}
                  </p>
                )}
                <p className='text-muted-foreground mt-3 text-xs'>
                  {t('Updated {{date}}', {
                    date: dayjs(selectedGuide.updatedAt).format(
                      'YYYY-MM-DD HH:mm'
                    ),
                  })}
                </p>
              </header>
              <RichContent
                className='max-w-none'
                content={selectedGuide.content}
                mode={selectedGuide.format}
                htmlVariant='isolated'
              />
            </article>
          )}
        </div>
      </div>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Guides')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>{renderContent()}</SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function StateMessage(props: {
  icon: React.ReactNode
  title: string
  description?: string
  action?: React.ReactNode
}) {
  return (
    <div className='flex min-h-80 flex-col items-center justify-center gap-3 text-center'>
      <div className='text-muted-foreground'>{props.icon}</div>
      <div>
        <h2 className='font-medium'>{props.title}</h2>
        {props.description && (
          <p className='text-muted-foreground mt-1 text-sm'>
            {props.description}
          </p>
        )}
      </div>
      {props.action}
    </div>
  )
}
