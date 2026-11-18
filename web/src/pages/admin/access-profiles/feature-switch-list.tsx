import { Switch } from '@/components/ui/switch';
import { AccessFeature, AccessFeatures } from '@/constants/access-profile';
import { AccessFeatureSwitches } from '@/interfaces/database/access-profile';
import { useCallback, useId } from 'react';
import { useTranslation } from 'react-i18next';

type FeatureChangeHandler = (feature: AccessFeature, enabled: boolean) => void;

function FeatureSwitchRow({
  feature,
  enabled,
  disabled,
  onFeatureChange,
}: {
  feature: AccessFeature;
  enabled: boolean;
  disabled?: boolean;
  onFeatureChange: FeatureChangeHandler;
}) {
  const { t } = useTranslation();
  const labelId = useId();
  const descriptionId = useId();

  const handleCheckedChange = useCallback(
    (checked: boolean) => onFeatureChange(feature, checked),
    [feature, onFeatureChange],
  );

  return (
    <div className="flex items-start justify-between gap-4 rounded-lg bg-bg-card p-4">
      <div className="min-w-0 space-y-1">
        {/* The shared Switch owns its id, so label it by reference. */}
        <div id={labelId} className="text-sm font-medium">
          {t(`accessProfile.feature.${feature}`)}
        </div>
        <p id={descriptionId} className="text-xs text-text-secondary">
          {t(`accessProfile.featureDescription.${feature}`)}
        </p>
        {feature === AccessFeature.Api && !enabled && (
          <p className="text-xs text-state-warning">
            {t('accessProfile.apiKeysWarning')}
          </p>
        )}
      </div>
      <Switch
        aria-labelledby={labelId}
        aria-describedby={descriptionId}
        checked={enabled}
        disabled={disabled}
        onCheckedChange={handleCheckedChange}
      />
    </div>
  );
}

export function FeatureSwitchList({
  features,
  disabled,
  onFeatureChange,
}: {
  features: AccessFeatureSwitches;
  disabled?: boolean;
  onFeatureChange: FeatureChangeHandler;
}) {
  return (
    <div className="grid gap-3 md:grid-cols-2">
      {AccessFeatures.map((feature) => (
        <FeatureSwitchRow
          key={feature}
          feature={feature}
          enabled={features[feature] !== false}
          disabled={disabled}
          onFeatureChange={onFeatureChange}
        />
      ))}
    </div>
  );
}
