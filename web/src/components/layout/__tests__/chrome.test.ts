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
import { describe, expect, test } from 'vitest'

import {
  HEADER_CHROME_CLASSES,
  PUBLIC_NAV_LINK_ACTIVE_CLASSES,
  PUBLIC_NAV_LINK_CLASSES,
} from '../constants'

describe('layout chrome', () => {
  test('console header stays frosted over the tinted canvas', () => {
    const classes = HEADER_CHROME_CLASSES.split(' ')

    expect(classes.includes('bg-background/80')).toBeTruthy()
    expect(classes.includes('backdrop-blur-xl')).toBeTruthy()
    expect(classes.includes('border-b')).toBeTruthy()
    expect(classes.includes('bg-transparent')).toBeFalsy()
  })

  test('public nav links use a pill active state', () => {
    const linkClasses = PUBLIC_NAV_LINK_CLASSES.split(' ')
    const activeClasses = PUBLIC_NAV_LINK_ACTIVE_CLASSES.split(' ')

    expect(linkClasses.includes('rounded-full')).toBeTruthy()
    expect(activeClasses.includes('bg-card')).toBeTruthy()
    expect(activeClasses.includes('text-primary')).toBeTruthy()
  })
})
