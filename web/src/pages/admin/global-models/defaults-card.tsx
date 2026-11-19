import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import message from '@/components/ui/message';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  GlobalModelDefaults,
  GlobalModelStatus,
  IGlobalProvider,
} from '@/interfaces/database/global-model';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { GlobalModelTypeKey, GlobalModelTypes } from './constants';
import { useFetchGlobalDefaults, useSetGlobalDefaults } from './hooks';

// Select needs a non-empty value for the "no default" choice.
const NoDefault = '__none__';

type ModelOption = { id: string; label: string };

function DefaultSelect({
  type,
  value,
  options,
  onChange,
}: {
  type: GlobalModelTypeKey;
  value: string;
  options: ModelOption[];
  onChange: (type: GlobalModelTypeKey, value: string) => void;
}) {
  const { t } = useTranslation();
  const handleChange = useCallback(
    (next: string) => onChange(type, next === NoDefault ? '' : next),
    [onChange, type],
  );
  return (
    <label className="space-y-1.5 text-sm">
      <span>{t(`globalModel.type.${type}`)}</span>
      <Select value={value || NoDefault} onValueChange={handleChange}>
        <SelectTrigger aria-label={t(`globalModel.type.${type}`)}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={NoDefault}>
            {t('globalModel.noDefault')}
          </SelectItem>
          {options.map((option) => (
            <SelectItem key={option.id} value={option.id}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </label>
  );
}

/** Global default model per type, used by users who have none of their own. */
export function DefaultsCard({ providers }: { providers: IGlobalProvider[] }) {
  const { t } = useTranslation();
  const { data: saved } = useFetchGlobalDefaults();
  const { mutate: saveDefaults, loading } = useSetGlobalDefaults();
  // Unsaved changes over the saved defaults.
  const [changes, setChanges] = useState<GlobalModelDefaults>({});
  const draft = useMemo(() => ({ ...saved, ...changes }), [saved, changes]);

  // Only enabled models of enabled instances can be defaults.
  const optionsByType = useMemo(() => {
    const options: Record<string, ModelOption[]> = {};
    for (const provider of providers) {
      for (const instance of provider.instances) {
        if (instance.status !== GlobalModelStatus.Active) continue;
        for (const model of instance.models) {
          if (model.status !== GlobalModelStatus.Active) continue;
          for (const type of model.model_types) {
            (options[type] ??= []).push({
              id: model.id,
              label: `${model.model_name} (${provider.provider_name} / ${instance.instance_name})`,
            });
          }
        }
      }
    }
    return options;
  }, [providers]);

  const handleChange = useCallback(
    (type: GlobalModelTypeKey, value: string) =>
      setChanges((previous) => ({ ...previous, [type]: value })),
    [],
  );

  const handleSave = useCallback(async () => {
    const body: GlobalModelDefaults = {};
    for (const { storedKey, requestKey } of GlobalModelTypes) {
      if (draft[storedKey]) {
        body[requestKey] = draft[storedKey];
      }
    }
    try {
      await saveDefaults(body);
    } catch {
      // The admin request client already shows the error.
      return;
    }
    setChanges({});
    message.success(t('globalModel.defaultsSaved'));
  }, [draft, saveDefaults, t]);

  return (
    <Card className="border-0.5 border-border-default bg-transparent">
      <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
        <div className="space-y-1.5">
          <CardTitle className="text-lg font-normal">
            {t('globalModel.defaultsTitle')}
          </CardTitle>
          <p className="text-sm text-text-secondary">
            {t('globalModel.defaultsDescription')}
          </p>
        </div>
        <Button className="shrink-0" loading={loading} onClick={handleSave}>
          {t('globalModel.saveDefaults')}
        </Button>
      </CardHeader>
      <CardContent className="grid gap-4 p-6 pt-0 md:grid-cols-2">
        {GlobalModelTypes.map(({ storedKey }) => (
          <DefaultSelect
            key={storedKey}
            type={storedKey}
            value={draft[storedKey] ?? ''}
            options={optionsByType[storedKey] ?? []}
            onChange={handleChange}
          />
        ))}
      </CardContent>
    </Card>
  );
}
