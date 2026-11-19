import { AccessFeature } from '@/constants/access-profile';
import { filterAccessProfiles, getDisabledFeatures } from '../utils';

const Profiles = [
  {
    id: 'default',
    name: 'Default',
    description: 'Applies to every user',
    features: {},
    is_default: true,
    member_count: 0,
    global_models: { all: true, model_ids: [] },
  },
  {
    id: 'p1',
    name: 'Interns',
    description: 'No API access',
    features: {},
    is_default: false,
    member_count: 3,
    global_models: { all: true, model_ids: [] },
  },
];

describe('filterAccessProfiles', () => {
  it('returns every profile for a blank keyword', () => {
    expect(filterAccessProfiles(Profiles, '   ')).toHaveLength(2);
  });

  it('matches name or description, ignoring case', () => {
    expect(filterAccessProfiles(Profiles, 'intern').map((p) => p.id)).toEqual([
      'p1',
    ]);
    expect(filterAccessProfiles(Profiles, 'EVERY').map((p) => p.id)).toEqual([
      'default',
    ]);
    expect(filterAccessProfiles(Profiles, 'nothing')).toEqual([]);
  });
});

describe('getDisabledFeatures', () => {
  it('lists switched-off features in display order', () => {
    expect(
      getDisabledFeatures({
        [AccessFeature.Api]: false,
        [AccessFeature.Mcp]: false,
        [AccessFeature.Team]: true,
      }),
    ).toEqual([AccessFeature.Mcp, AccessFeature.Api]);
  });

  it('treats missing switches as on', () => {
    expect(getDisabledFeatures({})).toEqual([]);
  });
});
