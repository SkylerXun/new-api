import { api } from '@/lib/api'

import type { SystemTask } from '../types'

export type RiskExemption = {
  user_id: number
  reason: string
  created_by: number
  created_at: number
  expires_at: number
  enabled: boolean
}

export async function getRiskExemptions() {
  const res = await api.get<{ success: boolean; data?: RiskExemption[]; message?: string }>('/api/user/risk/exemptions')
  return res.data
}

export async function addRiskExemption(input: { user_id: number; reason: string; expires_at?: number }) {
  const res = await api.post<{ success: boolean; data?: RiskExemption; message?: string }>('/api/user/risk/exemptions', input)
  return res.data
}

export async function removeRiskExemption(userId: number) {
  const res = await api.delete<{ success: boolean; message?: string }>(`/api/user/risk/exemptions/${userId}`)
  return res.data
}

export type RiskBackfillStatus =
  | 'pending'
  | 'running'
  | 'pause_requested'
  | 'paused'
  | 'succeeded'
  | 'failed'

export type RiskBackfillRun = {
  run_id: string
  task_id: string
  status: RiskBackfillStatus
  min_user_id: number
  max_user_id: number
  created_from: number
  created_to: number
  page_size: number
  last_user_id: number
  total_users: number
  processed_users: number
  candidate_groups: number
  candidate_accounts: number
  created_by: number
  created_at: number
  updated_at: number
  error: string
}

export type RiskBackfillOptions = {
  min_user_id?: number
  max_user_id?: number
  created_from?: number
  created_to?: number
  page_size?: number
}

export type RiskReviewMember = {
  id: number
  username: string
  display_name: string
  status: 'enabled' | 'disabled'
  created_at: number
  relation: 'main' | 'subaccount'
}

export type RiskAccountReview = {
  id: number
  kind: 'historical_cluster' | 'cluster_merge'
  status: 'pending' | 'approved' | 'rejected' | 'failed'
  proposed_main_user_id: number
  members: RiskReviewMember[]
  evidence: string
  policy_version: string
  created_at: number
  updated_at: number
  resolution: string
  created_by: number
  created_by_username?: string
  resolved_by: number
  resolved_by_username?: string
  resolved_at: number
}

export type RiskReviewStatus = 'pending' | 'approved' | 'rejected' | 'failed'

export type RiskAccountAction = {
  id: number
  user_id: number
  username: string
  display_name: string
  main_user_id: number
  main_username: string
  cluster_id: string
  action: string
  source: 'system' | 'admin'
  rule: string
  policy_version: string
  created_by: number
  created_by_username?: string
  created_at: number
}

type ApiResponse<T> = { success: boolean; data?: T; message?: string }

type RiskBackfillStartResponse = ApiResponse<{
  run: RiskBackfillRun
  task: SystemTask
  created: boolean
}>

export async function startRiskAccountBackfill(options: RiskBackfillOptions) {
  const res = await api.post<RiskBackfillStartResponse>(
    '/api/user/risk/backfill',
    options
  )
  return res.data
}

export async function getRiskAccountBackfill(runId = 'latest') {
  const res = await api.get<ApiResponse<RiskBackfillRun | null>>(
    `/api/user/risk/backfill/${runId}`
  )
  return res.data
}

export async function pauseRiskAccountBackfill(runId: string) {
  const res = await api.post<ApiResponse<RiskBackfillRun>>(
    `/api/user/risk/backfill/${runId}/pause`
  )
  return res.data
}

export async function resumeRiskAccountBackfill(runId: string) {
  const res = await api.post<
    ApiResponse<{ run: RiskBackfillRun; task: SystemTask }>
  >(`/api/user/risk/backfill/${runId}/resume`)
  return res.data
}

export async function getRiskAccountReviews(status = 'pending') {
  const res = await api.get<
    ApiResponse<{
      items: RiskAccountReview[]
      total: number
      page: number
      page_size: number
    }>
  >('/api/user/risk/reviews', { params: { status, page: 1, page_size: 100 } })
  return res.data
}

export async function getRiskAccountActions(params: {
  action?: string
  source?: 'system' | 'admin'
  user_id?: number
  page?: number
  page_size?: number
} = {}) {
  const res = await api.get<
    ApiResponse<{
      items: RiskAccountAction[]
      total: number
      page: number
      page_size: number
    }>
  >('/api/user/risk/actions', { params })
  return res.data
}

export async function approveRiskAccountReview(reviewId: number) {
  const res = await api.post<ApiResponse<{ review_id: number }>>(
    `/api/user/risk/reviews/${reviewId}/approve`,
    {}
  )
  return res.data
}

export async function rejectRiskAccountReview(reviewId: number) {
  const res = await api.post<ApiResponse<{ review_id: number }>>(
    `/api/user/risk/reviews/${reviewId}/reject`,
    {}
  )
  return res.data
}

export async function detachRiskAccount(userId: number, reason: string) {
  const res = await api.post<ApiResponse<{ affected_user_ids: number[] }>>(
    `/api/user/risk/accounts/${userId}/detach`,
    { reenable: true, reason }
  )
  return res.data
}
