export const RELEASE_PODS_CHANGED_EVENT = 'roncin:release-pods-changed';

export function notifyReleasePodsChanged(orderId: string) {
  window.dispatchEvent(
    new CustomEvent(RELEASE_PODS_CHANGED_EVENT, { detail: { orderId } }),
  );
}
