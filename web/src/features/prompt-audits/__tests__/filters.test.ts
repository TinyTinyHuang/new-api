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
import assert from 'node:assert/strict'
import { describe, test } from 'vitest'

import {
  firstColumnFilterValue,
  promptAuditSelectOptions,
  withSelectedOption,
} from '../lib/filters'

describe('prompt audit dropdown filters', () => {
  test('maps distinct names into searchable select options and drops blanks', () => {
    assert.deepEqual(promptAuditSelectOptions(['alice', '', 'bob']), [
      { label: 'alice', value: 'alice' },
      { label: 'bob', value: 'bob' },
    ])
  })

  test('reads the selected username or token from a single-select column filter', () => {
    assert.equal(
      firstColumnFilterValue(
        [
          { id: 'username', value: ['alice'] },
          { id: 'token_name', value: ['ops-key'] },
        ],
        'username'
      ),
      'alice'
    )
    assert.equal(
      firstColumnFilterValue(
        [{ id: 'token_name', value: ['ops-key'] }],
        'token_name'
      ),
      'ops-key'
    )
    assert.equal(firstColumnFilterValue([], 'username'), '')
  })

  test('keeps a selected value visible even when it is missing from loaded options', () => {
    assert.deepEqual(
      withSelectedOption([{ label: 'bob', value: 'bob' }], 'alice'),
      [
        { label: 'alice', value: 'alice' },
        { label: 'bob', value: 'bob' },
      ]
    )
  })
})
