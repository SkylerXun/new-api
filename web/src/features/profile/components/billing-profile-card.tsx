import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/lib/api'

export function BillingProfileCard() {
  const { t } = useTranslation()
  const [username, setUsername] = useState('')
  const [contact, setContact] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setLoading(true)
    void api.get('/api/user/billing-profile').then((res) => {
      const data = res.data?.data
      if (res.data?.success && data) {
        setUsername(data.billing_username || data.effective_username || '')
        setContact(data.billing_contact || data.effective_contact || '')
      }
    }).finally(() => setLoading(false))
  }, [])

  const save = async () => {
    setSaving(true)
    try {
      const res = await api.put('/api/user/billing-profile', { billing_username: username.trim(), billing_contact: contact.trim() })
      if (!res.data?.success) throw new Error(res.data?.message)
      toast.success(t('Billing profile saved'))
    } catch (error) {
      toast.error(error instanceof Error && error.message ? error.message : t('Failed to save billing profile'))
    } finally { setSaving(false) }
  }

  return (
    <Card>
      <CardHeader>
        <div className='font-semibold'>{t('Billing profile')}</div>
        <div className='text-muted-foreground text-sm'>{t('These values are used for future statements. Existing statements remain unchanged.')}</div>
      </CardHeader>
      <CardContent className='grid gap-4 sm:grid-cols-2'>
        <div className='space-y-2'>
          <Label htmlFor='profile-billing-username'>{t('Real name')}</Label>
          <Input id='profile-billing-username' maxLength={120} disabled={loading} value={username} onChange={(event) => setUsername(event.target.value)} />
        </div>
        <div className='space-y-2'>
          <Label htmlFor='profile-billing-contact'>{t('Contact information')}</Label>
          <Input id='profile-billing-contact' maxLength={300} disabled={loading} value={contact} onChange={(event) => setContact(event.target.value)} />
        </div>
        <div className='sm:col-span-2'><Button onClick={save} disabled={saving || loading}>{t('Save billing profile')}</Button></div>
      </CardContent>
    </Card>
  )
}
