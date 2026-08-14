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
import { Main } from '@/components/layout'
import type { SystemStatus } from '@/features/auth/types'

import { CheckinCalendarCard } from './components/checkin-calendar-card'

interface CheckinProps {
  status: SystemStatus
}

export function Checkin(props: CheckinProps) {
  const checkinEnabled = props.status.checkin_enabled === true
  const turnstileEnabled = Boolean(
    props.status.turnstile_check && props.status.turnstile_site_key
  )

  return (
    <Main>
      <div className='min-h-0 flex-1 overflow-auto px-3 py-3 sm:px-4 sm:py-6'>
        <div className='mx-auto w-full max-w-4xl'>
          <CheckinCalendarCard
            checkinEnabled={checkinEnabled}
            turnstileEnabled={turnstileEnabled}
            turnstileSiteKey={props.status.turnstile_site_key || ''}
          />
        </div>
      </div>
    </Main>
  )
}
