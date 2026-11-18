import api from '@/utils/api';
import { registerNextServer } from '@/utils/register-server';

const methods = {
  getMyFeatures: {
    url: api.myAccessFeatures,
    method: 'get',
  },
} as const;

const accessProfileService = registerNextServer<keyof typeof methods>(methods);

export default accessProfileService;
