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
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { DataTableColumnHeader } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { formatTimestampToDate } from '@/lib/format'

import type { PromptAudit } from '../types'

export function usePromptAuditsColumns(
  onView: (audit: PromptAudit) => void
): ColumnDef<PromptAudit>[] {
  const { t } = useTranslation()
  return [
    {
      accessorKey: 'created_at',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Time')} />
      ),
      cell: ({ row }) => formatTimestampToDate(row.original.created_at),
      size: 170,
    },
    {
      accessorKey: 'username',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Username')} />
      ),
      cell: ({ row }) => row.original.username || '-',
      size: 120,
    },
    {
      accessorKey: 'token_name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Token')} />
      ),
      cell: ({ row }) => row.original.token_name || '-',
      size: 120,
    },
    {
      accessorKey: 'model_name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Model')} />
      ),
      cell: ({ row }) => row.original.model_name || '-',
      size: 140,
    },
    {
      accessorKey: 'last_user_text',
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={t('Last User Message')}
        />
      ),
      cell: ({ row }) => {
        const text = row.original.last_user_text || '-'
        return (
          <button
            type='button'
            className='hover:text-foreground line-clamp-2 max-w-[28rem] text-left text-xs break-all'
            onClick={() => onView(row.original)}
          >
            {text}
          </button>
        )
      },
    },
    {
      accessorKey: 'blocked',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Status')} />
      ),
      cell: ({ row }) =>
        row.original.blocked ? (
          <StatusBadge
            label={t('Blocked')}
            variant='red'
            size='sm'
            copyable={false}
          />
        ) : (
          <StatusBadge
            label={t('Success')}
            variant='green'
            size='sm'
            copyable={false}
          />
        ),
      size: 90,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => (
        <Button
          variant='ghost'
          size='sm'
          onClick={() => onView(row.original)}
        >
          {t('View')}
        </Button>
      ),
      size: 80,
    },
  ]
}
