export function isSubscriptionPurchaseBlocked(
  activeSubscriptionCount: number
): boolean {
  return activeSubscriptionCount > 0
}
