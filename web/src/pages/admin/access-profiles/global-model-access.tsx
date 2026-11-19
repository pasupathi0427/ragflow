import { Checkbox } from '@/components/ui/checkbox';
import { Switch } from '@/components/ui/switch';
import { IGlobalModelAccess } from '@/interfaces/database/access-profile';
import { useCallback, useId, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useFetchGlobalProviders } from '../global-models/hooks';

type GlobalModelOption = { id: string; label: string };

function GlobalModelCheckbox({
  option,
  checked,
  disabled,
  onToggle,
}: {
  option: GlobalModelOption;
  checked: boolean;
  disabled?: boolean;
  onToggle: (modelId: string, checked: boolean) => void;
}) {
  const id = useId();
  const handleChange = useCallback(
    (value: boolean | 'indeterminate') => onToggle(option.id, value === true),
    [onToggle, option.id],
  );
  return (
    <label htmlFor={id} className="flex min-w-0 items-center gap-2 text-sm">
      <Checkbox
        id={id}
        checked={checked}
        disabled={disabled}
        onCheckedChange={handleChange}
      />
      <span className="truncate" title={option.label}>
        {option.label}
      </span>
    </label>
  );
}

/**
 * Chooses which global models a profile's users may call: every global model,
 * or only the checked ones. A model left out is refused on every call.
 */
export function GlobalModelAccessEditor({
  access,
  disabled,
  onChange,
}: {
  access: IGlobalModelAccess;
  disabled?: boolean;
  onChange: (access: IGlobalModelAccess) => void;
}) {
  const { t } = useTranslation();
  const labelId = useId();
  const { data: providers } = useFetchGlobalProviders();

  const options = useMemo(
    () =>
      providers.flatMap((provider) =>
        provider.instances.flatMap((instance) =>
          instance.models.map((model) => ({
            id: model.id,
            label: `${model.model_name} (${provider.provider_name} / ${instance.instance_name})`,
          })),
        ),
      ),
    [providers],
  );

  const handleAllChange = useCallback(
    (all: boolean) => onChange({ all, model_ids: all ? [] : access.model_ids }),
    [access.model_ids, onChange],
  );
  const handleToggle = useCallback(
    (modelId: string, checked: boolean) =>
      onChange({
        all: false,
        model_ids: checked
          ? [...access.model_ids.filter((id) => id !== modelId), modelId]
          : access.model_ids.filter((id) => id !== modelId),
      }),
    [access.model_ids, onChange],
  );

  return (
    <div className="space-y-3 rounded-lg bg-bg-card p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-1">
          <div id={labelId} className="text-sm font-medium">
            {t('accessProfile.allGlobalModels')}
          </div>
          <p className="text-xs text-text-secondary">
            {t('accessProfile.globalModelsDescription')}
          </p>
        </div>
        <Switch
          aria-labelledby={labelId}
          checked={access.all}
          disabled={disabled}
          onCheckedChange={handleAllChange}
        />
      </div>
      {!access.all &&
        (options.length ? (
          <div className="grid gap-2 md:grid-cols-2">
            {options.map((option) => (
              <GlobalModelCheckbox
                key={option.id}
                option={option}
                checked={access.model_ids.includes(option.id)}
                disabled={disabled}
                onToggle={handleToggle}
              />
            ))}
          </div>
        ) : (
          <p className="text-xs text-text-secondary">
            {t('accessProfile.noGlobalModels')}
          </p>
        ))}
    </div>
  );
}
