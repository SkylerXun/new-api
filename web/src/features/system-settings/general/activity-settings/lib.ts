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
import { z } from 'zod'

import type { AllUsersActivityGrantTask, SystemTaskStatus } from '../../types'

export const activityCampaignTypes = ['claimable', 'immediate'] as const

export const activitySettingsSchema = z.object({
  activity_setting: z.object({
    new_user_redeem_bonus_enabled: z.boolean(),
    new_user_redeem_bonus_percent: z.coerce.number().min(0).max(1000),
    new_user_redeem_bonus_window_days: z.coerce.number().int().min(1).max(3650),
  }),
})

export const grantAllUsersSchema = z.object({
  amountUSD: z
    .string()
    .trim()
    .min(1)
    .refine((value) => {
      const amount = Number(value)
      return Number.isFinite(amount) && amount > 0
    }),
  reason: z.string().trim().max(255),
})

export const activityCampaignSchema = z
  .object({
    type: z.enum(activityCampaignTypes),
    title: z.string().trim().min(1).max(128),
    description: z.string().trim().max(4000),
    amountUSD: z
      .string()
      .trim()
      .min(1)
      .refine((value) => {
        const amount = Number(value)
        return Number.isFinite(amount) && amount > 0
      }),
    endsAt: z.string().trim(),
  })
  .superRefine((values, context) => {
    if (
      values.type === 'claimable' &&
      parseActivityCampaignEndAt(values.endsAt) === undefined
    ) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['endsAt'],
      })
    }
  })

export type ActivitySettingsFormValues = z.infer<typeof activitySettingsSchema>
export type GrantAllUsersFormValues = z.infer<typeof grantAllUsersSchema>
export type ActivityCampaignFormValues = z.infer<typeof activityCampaignSchema>

export function parseActivityCampaignEndAt(value: string): number | undefined {
  const timestamp = new Date(value).getTime()
  if (!Number.isFinite(timestamp)) return undefined
  return Math.floor(timestamp / 1000)
}

export function isActiveSystemTaskStatus(status?: SystemTaskStatus): boolean {
  return status === 'pending' || status === 'running'
}

export function isActiveAllUsersGrantTask(
  task?: AllUsersActivityGrantTask | null
): boolean {
  return isActiveSystemTaskStatus(task?.status)
}

export function getAllUsersGrantProgress(
  task?: AllUsersActivityGrantTask | null
): number {
  const progress = task?.state?.progress ?? 0
  return Math.min(100, Math.max(0, progress))
}
