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
import { useQuery } from '@tanstack/react-query'
import { MessageSquareWarning } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { getPromptAuditByRequestId } from '@/features/prompt-audits/api'

interface PromptAuditSectionProps {
  requestId?: string
  enabled: boolean
}

export function PromptAuditSection({
  requestId,
  enabled,
}: PromptAuditSectionProps) {
  const { t } = useTranslation()
  const { data } = useQuery({
    queryKey: ['prompt-audit', requestId],
    queryFn: async () => {
      if (!requestId) return null
      const result = await getPromptAuditByRequestId(requestId)
      if (!result.success) return null
      return result.data
    },
    enabled: enabled && !!requestId,
  })

  if (!enabled || !requestId || !data?.last_user_text) {
    return null
  }

  return (
    <div className='min-w-0 space-y-1.5'>
      <div className='flex items-center gap-1.5 text-xs font-semibold'>
        <MessageSquareWarning className='size-3.5' />
        {t('Last User Message')}
        {data.blocked && (
          <StatusBadge
            label={t('Blocked')}
            variant='red'
            size='sm'
            copyable={false}
          />
        )}
      </div>
      <div className='bg-muted/30 min-w-0 space-y-1 overflow-hidden rounded-md border p-2.5'>
        <pre className='max-h-48 overflow-y-auto font-sans text-xs leading-relaxed wrap-break-word whitespace-pre-wrap'>
          {data.last_user_text}
        </pre>
        {data.blocked_words && (
          <div className='text-muted-foreground text-xs'>
            {t('Blocked Words')}: {data.blocked_words}
          </div>
        )}
      </div>
    </div>
  )
}
