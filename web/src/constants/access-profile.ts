import { Routes } from '@/routes';

// Feature keys controlled by an access profile. Values match the Go backend
// (internal/accessprofile/feature.go).
export enum AccessFeature {
  ModelProviders = 'model_providers',
  DataSources = 'data_sources',
  ChatChannels = 'chat_channels',
  Mcp = 'mcp',
  Team = 'team',
  Api = 'api',
}

export const AccessFeatures: AccessFeature[] = [
  AccessFeature.ModelProviders,
  AccessFeature.DataSources,
  AccessFeature.ChatChannels,
  AccessFeature.Mcp,
  AccessFeature.Team,
  AccessFeature.Api,
];

export const DefaultAccessProfileId = 'default';

// Pages owned by a feature. A page is matched when the pathname equals the path
// or starts with it followed by "/".
export const AccessFeatureRoutes: { path: string; feature: AccessFeature }[] = [
  {
    path: `${Routes.Integrations}${Routes.Model}`,
    feature: AccessFeature.ModelProviders,
  },
  {
    path: `${Routes.Integrations}${Routes.DataSource}`,
    feature: AccessFeature.DataSources,
  },
  {
    path: `${Routes.Integrations}${Routes.ChatChannel}`,
    feature: AccessFeature.ChatChannels,
  },
  {
    path: `${Routes.Integrations}${Routes.Mcp}`,
    feature: AccessFeature.Mcp,
  },
  { path: Routes.Team, feature: AccessFeature.Team },
  { path: `${Routes.UserSetting}${Routes.Api}`, feature: AccessFeature.Api },
];

// Integrations sidebar children (layouts/components/app-sidebar/navigation.ts)
// keyed by their `key`.
export const IntegrationAccessFeatures: Record<string, AccessFeature> = {
  model: AccessFeature.ModelProviders,
  'data-source': AccessFeature.DataSources,
  'chat-channel': AccessFeature.ChatChannels,
  mcp: AccessFeature.Mcp,
};
