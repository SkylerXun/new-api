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
import { Image, Link2, Plus, Save, Trash2, Video } from 'lucide-react'
import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { Dialog } from '@/components/dialog'
import { RichContent } from '@/components/rich-content'
import { StatusBadge } from '@/components/status-badge'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
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
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import dayjs from '@/lib/dayjs'

import { uploadGuideMedia } from '../api'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

type Guide = {
  id: number
  slug: string
  title: string
  summary: string
  content: string
  format: 'markdown' | 'html'
  updatedAt: string
  order: number
  published: boolean
}

type GuidesSectionProps = {
  data: string
}

const guideSchema = z.object({
  title: z.string().trim().min(1, 'Title is required').max(120),
  summary: z.string().trim().max(300),
  slug: z
    .string()
    .trim()
    .min(1, 'Slug is required')
    .max(100)
    .regex(
      /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
      'Use lowercase letters, numbers, and single hyphens'
    ),
  format: z.enum(['markdown', 'html']),
  content: z.string().trim().min(1, 'Content is required').max(200_000),
  order: z.number().int(),
  published: z.boolean(),
})

type GuideFormValues = z.infer<typeof guideSchema>

const GUIDE_FORM_ID = 'guide-form'

function parseGuides(data: string): Guide[] {
  try {
    const parsed: unknown = JSON.parse(data || '[]')
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (item): item is Guide =>
        typeof item === 'object' &&
        item !== null &&
        typeof (item as Guide).id === 'number'
    )
  } catch {
    return []
  }
}

export function GuidesSection({ data }: GuidesSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [guides, setGuides] = useState<Guide[]>([])
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [editingGuide, setEditingGuide] = useState<Guide | null>(null)
  const [showEditor, setShowEditor] = useState(false)
  const [deleteMode, setDeleteMode] = useState<'single' | 'batch' | null>(null)
  const [hasChanges, setHasChanges] = useState(false)
  const contentTextareaRef = useRef<HTMLTextAreaElement | null>(null)
  const imageInputRef = useRef<HTMLInputElement | null>(null)
  const videoInputRef = useRef<HTMLInputElement | null>(null)
  const selectionRef = useRef({ start: 0, end: 0 })
  const [uploadingMedia, setUploadingMedia] = useState<
    'image' | 'video' | null
  >(null)

  const form = useForm<GuideFormValues>({
    resolver: zodResolver(guideSchema),
    defaultValues: {
      title: '',
      summary: '',
      slug: '',
      format: 'markdown',
      content: '',
      order: 10,
      published: true,
    },
  })

  useEffect(() => {
    setGuides(parseGuides(data))
    setSelectedIds([])
    setHasChanges(false)
  }, [data])

  const sortedGuides = useMemo(
    () =>
      [...guides].sort(
        (a, b) => a.order - b.order || a.title.localeCompare(b.title)
      ),
    [guides]
  )

  const openAdd = () => {
    setEditingGuide(null)
    form.reset({
      title: '',
      summary: '',
      slug: '',
      format: 'markdown',
      content: '',
      order: Math.max(...guides.map((guide) => guide.order), 0) + 10,
      published: true,
    })
    setShowEditor(true)
  }

  const openEdit = (guide: Guide) => {
    setEditingGuide(guide)
    form.reset({
      title: guide.title,
      summary: guide.summary,
      slug: guide.slug,
      format: guide.format,
      content: guide.content,
      order: guide.order,
      published: guide.published,
    })
    setShowEditor(true)
  }

  const submitGuide = (values: GuideFormValues) => {
    const duplicate = guides.some(
      (guide) => guide.slug === values.slug && guide.id !== editingGuide?.id
    )
    if (duplicate) {
      form.setError('slug', { message: t('This slug is already in use') })
      return
    }

    const updatedAt = new Date().toISOString()
    if (editingGuide) {
      setGuides((current) =>
        current.map((guide) =>
          guide.id === editingGuide.id
            ? { ...guide, ...values, updatedAt }
            : guide
        )
      )
      toast.success(t('Guide updated. Save settings to apply.'))
    } else {
      const id = Math.max(...guides.map((guide) => guide.id), 0) + 1
      setGuides((current) => [...current, { id, ...values, updatedAt }])
      toast.success(t('Guide added. Save settings to apply.'))
    }
    setHasChanges(true)
    setShowEditor(false)
  }

  const updateGuide = (id: number, changes: Partial<Guide>) => {
    setGuides((current) =>
      current.map((guide) =>
        guide.id === id
          ? { ...guide, ...changes, updatedAt: new Date().toISOString() }
          : guide
      )
    )
    setHasChanges(true)
  }

  const confirmDelete = () => {
    if (deleteMode === 'single' && editingGuide) {
      setGuides((current) =>
        current.filter((guide) => guide.id !== editingGuide.id)
      )
    } else if (deleteMode === 'batch') {
      setGuides((current) =>
        current.filter((guide) => !selectedIds.includes(guide.id))
      )
      setSelectedIds([])
    }
    setHasChanges(true)
    setDeleteMode(null)
    setEditingGuide(null)
    toast.success(t('Guide deleted. Save settings to apply.'))
  }

  const saveAll = async () => {
    try {
      const result = await updateOption.mutateAsync({
        key: 'console_setting.guides',
        value: JSON.stringify(guides),
      })
      if (!result.success) {
        return
      }
      setHasChanges(false)
      toast.success(t('Guides saved successfully'))
    } catch {
      toast.error(t('Failed to save guides'))
    }
  }

  const insertMediaTag = (kind: 'image' | 'video') => {
    const url = window.prompt(
      t(kind === 'image' ? '请输入图片地址' : '请输入视频地址（MP4 直链）')
    )
    if (!url) return
    const trimmedUrl = url.trim()
    if (!/^https?:\/\//i.test(trimmedUrl)) {
      toast.error(t('媒体地址必须以 http:// 或 https:// 开头'))
      return
    }

    const textarea = contentTextareaRef.current
    const current = form.getValues('content')
    const snippet =
      kind === 'image'
        ? `<p><img src="${trimmedUrl}" alt="${t('教程图片')}" /></p>`
        : `<p><video controls preload="metadata" src="${trimmedUrl}"></video></p>`
    const start = textarea?.selectionStart ?? current.length
    const end = textarea?.selectionEnd ?? current.length
    const next = `${current.slice(0, start)}${snippet}${current.slice(end)}`

    form.setValue('format', 'html', { shouldDirty: true })
    form.setValue('content', next, { shouldDirty: true, shouldValidate: true })
    requestAnimationFrame(() => {
      if (!textarea) return
      const cursor = start + snippet.length
      textarea.focus()
      textarea.setSelectionRange(cursor, cursor)
    })
  }

  const rememberSelection = () => {
    const textarea = contentTextareaRef.current
    const current = form.getValues('content')
    selectionRef.current = {
      start: textarea?.selectionStart ?? current.length,
      end: textarea?.selectionEnd ?? current.length,
    }
  }

  const handleMediaFile = async (
    event: ChangeEvent<HTMLInputElement>,
    kind: 'image' | 'video'
  ) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return

    setUploadingMedia(kind)
    try {
      const result = await uploadGuideMedia(file, kind)
      if (!result.success || !result.url) {
        toast.error(result.message || t('媒体上传失败'))
        return
      }

      const current = form.getValues('content')
      const { start, end } = selectionRef.current
      const safeStart = Math.min(start, current.length)
      const safeEnd = Math.min(Math.max(end, safeStart), current.length)
      const snippet =
        kind === 'image'
          ? `<p><img src="${result.url}" alt="${t('教程图片')}" /></p>`
          : `<p><video controls preload="metadata" src="${result.url}"></video></p>`
      const next = `${current.slice(0, safeStart)}${snippet}${current.slice(safeEnd)}`
      form.setValue('format', 'html', { shouldDirty: true })
      form.setValue('content', next, {
        shouldDirty: true,
        shouldValidate: true,
      })
      requestAnimationFrame(() => {
        const textarea = contentTextareaRef.current
        if (!textarea) return
        const cursor = safeStart + snippet.length
        textarea.focus()
        textarea.setSelectionRange(cursor, cursor)
      })
    } catch {
      toast.error(t('媒体上传失败'))
    } finally {
      setUploadingMedia(null)
    }
  }

  const previewContent = form.watch('content')
  const previewFormat = form.watch('format')

  return (
    <SettingsSection title={t('Guides')}>
      <div className='space-y-4'>
        <div className='flex flex-wrap items-center gap-2'>
          <Button size='sm' onClick={openAdd}>
            <Plus className='mr-2 h-4 w-4' />
            {t('Add Guide')}
          </Button>
          <Button
            size='sm'
            variant='destructive'
            disabled={selectedIds.length === 0}
            onClick={() => setDeleteMode('batch')}
          >
            <Trash2 className='mr-2 h-4 w-4' />
            {t('Delete selected')} ({selectedIds.length})
          </Button>
          <Button
            size='sm'
            variant='secondary'
            disabled={!hasChanges || updateOption.isPending}
            onClick={saveAll}
          >
            <Save className='mr-2 h-4 w-4' />
            {updateOption.isPending ? t('Saving...') : t('Save Settings')}
          </Button>
        </div>

        <StaticDataTable
          data={sortedGuides}
          getRowKey={(guide) => guide.id}
          emptyContent={t('No guides yet. Add the first guide to get started.')}
          columns={[
            {
              id: 'select',
              className: 'w-12',
              header: (
                <Checkbox
                  checked={
                    guides.length > 0 && selectedIds.length === guides.length
                  }
                  onCheckedChange={(checked) =>
                    setSelectedIds(
                      checked ? guides.map((guide) => guide.id) : []
                    )
                  }
                />
              ),
              cell: (guide) => (
                <Checkbox
                  checked={selectedIds.includes(guide.id)}
                  onCheckedChange={(checked) =>
                    setSelectedIds((current) =>
                      checked
                        ? [...current, guide.id]
                        : current.filter((id) => id !== guide.id)
                    )
                  }
                />
              ),
            },
            {
              id: 'title',
              header: t('Title'),
              cell: (guide) => (
                <div className='min-w-44'>
                  <div className='font-medium'>{guide.title}</div>
                  <div className='text-muted-foreground text-xs'>
                    /guides/{guide.slug}
                  </div>
                </div>
              ),
            },
            {
              id: 'summary',
              header: t('Summary'),
              cellClassName: 'max-w-72 truncate text-muted-foreground',
              cell: (guide) => guide.summary || '-',
            },
            {
              id: 'updated',
              header: t('Updated At'),
              cell: (guide) =>
                dayjs(guide.updatedAt).format('YYYY-MM-DD HH:mm'),
            },
            {
              id: 'order',
              header: t('Order'),
              cell: (guide) => (
                <Input
                  aria-label={t('Order')}
                  className='h-8 w-20'
                  type='number'
                  value={guide.order}
                  onChange={(event) =>
                    updateGuide(guide.id, { order: Number(event.target.value) })
                  }
                />
              ),
            },
            {
              id: 'published',
              header: t('Published'),
              cell: (guide) => (
                <div className='flex items-center gap-2'>
                  <Checkbox
                    aria-label={t('Published')}
                    checked={guide.published}
                    onCheckedChange={(checked) =>
                      updateGuide(guide.id, { published: Boolean(checked) })
                    }
                  />
                  <StatusBadge
                    label={guide.published ? t('Published') : t('Hidden')}
                    variant={guide.published ? 'success' : 'neutral'}
                    copyable={false}
                  />
                </div>
              ),
            },
            {
              id: 'actions',
              header: t('Actions'),
              cell: (guide) => (
                <StaticRowActions
                  editLabel={t('Edit')}
                  deleteLabel={t('Delete')}
                  menuLabel={t('Open menu')}
                  onEdit={() => openEdit(guide)}
                  onDelete={() => {
                    setEditingGuide(guide)
                    setDeleteMode('single')
                  }}
                />
              ),
            },
          ]}
        />
      </div>

      <Dialog
        open={showEditor}
        onOpenChange={setShowEditor}
        title={editingGuide ? t('Edit Guide') : t('Add Guide')}
        description={t(
          'Write the source and review the live preview before saving.'
        )}
        contentClassName='max-w-6xl'
        contentHeight='80vh'
        bodyClassName='overflow-y-auto'
        footer={
          <>
            <Button variant='outline' onClick={() => setShowEditor(false)}>
              {t('Cancel')}
            </Button>
            <Button type='submit' form={GUIDE_FORM_ID}>
              {editingGuide ? t('Update') : t('Add')}
            </Button>
          </>
        }
      >
        <Form {...form}>
          <form
            id={GUIDE_FORM_ID}
            className='grid gap-6 lg:grid-cols-2'
            onSubmit={form.handleSubmit(submitGuide)}
          >
            <div className='space-y-4'>
              <FormField
                control={form.control}
                name='title'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Title')}</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='summary'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Summary')}</FormLabel>
                    <FormControl>
                      <Textarea rows={2} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className='grid gap-4 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='slug'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Slug')}</FormLabel>
                      <FormControl>
                        <Input placeholder='getting-started' {...field} />
                      </FormControl>
                      <FormDescription>
                        {t('Lowercase letters, numbers, and hyphens only.')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='format'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Format')}</FormLabel>
                      <Select
                        value={field.value}
                        onValueChange={field.onChange}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value='markdown'>Markdown</SelectItem>
                            <SelectItem value='html'>HTML</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name='content'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Content')}</FormLabel>
                    <FormControl>
                      <Textarea
                        className='min-h-80 font-mono text-sm'
                        {...field}
                        ref={(element) => {
                          field.ref(element)
                          contentTextareaRef.current = element
                        }}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        '图片使用可访问的 http(s) 地址，视频使用 MP4 直链；可用下方按钮插入 HTML。显示时会过滤危险 HTML。'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className='flex flex-wrap gap-2'>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  disabled={uploadingMedia !== null}
                  onClick={() => {
                    rememberSelection()
                    imageInputRef.current?.click()
                  }}
                >
                  <Image className='mr-2 h-4 w-4' />
                  {uploadingMedia === 'image' ? t('上传中...') : t('选择图片')}
                </Button>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  disabled={uploadingMedia !== null}
                  onClick={() => {
                    rememberSelection()
                    videoInputRef.current?.click()
                  }}
                >
                  <Video className='mr-2 h-4 w-4' />
                  {uploadingMedia === 'video' ? t('上传中...') : t('选择视频')}
                </Button>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  disabled={uploadingMedia !== null}
                  onClick={() => insertMediaTag('image')}
                >
                  <Link2 className='mr-2 h-4 w-4' />
                  {t('插入图片地址')}
                </Button>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  disabled={uploadingMedia !== null}
                  onClick={() => insertMediaTag('video')}
                >
                  <Link2 className='mr-2 h-4 w-4' />
                  {t('插入视频地址')}
                </Button>
                <input
                  ref={imageInputRef}
                  className='hidden'
                  type='file'
                  accept='image/png,image/jpeg,image/gif,image/webp'
                  onChange={(event) => void handleMediaFile(event, 'image')}
                />
                <input
                  ref={videoInputRef}
                  className='hidden'
                  type='file'
                  accept='video/mp4,video/webm,video/quicktime'
                  onChange={(event) => void handleMediaFile(event, 'video')}
                />
              </div>
              <div className='grid gap-4 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='order'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Order')}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          value={field.value}
                          onChange={(event) =>
                            field.onChange(Number(event.target.value))
                          }
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='published'
                  render={({ field }) => (
                    <FormItem className='flex items-center gap-3 pt-7'>
                      <FormControl>
                        <Checkbox
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                      <FormLabel className='m-0'>{t('Published')}</FormLabel>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>

            <div className='min-w-0'>
              <div className='mb-2 text-sm font-medium'>
                {t('Live Preview')}
              </div>
              <div className='bg-background min-h-96 overflow-auto rounded-md border p-5'>
                {previewContent ? (
                  <RichContent
                    content={previewContent}
                    mode={previewFormat}
                    htmlVariant='isolated'
                  />
                ) : (
                  <div className='text-muted-foreground text-sm'>
                    {t('Preview will appear here.')}
                  </div>
                )}
              </div>
            </div>
          </form>
        </Form>
      </Dialog>

      <AlertDialog
        open={deleteMode !== null}
        onOpenChange={(open) => !open && setDeleteMode(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Delete guide?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'The selected guide content will be removed after you save settings.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <AlertDialogAction variant='destructive' onClick={confirmDelete}>
              {t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  )
}
