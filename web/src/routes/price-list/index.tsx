import { createFileRoute, redirect } from '@tanstack/react-router'

import { PriceList } from '@/features/price-list'
import { getFreshModuleAccess } from '@/lib/nav-modules'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/price-list/')({
  beforeLoad: async ({ location }) => {
    const access = await getFreshModuleAccess('priceList')
    if (!access.enabled) {
      throw redirect({ to: '/' })
    }
    if (access.requireAuth) {
      const { auth } = useAuthStore.getState()
      if (!auth.user) {
        throw redirect({
          to: '/sign-in',
          search: { redirect: location.href },
        })
      }
    }
  },
  component: PriceList,
})
