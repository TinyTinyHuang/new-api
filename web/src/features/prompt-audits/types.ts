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
export interface PromptAudit {
  id: number
  request_id: string
  user_id: number
  created_at: number
  username: string
  token_name: string
  token_id: number
  model_name: string
  group: string
  ip: string
  relay_format: string
  last_user_text: string
  message_count: number
  blocked: boolean
  blocked_words: string
}

export interface GetPromptAuditsParams {
  p?: number
  page_size?: number
  username?: string
  token_name?: string
  model_name?: string
  keyword?: string
  request_id?: string
  blocked?: boolean
  start_timestamp?: number
  end_timestamp?: number
}

export interface PromptAuditsPage {
  page: number
  page_size: number
  total: number
  items: PromptAudit[]
}

export interface PromptAuditFilterOptions {
  usernames: string[]
  token_names: string[]
}

export interface PromptAuditApiResponse<T> {
  success: boolean
  message: string
  data: T
}
