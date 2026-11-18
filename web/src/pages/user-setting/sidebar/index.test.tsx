import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { SideBar } from './index';

// Access-profile switches come from the API; these tests see every feature on.
jest.mock('@/hooks/use-access-profile-request', () => ({
  useAccessFeatureFilter: (items: unknown[]) => items,
  useIsAccessFeatureEnabled: () => true,
  useIsAccessFeaturePathBlocked: () => false,
}));

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

jest.mock('./hooks', () => ({
  useHandleMenuClick: () => ({
    active: '/profile',
    handleMenuClick: jest.fn(),
  }),
}));

describe('settings SideBar', () => {
  it('renders only the settings navigation without the user header and footer', () => {
    const { container } = render(
      <MemoryRouter>
        <SideBar />
      </MemoryRouter>,
    );

    expect(
      screen.getByRole('button', { name: 'setting.profile' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'setting.api' }),
    ).toBeInTheDocument();
    expect(container.querySelector('header')).not.toBeInTheDocument();
    expect(container.querySelector('footer')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'setting.logout' }),
    ).not.toBeInTheDocument();
  });
});
