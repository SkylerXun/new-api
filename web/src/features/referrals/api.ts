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
import { api } from '@/lib/api'

import type {
  AffiliateTransferRequest,
  ApiResponse,
  ReferralSettings,
  ReferralUserData,
} from './types'

export async function getReferralCode(): Promise<ApiResponse<string>> {
  const response = await api.get('/api/user/aff')
  return response.data
}

export async function getReferralUser(): Promise<
  ApiResponse<ReferralUserData>
> {
  const response = await api.get('/api/user/self')
  return response.data
}

export async function getReferralSettings(): Promise<
  ApiResponse<ReferralSettings>
> {
  const response = await api.get('/api/user/topup/info')
  return response.data
}

export async function transferReferralRewards(
  request: AffiliateTransferRequest
): Promise<ApiResponse> {
  const response = await api.post('/api/user/aff_transfer', request)
  return response.data
}
