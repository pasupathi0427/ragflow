import {
  AccessFeatureSwitches,
  IGlobalModelAccess,
} from '@/interfaces/database/access-profile';

export interface IAccessProfileRequestBody {
  name: string;
  description: string;
  features: AccessFeatureSwitches;
  /** Omit to keep the profile's current global model access. */
  global_models?: IGlobalModelAccess;
}
