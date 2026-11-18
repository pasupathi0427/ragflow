import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router';
import IntegrationsIndexRedirect from './integrations-index-redirect';

let mockFetched = true;
let mockDisabledKeys: string[] = [];

jest.mock('@/hooks/use-access-profile-request', () => ({
  useFetchMyAccessFeatures: () => ({ fetched: mockFetched }),
  useAccessFeatureFilter: (
    items: { key: string }[],
    getFeature: (item: { key: string }) => string | undefined,
  ) => items.filter((item) => !mockDisabledKeys.includes(getFeature(item)!)),
}));

function CurrentPath() {
  return <div>{useLocation().pathname}</div>;
}

function renderRedirect() {
  return render(
    <MemoryRouter initialEntries={['/integrations']}>
      <Routes>
        <Route path="/integrations" element={<IntegrationsIndexRedirect />} />
        <Route path="*" element={<CurrentPath />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('IntegrationsIndexRedirect', () => {
  beforeEach(() => {
    mockFetched = true;
    mockDisabledKeys = [];
  });

  it('waits for the access-profile switches', () => {
    mockFetched = false;
    const { container } = renderRedirect();
    expect(container).toBeEmptyDOMElement();
  });

  it('opens model providers when it is enabled', () => {
    renderRedirect();
    expect(screen.getByText('/integrations/model')).toBeInTheDocument();
  });

  it('skips disabled integrations', () => {
    mockDisabledKeys = ['model_providers', 'data_sources'];
    renderRedirect();
    expect(screen.getByText('/integrations/chat-channel')).toBeInTheDocument();
  });

  it('falls back to home when every integration is disabled', () => {
    mockDisabledKeys = [
      'model_providers',
      'data_sources',
      'chat_channels',
      'mcp',
    ];
    renderRedirect();
    expect(screen.getByText('/')).toBeInTheDocument();
  });
});
