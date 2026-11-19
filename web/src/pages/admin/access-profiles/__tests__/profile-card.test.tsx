import { AccessFeature } from '@/constants/access-profile';
import { TooltipProvider } from '@/components/ui/tooltip';
import { fireEvent, render, screen } from '@testing-library/react';
import { ProfileCard } from '../profile-card';

jest.mock('../../global-models/hooks', () => ({
  useFetchGlobalProviders: () => ({ data: [], loading: false }),
}));

jest.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options ? `${key} ${JSON.stringify(options)}` : key,
  }),
}));

const Profile = {
  id: 'p1',
  name: 'Interns',
  description: 'Limited access',
  features: { [AccessFeature.Mcp]: false },
  is_default: false,
  member_count: 2,
  global_models: { all: true, model_ids: [] },
};

function renderCard(expanded: boolean, onExpandedChange = jest.fn()) {
  render(
    <TooltipProvider>
      <ProfileCard
        profile={Profile}
        saving={false}
        expanded={expanded}
        onExpandedChange={onExpandedChange}
        onFeatureChange={jest.fn()}
        onGlobalModelsChange={jest.fn()}
        onEdit={jest.fn()}
        onDelete={jest.fn()}
      />
    </TooltipProvider>,
  );
  return onExpandedChange;
}

describe('ProfileCard', () => {
  it('shows a summary and hides the switches when collapsed', () => {
    renderCard(false);

    expect(screen.getByText('Interns')).toBeInTheDocument();
    expect(
      screen.getByText(/accessProfile\.featuresOnCount/),
    ).toHaveTextContent('"on":5,"total":6');
    expect(screen.getByText(/accessProfile\.featuresOff/)).toHaveTextContent(
      'accessProfile.feature.mcp',
    );
    expect(screen.queryAllByRole('switch')).toHaveLength(0);
  });

  it('shows the switches when expanded', () => {
    renderCard(true);
    // Six feature switches plus the all-global-models switch.
    expect(screen.getAllByRole('switch')).toHaveLength(7);
  });

  it('asks to expand when the header is clicked', () => {
    const onExpandedChange = renderCard(false);

    fireEvent.click(screen.getByRole('button', { expanded: false }));

    expect(onExpandedChange).toHaveBeenCalledWith(Profile, true);
  });
});
