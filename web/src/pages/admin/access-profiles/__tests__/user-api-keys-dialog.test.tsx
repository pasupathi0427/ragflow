import { TooltipProvider } from '@/components/ui/tooltip';
import {
  createUserAPIKey,
  deleteUserAPIKey,
  listUserAPIKeys,
  setUserAPIKeyEnabled,
} from '@/services/admin-service';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { UserAPIKeysDialog } from '../user-api-keys-dialog';

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

jest.mock('@/hooks/common-hooks', () => ({
  useTranslate: () => ({ t: (key: string) => key }),
}));

jest.mock('@/components/ui/message', () => ({
  __esModule: true,
  default: { success: jest.fn() },
}));

jest.mock('@/services/admin-service', () => ({
  listUserAPIKeys: jest.fn(),
  createUserAPIKey: jest.fn(),
  setUserAPIKeyEnabled: jest.fn(),
  deleteUserAPIKey: jest.fn(),
}));

const ok = <T,>(data: T) => ({ data: { code: 0, message: '', data } });

const existingKey = {
  id: 'k1',
  name: 'Open WebUI',
  masked_token: 'ragflow-abcd…wxyz',
  enabled: true,
  created_by: 'admin@example.com',
  create_time: 1763700000000,
};

function renderDialog() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <TooltipProvider>
        <UserAPIKeysDialog
          email="a@example.com"
          open
          onOpenChange={jest.fn()}
        />
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

describe('UserAPIKeysDialog', () => {
  beforeEach(() => {
    jest.mocked(listUserAPIKeys).mockResolvedValue(ok([existingKey]) as never);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('lists keys by their masked value only', async () => {
    renderDialog();

    expect(await screen.findByText('Open WebUI')).toBeInTheDocument();
    expect(screen.getByText('ragflow-abcd…wxyz')).toBeInTheDocument();
    expect(listUserAPIKeys).toHaveBeenCalledWith('a@example.com');
  });

  it('shows a new key in full once and hides the form until done', async () => {
    jest.mocked(createUserAPIKey).mockResolvedValue(
      ok({
        ...existingKey,
        id: 'k2',
        name: 'CI',
        token: 'ragflow-full-secret',
      }) as never,
    );
    renderDialog();

    fireEvent.change(
      screen.getByRole('textbox', { name: 'accessProfile.apiKeys.name' }),
      { target: { value: '  CI  ' } },
    );
    fireEvent.click(
      screen.getByRole('button', { name: 'accessProfile.apiKeys.create' }),
    );

    expect(await screen.findByText('ragflow-full-secret')).toBeInTheDocument();
    expect(createUserAPIKey).toHaveBeenCalledWith('a@example.com', 'CI');
    expect(
      screen.queryByRole('textbox', { name: 'accessProfile.apiKeys.name' }),
    ).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: 'accessProfile.apiKeys.done' }),
    );
    expect(screen.queryByText('ragflow-full-secret')).not.toBeInTheDocument();
  });

  it('switches a key off', async () => {
    jest
      .mocked(setUserAPIKeyEnabled)
      .mockResolvedValue(ok({ ...existingKey, enabled: false }) as never);
    renderDialog();

    fireEvent.click(
      await screen.findByRole('switch', {
        name: 'accessProfile.apiKeys.enabledToggle',
      }),
    );

    await waitFor(() =>
      expect(setUserAPIKeyEnabled).toHaveBeenCalledWith(
        'a@example.com',
        'k1',
        false,
      ),
    );
  });

  it('deletes a key after confirmation', async () => {
    jest.mocked(deleteUserAPIKey).mockResolvedValue(ok(null) as never);
    renderDialog();

    fireEvent.click(
      await screen.findByRole('button', {
        name: 'accessProfile.apiKeys.delete',
      }),
    );
    expect(deleteUserAPIKey).not.toHaveBeenCalled();
    fireEvent.click(await screen.findByRole('button', { name: /delete/i }));

    await waitFor(() =>
      expect(deleteUserAPIKey).toHaveBeenCalledWith('a@example.com', 'k1'),
    );
  });
});
