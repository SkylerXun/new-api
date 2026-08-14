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
import { CircleCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'

type RedemptionsCreatedDialogProps = {
  codes: string[] | null
  onOpenChange: (open: boolean) => void
}

export function RedemptionsCreatedDialog({
  codes,
  onOpenChange,
}: RedemptionsCreatedDialogProps) {
  const { t } = useTranslation()
  const createdCodes = codes ?? []
  const createdCodesText = createdCodes.join('\n')

  return (
    <Dialog
      open={codes !== null}
      onOpenChange={onOpenChange}
      title={
        <span className='flex items-center gap-2'>
          <CircleCheck className='text-success size-5' aria-hidden='true' />
          {t('Redemption code(s) created successfully')}
        </span>
      }
      description={t('Successfully created {{count}} redemption codes', {
        count: createdCodes.length,
      })}
      contentClassName='sm:max-w-lg'
      contentHeight='auto'
      showCloseButton={false}
      footer={
        <>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Close')}
          </Button>
          <CopyButton
            value={createdCodesText}
            variant='default'
            size='default'
            className='gap-2'
            iconClassName='size-4'
            aria-label={t('Copy All Codes')}
          >
            {t('Copy All Codes')}
          </CopyButton>
        </>
      }
    >
      <Textarea
        readOnly
        value={createdCodesText}
        rows={Math.min(Math.max(createdCodes.length, 3), 10)}
        wrap='off'
        spellCheck={false}
        aria-label={t('Redemption Codes')}
        className='max-h-64 min-h-24 resize-none font-mono leading-6'
      />
    </Dialog>
  )
}
