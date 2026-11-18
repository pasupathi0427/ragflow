import { useVisibleIntegrationChildren } from '@/ee/hooks/use-visible-integration-children';
import {
  useAccessFeatureFilter,
  useFetchMyAccessFeatures,
} from '@/hooks/use-access-profile-request';
import { Routes } from '@/routes';
import { Navigate } from 'react-router';
import { getIntegrationAccessFeature } from './app-sidebar/access-features';
import { IntegrationChildren } from './app-sidebar/navigation';

/**
 * Opens the first integrations page the user may see. Waits for the
 * access-profile switches so a disabled page is never the landing page.
 */
export default function IntegrationsIndexRedirect() {
  const { fetched } = useFetchMyAccessFeatures();
  const visibleChildren = useVisibleIntegrationChildren(IntegrationChildren);
  const enabledChildren = useAccessFeatureFilter(
    visibleChildren,
    getIntegrationAccessFeature,
  );

  if (!fetched) {
    return null;
  }

  return <Navigate to={enabledChildren[0]?.path ?? Routes.Root} replace />;
}
