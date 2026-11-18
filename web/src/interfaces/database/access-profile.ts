import { AccessFeature } from '@/constants/access-profile';

export type AccessFeatureSwitches = Partial<Record<AccessFeature, boolean>>;

export interface IMyAccessFeatures {
  profile_id: string;
  profile_name: string;
  features: AccessFeatureSwitches;
}

export interface IAccessProfile {
  id: string;
  name: string;
  description: string;
  features: AccessFeatureSwitches;
  is_default: boolean;
  member_count: number;
}

export interface IAccessProfileAssignment {
  user_id: string;
  email: string;
  profile_id: string;
}
