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

export type CountdownParts = {
  days: number
  hours: number
  minutes: number
  seconds: number
}

export function getRemainingSeconds(
  endsAt: number,
  serverTime: number,
  elapsedMilliseconds: number
): number {
  if (
    !Number.isFinite(endsAt) ||
    !Number.isFinite(serverTime) ||
    !Number.isFinite(elapsedMilliseconds)
  ) {
    return 0
  }
  const elapsedSeconds = Math.max(0, Math.floor(elapsedMilliseconds / 1000))
  return Math.max(0, Math.floor(endsAt - serverTime - elapsedSeconds))
}

export function splitCountdown(totalSeconds: number): CountdownParts {
  const safeSeconds = Number.isFinite(totalSeconds)
    ? Math.max(0, Math.floor(totalSeconds))
    : 0
  return {
    days: Math.floor(safeSeconds / 86400),
    hours: Math.floor((safeSeconds % 86400) / 3600),
    minutes: Math.floor((safeSeconds % 3600) / 60),
    seconds: safeSeconds % 60,
  }
}
