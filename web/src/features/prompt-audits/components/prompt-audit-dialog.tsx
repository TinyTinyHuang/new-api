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
import { Copy, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { formatTimestampToDate } from '@/lib/format'

import type { PromptAudit } from '../types'

interface PromptAuditDialogProps {
  audit: PromptAudit | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function PromptAuditDialog({
  audit,
  open,
  onOpenChange,
}: PromptAuditDialogProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const text = audit?.last_user_text ?? ''

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Prompt Audits')}
      description={t('View the last user message sent in this request')}
      contentClassName='sm:max-w-lg'
      contentHeight='auto'
      bodyClassName='space-y-4'
    >
      {audit && (
        <ScrollArea className='max-h-[500px] pr-4'>
          <div className='space-y-3 py-2'>
            <div className='text-muted-foreground space-y-1 text-xs'>
              <div>
                {t('Time')}: {formatTimestampToDate(audit.created_at)}
              </div>
              <div>
                {t('Username')}: {audit.username || '-'}
              </div>
              <div>
                {t('Token')}: {audit.token_name || '-'}
              </div>
              <div>
                {t('Model')}: {audit.model_name || '-'}
              </div>
              <div>
                {t('Request ID')}: {audit.request_id || '-'}
              </div>
              {audit.blocked && (
                <StatusBadge
                  label={t('Blocked')}
                  variant='red'
                  size='sm'
                  copyable={false}
                />
              )}
              {audit.blocked_words && (
                <div>
                  {t('Blocked Words')}: {audit.blocked_words}
                </div>
              )}
            </div>
            <div className='space-y-2'>
              <Label className='text-sm font-semibold'>
                {t('Last User Message')}
              </Label>
              <div className='bg-muted/50 relative rounded-md border p-3'>
                <Button
                  variant='ghost'
                  size='sm'
                  className='absolute top-2 right-2 h-8 w-8 p-0'
                  onClick={() => copyToClipboard(text)}
                  title={t('Copy to clipboard')}
                >
                  {copiedText === text ? (
                    <Check className='size-4 text-green-600' />
                  ) : (
                    <Copy className='size-4' />
                  )}
                </Button>
                <p className='pr-10 text-sm leading-relaxed break-words whitespace-pre-wrap'>
                  {text || '-'}
                </p>
              </div>
            </div>
          </div>
        </ScrollArea>
      )}
    </Dialog>
  )
}
