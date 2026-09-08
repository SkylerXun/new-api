import { Link2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

import type { UserProfile } from '../types'

type LinkedAccountsCardProps = {
  profile: UserProfile | null
  loading: boolean
}

export function LinkedAccountsCard({ profile, loading }: LinkedAccountsCardProps) {
  const { t } = useTranslation()
  const linkage = profile?.account_linkage

  if (loading || !linkage || linkage.relation === 'none') return null

  return (
    <Card data-card-hover='false'>
      <CardHeader className='flex flex-row items-start gap-3'>
        <div className='bg-primary/10 text-primary rounded-lg p-2'>
          <Link2 className='h-4 w-4' />
        </div>
        <div>
          <div className='font-semibold'>{t('Linked accounts')}</div>
          <div className='text-muted-foreground text-sm'>
            {linkage.relation === 'main'
              ? t('This is the main account. Linked subaccounts are shown below.')
              : t('This account is linked to the following main account.')}
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-2'>
        {linkage.linked_accounts.map((account) => (
          <div key={account.id} className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
            <div className='min-w-0'>
              <div className='truncate text-sm font-medium'>
                {account.display_name || account.username}
                <span className='text-muted-foreground ml-2'>@{account.username}</span>
              </div>
              <div className='text-muted-foreground text-xs'>
                {t('User ID')} {account.id} · {new Date(account.created_at * 1000).toLocaleDateString()}
              </div>
            </div>
			<div className='flex items-center gap-2'>
				<Badge variant='secondary'>
					{account.relation === 'main'
						? t('Main account')
						: t('Subaccount')}
				</Badge>
                <Badge variant={account.status === 'enabled' ? 'secondary' : 'destructive'}>
                  {account.status === 'enabled' ? t('Enabled') : t('Disabled')}
				</Badge>
			</div>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
