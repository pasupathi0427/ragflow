import { fireEvent, render, screen } from '@testing-library/react';
import { GlobalModelAccessEditor } from '../global-model-access';

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

jest.mock('../../global-models/hooks', () => ({
  useFetchGlobalProviders: () => ({
    data: [
      {
        id: 'p1',
        provider_name: 'VLLM',
        instances: [
          {
            id: 'i1',
            instance_name: 'shared',
            models: [
              { id: 'm1', model_name: 'gpt-oss-20b', model_types: ['chat'] },
              { id: 'm2', model_name: 'qwen', model_types: ['chat'] },
            ],
          },
        ],
      },
    ],
    loading: false,
  }),
}));

describe('GlobalModelAccessEditor', () => {
  it('hides the model list while every global model is allowed', () => {
    render(
      <GlobalModelAccessEditor
        access={{ all: true, model_ids: [] }}
        onChange={jest.fn()}
      />,
    );
    expect(
      screen.getByRole('switch', { name: 'accessProfile.allGlobalModels' }),
    ).toBeChecked();
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
  });

  it('switches to selected models', () => {
    const onChange = jest.fn();
    render(
      <GlobalModelAccessEditor
        access={{ all: true, model_ids: [] }}
        onChange={onChange}
      />,
    );
    fireEvent.click(
      screen.getByRole('switch', { name: 'accessProfile.allGlobalModels' }),
    );
    expect(onChange).toHaveBeenCalledWith({ all: false, model_ids: [] });
  });

  it('adds and removes a model from the allow list', () => {
    const onChange = jest.fn();
    render(
      <GlobalModelAccessEditor
        access={{ all: false, model_ids: ['m1'] }}
        onChange={onChange}
      />,
    );
    const [first, second] = screen.getAllByRole('checkbox');
    expect(first).toBeChecked();
    expect(second).not.toBeChecked();

    fireEvent.click(second);
    expect(onChange).toHaveBeenLastCalledWith({
      all: false,
      model_ids: ['m1', 'm2'],
    });
    fireEvent.click(first);
    expect(onChange).toHaveBeenLastCalledWith({ all: false, model_ids: [] });
  });
});
