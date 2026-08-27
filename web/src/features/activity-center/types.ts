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

export type ActivityStatus =
  | 'active'
  | 'claimable'
  | 'claimed'
  | 'credited'
  | 'expired'
  | 'closed'
  | 'unavailable'

export type ActivityAction = {
  to?: string
  label: string
  type?: 'navigate' | 'claim'
  endpoint?: string
}

export type UserActivity = {
  id: string
  type: string
  title: string
  description: string
  status: ActivityStatus
  starts_at: number
  ends_at: number
  remaining_seconds: number
  bonus_percent: number
  reward_quota?: number
  granted_at?: number
  action?: ActivityAction
}

export type ActivityCenterData = {
  server_time: number
  activities: UserActivity[]
  next_cursor?: string
}

export type ActivityCenterResponse = {
  success: boolean
  message: string
  data?: ActivityCenterData
}

export type ClaimActivityResponse = {
  success: boolean
  message: string
  data?: {
    granted: boolean
    reward_quota: number
  }
}
