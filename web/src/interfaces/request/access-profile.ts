import { AccessFeatureSwitches } from '@/interfaces/database/access-profile';

export interface IAccessProfileRequestBody {
  name: string;
  description: string;
  features: AccessFeatureSwitches;
}
