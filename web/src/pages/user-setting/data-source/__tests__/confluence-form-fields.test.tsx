import { DynamicForm } from '@/components/dynamic-form';
import { TooltipProvider } from '@/components/ui/tooltip';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { confluenceConstant } from '../constant/confluence-constant';

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

const translate = ((key: string) => key) as never;

// jsdom has no ResizeObserver; the Segmented control observes its size.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
Object.assign(globalThis, {
  ResizeObserver: globalThis.ResizeObserver ?? ResizeObserverStub,
});

function renderConfluenceForm() {
  return render(
    <TooltipProvider>
      <DynamicForm.Root
        fields={confluenceConstant(translate)}
        onSubmit={jest.fn()}
        defaultValues={{
          config: { index_mode: 'space', space: 'AITestingS', page_id: '' },
        }}
      />
    </TooltipProvider>,
  );
}

// The Confluence config declares config.space, config.page_id and
// config.index_recursively twice (a visible field and a hidden copy).
describe('Confluence data source form', () => {
  it('renders one Space Key field however often the mode changes', () => {
    const consoleError = jest
      .spyOn(console, 'error')
      .mockImplementation(() => {});
    try {
      renderConfluenceForm();
      for (const mode of [
        'setting.dataSourceOptionPage',
        'setting.dataSourceOptionSpace',
        'setting.dataSourceOptionEverything',
        'setting.dataSourceOptionSpace',
        'setting.dataSourceOptionPage',
        'setting.dataSourceOptionSpace',
      ]) {
        act(() => {
          fireEvent.click(screen.getByText(mode));
        });
      }

      const duplicateKeyWarnings = consoleError.mock.calls.filter((call) =>
        String(call[0]).includes('same key'),
      );
      expect(duplicateKeyWarnings).toEqual([]);

      const visibleSpaceInputs = screen
        .getAllByDisplayValue('AITestingS')
        .filter((input) => !input.closest('.hidden'));
      expect(visibleSpaceInputs).toHaveLength(1);
    } finally {
      consoleError.mockRestore();
    }
  });
});
