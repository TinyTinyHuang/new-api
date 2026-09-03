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
export type PromptAuditSelectOption = {
  label: string
  value: string
}

export function promptAuditSelectOptions(
  values: string[] | undefined
): PromptAuditSelectOption[] {
  return (values ?? [])
    .filter((value) => value !== '')
    .map((value) => ({ label: value, value }))
}

export function firstColumnFilterValue(
  columnFilters: Array<{ id: string; value?: unknown }>,
  columnId: string
): string {
  const value = columnFilters.find((filter) => filter.id === columnId)?.value
  if (Array.isArray(value) && typeof value[0] === 'string') {
    return value[0]
  }
  if (typeof value === 'string') {
    return value
  }
  return ''
}

export function withSelectedOption(
  options: PromptAuditSelectOption[],
  selected: string
): PromptAuditSelectOption[] {
  if (!selected || options.some((option) => option.value === selected)) {
    return options
  }
  return [{ label: selected, value: selected }, ...options]
}
