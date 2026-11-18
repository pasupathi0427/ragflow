import {
  AccessFeature,
  IntegrationAccessFeatures,
} from '@/constants/access-profile';
import { IntegrationChild, NavItem, NavItemKey } from './navigation';

// Sidebar entries owned by an access-profile feature. Entries without a
// feature are always shown.
export const getNavItemAccessFeature = (item: NavItem) =>
  item.key === NavItemKey.Team ? AccessFeature.Team : undefined;

export const getIntegrationAccessFeature = (child: IntegrationChild) =>
  IntegrationAccessFeatures[child.key];
