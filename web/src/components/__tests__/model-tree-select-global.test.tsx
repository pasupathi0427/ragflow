import { buildModelTree } from '@/components/model-tree-select';

jest.mock('@/locales/config', () => ({
  __esModule: true,
  default: { t: (key: string) => key },
}));

const OwnModel = {
  model_id: 'own',
  model_type: ['chat'],
  name: 'qwen',
  provider_id: 'p',
  provider_name: 'VLLM',
  instance_id: 'i',
  instance_name: 'default',
};

const GlobalModel = {
  ...OwnModel,
  model_id: 'global',
  provider_id: 'gp',
  instance_id: 'gi',
  tenant_id: 'global',
  is_global: true,
};

describe('buildModelTree with global models', () => {
  it('keeps global models in their own provider group', () => {
    const tree = buildModelTree([OwnModel, GlobalModel], ['chat']);

    expect(tree.map((node) => node.title)).toEqual([
      'VLLM',
      'VLLM (globalModel.badge)',
    ]);
    const leafIds = tree.map((node) => node.children?.[0]?.children?.[0]?.id);
    expect(leafIds).toEqual(['own', 'global']);
  });
});
