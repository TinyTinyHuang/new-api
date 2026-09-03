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
  GetPromptAuditsParams,
  PromptAudit,
  PromptAuditApiResponse,
  PromptAuditFilterOptions,
  PromptAuditsPage,
} from './types'

export async function getPromptAudits(
  params: GetPromptAuditsParams = {}
): Promise<PromptAuditApiResponse<PromptAuditsPage>> {
  const query = new URLSearchParams()
  query.set('p', String(params.p || 1))
  query.set('page_size', String(params.page_size || 20))
  if (params.username) query.set('username', params.username)
  if (params.token_name) query.set('token_name', params.token_name)
  if (params.model_name) query.set('model_name', params.model_name)
  if (params.keyword) query.set('keyword', params.keyword)
  if (params.request_id) query.set('request_id', params.request_id)
  if (params.blocked != null) query.set('blocked', String(params.blocked))
  if (params.start_timestamp)
    query.set('start_timestamp', String(params.start_timestamp))
  if (params.end_timestamp)
    query.set('end_timestamp', String(params.end_timestamp))
  const res = await api.get(`/api/log/prompt-audits?${query.toString()}`)
  return res.data
}

export async function getPromptAuditFilterOptions(): Promise<
  PromptAuditApiResponse<PromptAuditFilterOptions>
> {
  const res = await api.get('/api/log/prompt-audit-filters')
  return res.data
}

export async function getPromptAuditByRequestId(
  requestId: string
): Promise<PromptAuditApiResponse<PromptAudit | null>> {
  const res = await api.get('/api/log/prompt-audit', {
    params: { request_id: requestId },
  })
  return res.data
}
