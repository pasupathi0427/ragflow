import { AccessFeature, AccessFeatures } from '@/constants/access-profile';
import { fireEvent, render, screen } from '@testing-library/react';
import { FeatureSwitchList } from '../feature-switch-list';

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

describe('FeatureSwitchList', () => {
  it('shows one switch per feature, missing switches counting as on', () => {
    render(
      <FeatureSwitchList
        features={{ [AccessFeature.Mcp]: false }}
        onFeatureChange={jest.fn()}
      />,
    );

    const switches = screen.getAllByRole('switch');
    expect(switches).toHaveLength(AccessFeatures.length);
    expect(
      screen.getByRole('switch', { name: 'accessProfile.feature.mcp' }),
    ).not.toBeChecked();
    expect(
      screen.getByRole('switch', { name: 'accessProfile.feature.team' }),
    ).toBeChecked();
  });

  it('warns that existing API keys stop working when API is off', () => {
    const { rerender } = render(
      <FeatureSwitchList features={{}} onFeatureChange={jest.fn()} />,
    );
    expect(
      screen.queryByText('accessProfile.apiKeysWarning'),
    ).not.toBeInTheDocument();

    rerender(
      <FeatureSwitchList
        features={{ [AccessFeature.Api]: false }}
        onFeatureChange={jest.fn()}
      />,
    );
    expect(
      screen.getByText('accessProfile.apiKeysWarning'),
    ).toBeInTheDocument();
  });

  it('reports the toggled feature', () => {
    const onFeatureChange = jest.fn();
    render(
      <FeatureSwitchList features={{}} onFeatureChange={onFeatureChange} />,
    );

    fireEvent.click(
      screen.getByRole('switch', { name: 'accessProfile.feature.team' }),
    );

    expect(onFeatureChange).toHaveBeenCalledWith(AccessFeature.Team, false);
  });
});
