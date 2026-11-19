import { TruncatedText } from '@/components/truncated-text';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import { AccessFeature, AccessFeatures } from '@/constants/access-profile';
import {
  IAccessProfile,
  IGlobalModelAccess,
} from '@/interfaces/database/access-profile';
import { cn } from '@/lib/utils';
import { LucideChevronRight, LucideEdit3, LucideTrash2 } from 'lucide-react';
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { FeatureSwitchList } from './feature-switch-list';
import { GlobalModelAccessEditor } from './global-model-access';
import { getDisabledFeatures } from './utils';

const DefaultGlobalModelAccess: IGlobalModelAccess = {
  all: true,
  model_ids: [],
};

export function ProfileCard({
  profile,
  saving,
  expanded,
  onExpandedChange,
  onFeatureChange,
  onGlobalModelsChange,
  onEdit,
  onDelete,
}: {
  profile: IAccessProfile;
  saving: boolean;
  expanded: boolean;
  onExpandedChange: (profile: IAccessProfile, expanded: boolean) => void;
  onFeatureChange: (
    profile: IAccessProfile,
    feature: AccessFeature,
    enabled: boolean,
  ) => void;
  onGlobalModelsChange: (
    profile: IAccessProfile,
    access: IGlobalModelAccess,
  ) => void;
  onEdit: (profile: IAccessProfile) => void;
  onDelete: (profile: IAccessProfile) => void;
}) {
  const { t } = useTranslation();

  const handleOpenChange = useCallback(
    (open: boolean) => onExpandedChange(profile, open),
    [onExpandedChange, profile],
  );
  const handleFeatureChange = useCallback(
    (feature: AccessFeature, enabled: boolean) =>
      onFeatureChange(profile, feature, enabled),
    [onFeatureChange, profile],
  );
  const handleGlobalModelsChange = useCallback(
    (access: IGlobalModelAccess) => onGlobalModelsChange(profile, access),
    [onGlobalModelsChange, profile],
  );
  const handleEdit = useCallback(() => onEdit(profile), [onEdit, profile]);
  const handleDelete = useCallback(
    () => onDelete(profile),
    [onDelete, profile],
  );

  const featureSummary = useMemo(() => {
    const disabled = getDisabledFeatures(profile.features);
    if (disabled.length === 0) {
      return t('accessProfile.allFeaturesOn');
    }
    const enabledCount = t('accessProfile.featuresOnCount', {
      on: AccessFeatures.length - disabled.length,
      total: AccessFeatures.length,
    });
    const disabledNames = disabled
      .map((feature) => t(`accessProfile.feature.${feature}`))
      .join(', ');
    return `${enabledCount} · ${t('accessProfile.featuresOff', { features: disabledNames })}`;
  }, [profile.features, t]);

  const description =
    profile.is_default && !profile.description
      ? t('accessProfile.defaultProfileHint')
      : profile.description;

  return (
    <Collapsible open={expanded} onOpenChange={handleOpenChange}>
      <Card className="border-0.5 border-border-default bg-transparent">
        <CardHeader
          className={cn(
            'flex flex-row items-start gap-4 space-y-0',
            expanded && 'border-b-0.5 border-border-button',
          )}
        >
          <CollapsibleTrigger asChild>
            <Button
              variant="transparent"
              className="h-auto min-w-0 flex-1 items-start justify-start gap-3 whitespace-normal border-0 p-0 text-left hover:bg-transparent"
            >
              <LucideChevronRight
                className={cn(
                  'mt-1.5 size-4 shrink-0 text-text-secondary transition-transform',
                  expanded && 'rotate-90',
                )}
              />
              <div className="min-w-0 flex-1 space-y-1.5">
                <div className="flex min-w-0 items-center gap-2">
                  <TruncatedText
                    className="min-w-0 truncate text-xl font-normal text-text-primary"
                    tooltip={profile.name}
                  >
                    {profile.name}
                  </TruncatedText>
                  {profile.is_default && (
                    <Badge variant="secondary">
                      {t('accessProfile.defaultBadge')}
                    </Badge>
                  )}
                </div>
                {description && (
                  <p className="line-clamp-2 break-words text-sm font-normal text-text-secondary">
                    {description}
                  </p>
                )}
                <p className="break-words text-xs font-normal text-text-secondary">
                  {!profile.is_default &&
                    `${t('accessProfile.memberCount', { count: profile.member_count })} · `}
                  {featureSummary}
                </p>
              </div>
            </Button>
          </CollapsibleTrigger>

          <Button
            variant="ghost"
            size="icon"
            aria-label={t('accessProfile.editProfile')}
            onClick={handleEdit}
          >
            <LucideEdit3 />
          </Button>
          {!profile.is_default && (
            <Button
              variant="ghost"
              size="icon"
              aria-label={t('accessProfile.deleteProfile')}
              onClick={handleDelete}
            >
              <LucideTrash2 />
            </Button>
          )}
        </CardHeader>

        <CollapsibleContent>
          <CardContent className="space-y-4 p-6">
            <FeatureSwitchList
              features={profile.features}
              disabled={saving}
              onFeatureChange={handleFeatureChange}
            />
            <GlobalModelAccessEditor
              access={profile.global_models ?? DefaultGlobalModelAccess}
              disabled={saving}
              onChange={handleGlobalModelsChange}
            />
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}
