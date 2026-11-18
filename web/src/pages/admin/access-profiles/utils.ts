import { AccessFeature, AccessFeatures } from '@/constants/access-profile';
import {
  AccessFeatureSwitches,
  IAccessProfile,
} from '@/interfaces/database/access-profile';

/** Profiles whose name or description contains the keyword, ignoring case. */
export function filterAccessProfiles(
  profiles: IAccessProfile[],
  keyword: string,
) {
  const needle = keyword.trim().toLowerCase();
  if (!needle) {
    return profiles;
  }
  return profiles.filter(
    (profile) =>
      profile.name.toLowerCase().includes(needle) ||
      profile.description.toLowerCase().includes(needle),
  );
}

/** Features a profile turns off, in display order. Missing switches are on. */
export function getDisabledFeatures(
  features: AccessFeatureSwitches,
): AccessFeature[] {
  return AccessFeatures.filter((feature) => features[feature] === false);
}
