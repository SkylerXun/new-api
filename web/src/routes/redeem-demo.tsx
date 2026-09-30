import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/redeem-demo')({
  beforeLoad: () => { throw redirect({ to: '/wallet' }) },
})
