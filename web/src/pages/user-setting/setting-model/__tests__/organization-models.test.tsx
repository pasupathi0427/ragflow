import { TooltipProvider } from '@/components/ui/tooltip';
import { render, screen } from '@testing-library/react';
import { OrganizationModels } from '../organization-models';

let mockModels: Record<string, unknown>[] = [];

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

jest.mock('@/locales/config', () => ({
  __esModule: true,
  default: { t: (key: string) => key },
}));

jest.mock('@/components/svg-icon', () => ({
  LlmIcon: () => null,
}));

jest.mock('@/hooks/use-llm-request', () => ({
  useFetchAllAddedModels: () => ({ data: mockModels }),
}));

function renderList() {
  return render(
    <TooltipProvider>
      <OrganizationModels />
    </TooltipProvider>,
  );
}

describe('OrganizationModels', () => {
  it('lists only global models, read-only', () => {
    mockModels = [
      {
        model_id: 'own',
        name: 'my-model',
        provider_name: 'OpenAI',
        instance_name: 'default',
        model_type: ['chat'],
      },
      {
        model_id: 'global',
        name: 'gpt-oss-20b',
        provider_name: 'VLLM',
        instance_name: 'shared',
        model_type: ['chat'],
        is_global: true,
      },
    ];
    renderList();

    expect(screen.getByText('gpt-oss-20b')).toBeInTheDocument();
    expect(screen.queryByText('my-model')).not.toBeInTheDocument();
    expect(screen.getByText('globalModel.badge')).toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('renders nothing when the organization provides no models', () => {
    mockModels = [];
    const { container } = renderList();
    expect(container).toBeEmptyDOMElement();
  });
});
