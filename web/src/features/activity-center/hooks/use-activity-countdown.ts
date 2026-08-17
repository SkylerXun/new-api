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

import { useEffect, useState } from 'react'

import { getRemainingSeconds } from '../lib/countdown'

export function useActivityCountdown(
  endsAt: number,
  serverTime: number
): number {
  const [receivedAt, setReceivedAt] = useState(() => Date.now())
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const nextReceivedAt = Date.now()
    setReceivedAt(nextReceivedAt)
    setNow(nextReceivedAt)

    const updateNow = () => setNow(Date.now())
    const intervalId = window.setInterval(updateNow, 1000)
    document.addEventListener('visibilitychange', updateNow)
    window.addEventListener('focus', updateNow)
    return () => {
      window.clearInterval(intervalId)
      document.removeEventListener('visibilitychange', updateNow)
      window.removeEventListener('focus', updateNow)
    }
  }, [endsAt, serverTime])

  return getRemainingSeconds(endsAt, serverTime, now - receivedAt)
}
