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
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface StatementRecipientFieldsProps {
  billingTitle: string
  billingAddress: string
  billingUsername: string
  billingContact: string
  onBillingTitleChange: (value: string) => void
  onBillingAddressChange: (value: string) => void
}

export function StatementRecipientFields(props: StatementRecipientFieldsProps) {
  const { t } = useTranslation()

  return (
    <div className='grid gap-4 sm:grid-cols-2'>
      <div className='space-y-2'>
        <Label htmlFor='statement-title'>
          {t('Reconciliation title (optional)')}
        </Label>
        <Input
          id='statement-title'
          maxLength={120}
          value={props.billingTitle}
          onChange={(event) => props.onBillingTitleChange(event.target.value)}
          placeholder={t('For example: company or department name')}
        />
        <p className='text-muted-foreground text-xs'>
          {t(
            'This is user-supplied reconciliation information, not an invoice title.'
          )}
        </p>
      </div>
      <div className='space-y-2'>
        <Label htmlFor='statement-address'>
          {t('Contact address (optional)')}
        </Label>
        <Input
          id='statement-address'
          maxLength={300}
          value={props.billingAddress}
          onChange={(event) => props.onBillingAddressChange(event.target.value)}
          placeholder={t('For example: Guangdong Province, Shenzhen...')}
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='statement-username'>
          {t('Real name (editable in Profile)')}
        </Label>
        <Input
          id='statement-username'
          maxLength={120}
          value={props.billingUsername}
          readOnly
          className='bg-muted text-muted-foreground'
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='statement-contact'>
          {t('Contact information (editable in Profile)')}
        </Label>
        <Input
          id='statement-contact'
          maxLength={300}
          value={props.billingContact}
          readOnly
          className='bg-muted text-muted-foreground'
        />
      </div>
    </div>
  )
}
