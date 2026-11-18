import Empty from '@/components/empty/empty';
import { useIsAccessFeaturePathBlocked } from '@/hooks/use-access-profile-request';
import { PropsWithChildren } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-router';

/**
 * Replaces a page with a notice when the signed-in user's access profile turns
 * off the feature that owns the current path. The API rejects the page's
 * requests either way; this keeps users from landing on a broken page.
 */
export function AccessFeatureRouteGuard({ children }: PropsWithChildren) {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const blocked = useIsAccessFeaturePathBlocked(pathname);

  if (blocked) {
    return (
      <Empty className="size-full" text={t('accessProfile.featureDisabled')} />
    );
  }

  return children;
}
