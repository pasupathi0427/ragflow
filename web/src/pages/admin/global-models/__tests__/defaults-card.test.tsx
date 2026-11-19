import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { DefaultsCard } from '../defaults-card';

const mockSaveDefaults = jest.fn();

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

jest.mock('@/components/ui/message', () => ({
  __esModule: true,
  default: { success: jest.fn() },
}));

jest.mock('../hooks', () => ({
  useFetchGlobalDefaults: () => ({
    data: { chat: 'chat-model', speech2text: 'asr-model' },
    loading: false,
  }),
  useSetGlobalDefaults: () => ({ mutate: mockSaveDefaults, loading: false }),
}));

const Providers = [
  {
    id: 'p1',
    provider_name: 'VLLM',
    instances: [
      {
        id: 'i1',
        instance_name: 'shared',
        status: 'active',
        base_url: '',
        region: '',
        has_api_key: false,
        api_key_masked: '',
        models: [
          {
            id: 'chat-model',
            model_name: 'gpt-oss-20b',
            model_types: ['chat'],
            status: 'active',
          },
          {
            id: 'asr-model',
            model_name: 'whisper',
            model_types: ['speech2text'],
            status: 'active',
          },
        ],
      },
    ],
  },
];

describe('DefaultsCard', () => {
  it('saves defaults under request keys that survive snake-casing', async () => {
    mockSaveDefaults.mockResolvedValue({});
    render(<DefaultsCard providers={Providers} />);

    fireEvent.click(screen.getByText('globalModel.saveDefaults'));

    await waitFor(() =>
      expect(mockSaveDefaults).toHaveBeenCalledWith({
        chat: 'chat-model',
        asr: 'asr-model',
      }),
    );
  });
});
