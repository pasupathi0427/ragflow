import { AccessFeatureRouteGuard } from '@/components/access-feature-route-guard';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';

const mockIsBlocked = jest.fn();

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

jest.mock('@/components/empty/empty', () => ({
  __esModule: true,
  default: ({ text }: { text: string }) => <div>{text}</div>,
}));

jest.mock('@/hooks/use-access-profile-request', () => ({
  useIsAccessFeaturePathBlocked: (pathname: string) => mockIsBlocked(pathname),
}));

function renderAt(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <AccessFeatureRouteGuard>
        <div>page content</div>
      </AccessFeatureRouteGuard>
    </MemoryRouter>,
  );
}

describe('AccessFeatureRouteGuard', () => {
  it('replaces a page whose feature is disabled', () => {
    mockIsBlocked.mockReturnValue(true);
    renderAt('/integrations/mcp');

    expect(mockIsBlocked).toHaveBeenCalledWith('/integrations/mcp');
    expect(
      screen.getByText('accessProfile.featureDisabled'),
    ).toBeInTheDocument();
    expect(screen.queryByText('page content')).not.toBeInTheDocument();
  });

  it('renders the page when its feature is enabled', () => {
    mockIsBlocked.mockReturnValue(false);
    renderAt('/integrations/model');

    expect(screen.getByText('page content')).toBeInTheDocument();
  });
});
