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
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

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
import { Textarea } from '@/components/ui/textarea'

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

type StatementSettingsValues = {
  statement_setting: {
    contact_email: string
    issuer_address: string
  }
}

export function StatementSettingsSection({
  defaultValues,
}: {
  defaultValues: StatementSettingsValues
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const schema = z.object({
    statement_setting: z.object({
      contact_email: z.string().email().or(z.literal('')),
      issuer_address: z.string().max(300),
    }),
  })
  const { form, handleSubmit, handleReset, isDirty, isSubmitting } =
    useSettingsForm<StatementSettingsValues>({
      resolver: zodResolver(schema) as Resolver<
        StatementSettingsValues,
        unknown,
        StatementSettingsValues
      >,
      defaultValues,
      onSubmit: async (_data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          await updateOption.mutateAsync({ key, value: String(value ?? '') })
        }
      },
    })

  return (
    <>
      <FormNavigationGuard when={isDirty} />
      <SettingsSection title={t('Consumption Statement')}>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Configure the issuer identity snapshotted into statements. Compliance notices cannot be removed.'
          )}
        </p>
        <Form {...form}>
          <SettingsForm onSubmit={handleSubmit}>
            <SettingsPageFormActions
              onSave={handleSubmit}
              onReset={handleReset}
              isSaving={isSubmitting || updateOption.isPending}
              isResetDisabled={!isDirty}
            />
            <FormDirtyIndicator isDirty={isDirty} />
            <SettingsFormGrid>
              <FormField
                control={form.control}
                name='statement_setting.contact_email'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Statement contact email')}</FormLabel>
                    <FormControl>
                      <Input {...field} type='email' />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <SettingsFormGridItem span='full'>
                <FormField
                  control={form.control}
                  name='statement_setting.issuer_address'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Issuer address')}</FormLabel>
                      <FormControl>
                        <Textarea {...field} rows={3} />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'This is issuer contact information, not tax registration information.'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </SettingsFormGridItem>
            </SettingsFormGrid>
          </SettingsForm>
        </Form>
      </SettingsSection>
    </>
  )
}
