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

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import { Switch } from '@/components/ui/switch'

import { FormDirtyIndicator } from '../../components/form-dirty-indicator'
import { FormNavigationGuard } from '../../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../../components/settings-form-layout'
import { SettingsPageFormActions } from '../../components/settings-page-context'
import { useSettingsForm } from '../../hooks/use-settings-form'
import { useUpdateOption } from '../../hooks/use-update-option'
import { activitySettingsSchema, type ActivitySettingsFormValues } from './lib'

type ActivitySettingsFormProps = {
  defaultValues: ActivitySettingsFormValues
}

export function ActivitySettingsForm(props: ActivitySettingsFormProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<ActivitySettingsFormValues>({
      resolver: zodResolver(activitySettingsSchema) as Resolver<
        ActivitySettingsFormValues,
        unknown,
        ActivitySettingsFormValues
      >,
      defaultValues: props.defaultValues,
      onSubmit: async (_values, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          await updateOption.mutateAsync({
            key,
            value: value as string | number | boolean,
          })
        }
      },
    })

  const disabled = updateOption.isPending || isSubmitting

  return (
    <Form {...form}>
      <FormNavigationGuard when={isDirty} />
      <SettingsForm onSubmit={handleSubmit}>
        <SettingsPageFormActions
          onSave={handleSubmit}
          isSaving={disabled}
          isSaveDisabled={!isDirty}
          saveLabel='Save activity settings'
        />
        <FormDirtyIndicator isDirty={isDirty} />

        <FormField
          control={form.control}
          name='activity_setting.new_user_redeem_bonus_enabled'
          render={({ field }) => (
            <SettingsSwitchItem>
              <SettingsSwitchContent>
                <FormLabel>{t('Enable new-user redemption bonus')}</FormLabel>
                <FormDescription>
                  {t(
                    'Give eligible new users a bonus on every redeemed code during the activity window.'
                  )}
                </FormDescription>
              </SettingsSwitchContent>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                  disabled={disabled}
                />
              </FormControl>
              <FormMessage />
            </SettingsSwitchItem>
          )}
        />

        <FormField
          control={form.control}
          name='activity_setting.new_user_redeem_bonus_percent'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Bonus percentage')}</FormLabel>
              <FormControl>
                <InputGroup>
                  <InputGroupInput
                    type='number'
                    min={0}
                    max={1000}
                    step='0.1'
                    {...field}
                    disabled={disabled}
                  />
                  <InputGroupAddon align='inline-end'>%</InputGroupAddon>
                </InputGroup>
              </FormControl>
              <FormDescription>
                {t('Percentage added to the redeemed quota.')}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name='activity_setting.new_user_redeem_bonus_window_days'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Eligibility window')}</FormLabel>
              <FormControl>
                <InputGroup>
                  <InputGroupInput
                    type='number'
                    min={1}
                    max={3650}
                    step={1}
                    {...field}
                    disabled={disabled}
                  />
                  <InputGroupAddon align='inline-end'>
                    {t('days')}
                  </InputGroupAddon>
                </InputGroup>
              </FormControl>
              <FormDescription>
                {t(
                  'Time after registration when every redeemed code can receive a bonus.'
                )}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      </SettingsForm>
    </Form>
  )
}
