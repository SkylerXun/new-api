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
import { createFileRoute } from '@tanstack/react-router'
import { ExternalLink } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

const PURCHASE_URL =
  import.meta.env.PUBLIC_PURCHASE_URL || 'https://catfk.com/shop/HDN89CZZ'

export const Route = createFileRoute('/_authenticated/purchase/')({
  component: PurchasePage,
})

function PurchasePage() {
  const { t } = useTranslation()

  return (
    <div className='flex h-full min-h-0 flex-col'>
      <div className='flex h-12 shrink-0 items-center justify-end border-b px-4'>
        <Button
          variant='outline'
          size='sm'
          render={<a href={PURCHASE_URL} target='_blank' rel='noreferrer' />}
        >
          <ExternalLink aria-hidden='true' />
          {t('Open in new window')}
        </Button>
      </div>
      <iframe
        src={PURCHASE_URL}
        className='min-h-0 flex-1 border-0'
        sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-scripts allow-top-navigation-by-user-activation'
        title={t('Purchase Credits')}
      />
    </div>
  )
}
