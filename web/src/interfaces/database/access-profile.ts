import { AccessFeature } from '@/constants/access-profile';

export type AccessFeatureSwitches = Partial<Record<AccessFeature, boolean>>;

export interface IMyAccessFeatures {
  profile_id: string;
  profile_name: string;
  features: AccessFeatureSwitches;
}

/** Which global models a profile may call: all of them, or the listed IDs. */
export interface IGlobalModelAccess {
  all: boolean;
  model_ids: string[];
}

export interface IAccessProfile {
  id: string;
  name: string;
  description: string;
  features: AccessFeatureSwitches;
  global_models: IGlobalModelAccess;
  is_default: boolean;
  member_count: number;
}

/** A key the super admin manages for a user. `token` is set only right after
 * the key is created; the list returns `masked_token` only. */
export interface IAdminAPIKey {
  id: string;
  name: string;
  masked_token: string;
  token?: string;
  enabled: boolean;
  created_by: string;
  create_time?: number;
}

export interface IAccessProfileAssignment {
  user_id: string;
  email: string;
  profile_id: string;
}
