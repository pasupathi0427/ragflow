import { AccessFeature } from '@/constants/access-profile';
import {
  findAccessFeatureForPath,
  useAccessFeatureFilter,
  useCanRequestAccessFeature,
  useIsAccessFeatureEnabled,
  useIsAccessFeaturePathBlocked,
} from '@/hooks/use-access-profile-request';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';

const mockGetMyFeatures = jest.fn();

jest.mock('@/services/access-profile-service', () => ({
  __esModule: true,
  default: {
    getMyFeatures: (...args: unknown[]) => mockGetMyFeatures(...args),
  },
}));

const RestrictedFeatures = {
  profile_id: 'p1',
  profile_name: 'Interns',
  features: {
    [AccessFeature.ModelProviders]: true,
    [AccessFeature.DataSources]: false,
    [AccessFeature.ChatChannels]: true,
    [AccessFeature.Mcp]: false,
    [AccessFeature.Team]: true,
    [AccessFeature.Api]: true,
  },
};

function createWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
  };
}

const Items = [
  { key: 'model', feature: AccessFeature.ModelProviders },
  { key: 'data-source', feature: AccessFeature.DataSources },
  { key: 'mcp', feature: AccessFeature.Mcp },
  { key: 'home', feature: undefined },
];
const getFeature = (item: (typeof Items)[number]) => item.feature;

describe('access profile hooks', () => {
  beforeEach(() => {
    mockGetMyFeatures.mockReset();
  });

  it('keeps every item until the switches load, then drops disabled ones', async () => {
    mockGetMyFeatures.mockResolvedValue({
      data: { code: 0, data: RestrictedFeatures },
    });
    const { result } = renderHook(
      () => useAccessFeatureFilter(Items, getFeature),
      { wrapper: createWrapper() },
    );

    expect(result.current.map((item) => item.key)).toEqual([
      'model',
      'data-source',
      'mcp',
      'home',
    ]);
    await waitFor(() =>
      expect(result.current.map((item) => item.key)).toEqual(['model', 'home']),
    );
  });

  it('reports enabled and disabled features', async () => {
    mockGetMyFeatures.mockResolvedValue({
      data: { code: 0, data: RestrictedFeatures },
    });
    const { result } = renderHook(
      () => ({
        mcp: useIsAccessFeatureEnabled(AccessFeature.Mcp),
        team: useIsAccessFeatureEnabled(AccessFeature.Team),
      }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.mcp).toBe(false));
    expect(result.current.team).toBe(true);
  });

  it('allows background requests only after the switches load', async () => {
    let resolveFeatures: (value: unknown) => void = () => {};
    mockGetMyFeatures.mockReturnValue(
      new Promise((resolve) => {
        resolveFeatures = resolve;
      }),
    );
    const { result } = renderHook(
      () => ({
        team: useCanRequestAccessFeature(AccessFeature.Team),
        mcp: useCanRequestAccessFeature(AccessFeature.Mcp),
      }),
      { wrapper: createWrapper() },
    );

    expect(result.current).toEqual({ team: false, mcp: false });
    resolveFeatures({ data: { code: 0, data: RestrictedFeatures } });
    await waitFor(() => expect(result.current.team).toBe(true));
    expect(result.current.mcp).toBe(false);
  });

  it('blocks only pages owned by a disabled feature', async () => {
    mockGetMyFeatures.mockResolvedValue({
      data: { code: 0, data: RestrictedFeatures },
    });
    const { result } = renderHook(
      () => ({
        dataSource: useIsAccessFeaturePathBlocked(
          '/integrations/data-source/data-source-detail-page',
        ),
        model: useIsAccessFeaturePathBlocked('/integrations/model'),
        datasets: useIsAccessFeaturePathBlocked('/datasets'),
      }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.dataSource).toBe(true));
    expect(result.current.model).toBe(false);
    expect(result.current.datasets).toBe(false);
  });

  it('matches feature pages on path-segment boundaries', () => {
    expect(findAccessFeatureForPath('/team')).toBe(AccessFeature.Team);
    expect(findAccessFeatureForPath('/teams')).toBeUndefined();
    expect(findAccessFeatureForPath('/user-setting/api')).toBe(
      AccessFeature.Api,
    );
    expect(findAccessFeatureForPath('/integrations/mcp/edit')).toBe(
      AccessFeature.Mcp,
    );
  });
});
