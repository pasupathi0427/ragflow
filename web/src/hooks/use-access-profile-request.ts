import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

import { AccessFeature, AccessFeatureRoutes } from '@/constants/access-profile';
import { IMyAccessFeatures } from '@/interfaces/database/access-profile';
import accessProfileService from '@/services/access-profile-service';

export const AccessProfileKeys = {
  all: ['accessProfile'] as const,
  myFeatures: () => [...AccessProfileKeys.all, 'myFeatures'] as const,
};

// The API server caches profile lookups for 30 seconds, so refetching more
// often than that cannot show a newer answer.
const MyFeaturesStaleTime = 30_000;

export const useFetchMyAccessFeatures = () => {
  const {
    data,
    isFetching: loading,
    isFetched: fetched,
  } = useQuery({
    queryKey: AccessProfileKeys.myFeatures(),
    staleTime: MyFeaturesStaleTime,
    queryFn: async () => {
      const { data } = await accessProfileService.getMyFeatures({}, true);
      return (data?.data ?? null) as IMyAccessFeatures | null;
    },
  });

  return { data, loading, fetched };
};

/**
 * Whether background requests of a feature may run: true only once the
 * switches have loaded and the feature is not off, so a disabled feature never
 * triggers a rejected request.
 */
export const useCanRequestAccessFeature = (feature: AccessFeature) => {
  const { data, fetched } = useFetchMyAccessFeatures();
  return fetched && data?.features?.[feature] !== false;
};

/**
 * Whether the signed-in user's access profile enables a feature. Until the
 * switches load, features count as enabled; the API enforces the real switch.
 */
export const useIsAccessFeatureEnabled = (feature: AccessFeature) => {
  const { data } = useFetchMyAccessFeatures();
  return data?.features?.[feature] !== false;
};

/** Returns the items whose feature is enabled. Items without a feature stay. */
export function useAccessFeatureFilter<T>(
  items: T[],
  getFeature: (item: T) => AccessFeature | undefined,
) {
  const { data } = useFetchMyAccessFeatures();
  const features = data?.features;

  return useMemo(
    () =>
      items.filter((item) => {
        const feature = getFeature(item);
        return !feature || features?.[feature] !== false;
      }),
    [items, getFeature, features],
  );
}

export const findAccessFeatureForPath = (pathname: string) =>
  AccessFeatureRoutes.find(
    ({ path }) => pathname === path || pathname.startsWith(`${path}/`),
  )?.feature;

/** Whether the page at the pathname belongs to a feature that is switched off. */
export const useIsAccessFeaturePathBlocked = (pathname: string) => {
  const { data } = useFetchMyAccessFeatures();
  const feature = findAccessFeatureForPath(pathname);
  return Boolean(feature && data?.features?.[feature] === false);
};
