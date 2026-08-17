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
import { useMemo } from 'react'
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

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'
import {
  BILLING_CURVE_OPTION_KEY,
  createBillingCurveConfigSchema,
  parseBillingCurveConfig,
  serializeBillingCurveConfig,
  type BillingCurveConfig,
} from './progressive-billing-config'

type ProgressiveBillingSectionProps = {
  defaultValue: string
}

export function ProgressiveBillingSection(
  props: ProgressiveBillingSectionProps
) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const defaultValues = useMemo(
    () => parseBillingCurveConfig(props.defaultValue),
    [props.defaultValue]
  )
  const schema = useMemo(
    () => createBillingCurveConfigSchema((key) => t(key)),
    [t]
  )
  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<BillingCurveConfig>({
      resolver: zodResolver(schema) as Resolver<
        BillingCurveConfig,
        unknown,
        BillingCurveConfig
      >,
      defaultValues,
      onSubmit: async (values) => {
        await updateOption.mutateAsync({
          key: BILLING_CURVE_OPTION_KEY,
          value: serializeBillingCurveConfig(values),
        })
      },
    })

  const disabled = updateOption.isPending || isSubmitting
  const k1 = form.watch('k1')
  const k2 = form.watch('k2')

  return (
    <SettingsSection title={t('Progressive Billing')}>
      <FormNavigationGuard when={isDirty} />
      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={disabled}
            isSaveDisabled={!isDirty}
            saveLabel={t('Save Progressive Billing')}
          />
          <FormDirtyIndicator isDirty={isDirty} />

          <SettingsFormGrid>
            <SettingsFormGridItem span='full'>
              <FormField
                control={form.control}
                name='enabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Enable progressive billing')}</FormLabel>
                      <FormDescription>
                        {t(
                          "Apply a multiplier that grows with each user's cumulative base usage. It is applied after model and group pricing; balance adjustments do not change progress."
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
            </SettingsFormGridItem>

            <FormField
              control={form.control}
              name='k1'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Initial multiplier (K1)')}</FormLabel>
                  <FormControl>
                    <InputGroup>
                      <InputGroupInput
                        type='number'
                        min={0.000001}
                        max={1_000_000}
                        step='0.01'
                        {...safeNumberFieldProps(field)}
                        disabled={disabled}
                      />
                      <InputGroupAddon align='inline-end'>x</InputGroupAddon>
                    </InputGroup>
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Multiplier used before the cumulative base usage threshold.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='k2'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Final multiplier (K2)')}</FormLabel>
                  <FormControl>
                    <InputGroup>
                      <InputGroupInput
                        type='number'
                        min={k1 || 0.000001}
                        max={1_000_000}
                        step='0.01'
                        {...safeNumberFieldProps(field)}
                        disabled={disabled}
                      />
                      <InputGroupAddon align='inline-end'>x</InputGroupAddon>
                    </InputGroup>
                  </FormControl>
                  <FormDescription>
                    {t('Multiplier used after the transition window.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='threshold_usd'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Start threshold (USD)')}</FormLabel>
                  <FormControl>
                    <InputGroup>
                      <InputGroupAddon>$</InputGroupAddon>
                      <InputGroupInput
                        type='number'
                        min={0}
                        max={1_000_000_000_000}
                        step='0.01'
                        {...safeNumberFieldProps(field)}
                        disabled={disabled}
                      />
                    </InputGroup>
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Cumulative base usage at which the multiplier starts increasing.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='window_usd'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Transition window (USD)')}</FormLabel>
                  <FormControl>
                    <InputGroup>
                      <InputGroupAddon>$</InputGroupAddon>
                      <InputGroupInput
                        type='number'
                        min={0.000001}
                        max={1_000_000_000_000}
                        step='0.01'
                        {...safeNumberFieldProps(field)}
                        disabled={disabled}
                      />
                    </InputGroup>
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Base usage range over which the multiplier rises linearly.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <SettingsFormGridItem span='full'>
              <FormField
                control={form.control}
                name='target_average_k'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Target average multiplier')}</FormLabel>
                    <FormControl>
                      <InputGroup>
                        <InputGroupInput
                          type='number'
                          min={k1 || 0.000001}
                          max={k2 || 1_000_000}
                          step='0.01'
                          {...safeNumberFieldProps(field)}
                          disabled={disabled}
                        />
                        <InputGroupAddon align='inline-end'>x</InputGroupAddon>
                      </InputGroup>
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Operational target for the long-run weighted average multiplier.'
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
  )
}
