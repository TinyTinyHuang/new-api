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
import { getRouteApi } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DataTablePage, useDataTable } from '@/components/data-table'
import { useTableUrlState } from '@/hooks/use-table-url-state'

import { getPromptAudits } from '../api'
import type { PromptAudit } from '../types'
import { PromptAuditDialog } from './prompt-audit-dialog'
import { usePromptAuditsColumns } from './prompt-audits-columns'

const route = getRouteApi('/_authenticated/prompt-audits/')

export function PromptAuditsTable() {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<PromptAudit | null>(null)
  const columns = usePromptAuditsColumns(setSelected)

  const {
    globalFilter,
    onGlobalFilterChange,
    columnFilters,
    onColumnFiltersChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: route.useSearch(),
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: 20 },
    globalFilter: { enabled: true, key: 'filter' },
    columnFilters: [
      { columnId: 'username', searchKey: 'username', type: 'string' },
      { columnId: 'blocked', searchKey: 'blocked', type: 'array' },
    ],
  })

  const usernameFilter =
    (columnFilters.find((filter) => filter.id === 'username')?.value as
      | string
      | undefined) ?? ''
  const blockedFilter =
    (columnFilters.find((filter) => filter.id === 'blocked')?.value as
      | string[]
      | undefined) ?? []
  const blockedOnly = blockedFilter.includes('true')

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'prompt-audits',
      pagination.pageIndex + 1,
      pagination.pageSize,
      globalFilter,
      usernameFilter,
      blockedOnly,
    ],
    queryFn: async () => {
      const result = await getPromptAudits({
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        keyword: globalFilter,
        username: usernameFilter,
        blocked: blockedOnly ? true : undefined,
      })
      if (!result.success) {
        toast.error(result.message || t('Failed to load logs'))
        return { items: [] as PromptAudit[], total: 0 }
      }
      return {
        items: result.data?.items || [],
        total: result.data?.total || 0,
      }
    },
    placeholderData: (previousData) => previousData,
  })

  const { table } = useDataTable({
    data: data?.items || [],
    columns,
    columnFilters,
    globalFilter,
    pagination,
    onPaginationChange,
    onGlobalFilterChange,
    onColumnFiltersChange,
    manualPagination: true,
    manualFiltering: true,
    enableSorting: false,
    totalCount: data?.total || 0,
    ensurePageInRange,
  })

  return (
    <>
      <DataTablePage
        table={table}
        columns={columns}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyTitle={t('No Prompt Audits Found')}
        emptyDescription={t(
          'No prompt audits match the current filters.'
        )}
        skeletonKeyPrefix='prompt-audits-skeleton'
        toolbarProps={{
          searchPlaceholder: t('Search prompt text...'),
          searchDebounceMs: 500,
          filters: [
            {
              columnId: 'blocked',
              title: t('Status'),
              options: [
                { label: t('Only Blocked'), value: 'true' },
              ],
              singleSelect: true,
            },
          ],
        }}
      />
      <PromptAuditDialog
        audit={selected}
        open={selected != null}
        onOpenChange={(open) => {
          if (!open) setSelected(null)
        }}
      />
    </>
  )
}
