import { TooltipProvider } from '@/components/ui/tooltip';
import { fireEvent, render, screen } from '@testing-library/react';
import AdminAccessProfiles from '../index';

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

jest.mock('@/components/spotlight', () => ({
  __esModule: true,
  default: () => null,
}));

jest.mock('@/components/empty/empty', () => ({
  __esModule: true,
  default: ({ text }: { text?: string }) => <div>{text ?? 'empty'}</div>,
}));

jest.mock('@/components/ui/message', () => ({
  __esModule: true,
  default: { success: jest.fn() },
}));

jest.mock('../hooks', () => ({
  useFetchAccessProfiles: () => ({
    data: [
      {
        id: 'default',
        name: 'Default',
        description: '',
        features: {},
        is_default: true,
        member_count: 0,
      },
      {
        id: 'p1',
        name: 'Interns',
        description: 'Limited access',
        features: { mcp: false },
        is_default: false,
        member_count: 2,
      },
    ],
    loading: false,
  }),
  useCreateAccessProfile: () => ({ createAccessProfile: jest.fn() }),
  useUpdateAccessProfile: () => ({ updateAccessProfile: jest.fn() }),
  useDeleteAccessProfile: () => ({ deleteAccessProfile: jest.fn() }),
}));

function renderPage() {
  return render(
    <TooltipProvider>
      <AdminAccessProfiles />
    </TooltipProvider>,
  );
}

describe('AdminAccessProfiles', () => {
  it('starts with every profile collapsed', () => {
    renderPage();

    expect(screen.getByText('Default')).toBeInTheDocument();
    expect(screen.getByText('Interns')).toBeInTheDocument();
    expect(screen.queryAllByRole('switch')).toHaveLength(0);
  });

  it('filters profiles by the search keyword', () => {
    renderPage();

    fireEvent.change(
      screen.getByRole('textbox', { name: 'accessProfile.searchPlaceholder' }),
      { target: { value: 'intern' } },
    );

    expect(screen.queryByText('Default')).not.toBeInTheDocument();
    expect(screen.getByText('Interns')).toBeInTheDocument();
  });

  it('shows a no-match notice when nothing matches', () => {
    renderPage();

    fireEvent.change(
      screen.getByRole('textbox', { name: 'accessProfile.searchPlaceholder' }),
      { target: { value: 'zzz' } },
    );

    expect(screen.getByText('accessProfile.noMatches')).toBeInTheDocument();
  });

  it('expands and collapses all visible profiles', () => {
    renderPage();

    fireEvent.click(screen.getByText('accessProfile.expandAll'));
    expect(screen.getAllByRole('switch')).toHaveLength(12);

    fireEvent.click(screen.getByText('accessProfile.collapseAll'));
    expect(screen.queryAllByRole('switch')).toHaveLength(0);
  });
});
