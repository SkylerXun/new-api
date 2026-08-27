/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export function getMonthlyDiscountScale(maximumTierUSD: number): number {
  return Math.max(maximumTierUSD * 1.2, 1)
}

export function getMonthlyDiscountPosition(
  valueUSD: number,
  scaleMaximumUSD: number
): number {
  if (scaleMaximumUSD <= 0) return 0
  return Math.min(100, Math.max(0, (valueUSD / scaleMaximumUSD) * 100))
}
