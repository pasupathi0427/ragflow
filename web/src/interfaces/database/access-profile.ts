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

export interface IAccessProfileAssignment {
  user_id: string;
  email: string;
  profile_id: string;
}
