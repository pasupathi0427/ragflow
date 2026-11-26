import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import message from '@/components/ui/message';
import { IGlobalProvider } from '@/interfaces/database/global-model';
import { IGlobalModelInput } from '@/interfaces/request/global-model';
import { LucidePlus, LucideTrash2 } from 'lucide-react';
import {
  ChangeEvent,
  FormEvent,
  useCallback,
  useId,
  useMemo,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';
import {
  DefaultMaxTokens,
  GlobalModelTypeKey,
  toRequestModelType,
} from './constants';
import { useCreateGlobalInstance, useTestGlobalConnection } from './hooks';
import { ModelTypeCheckboxes } from './model-type-checkboxes';

export type ModelDraft = {
  key: number;
  name: string;
  types: GlobalModelTypeKey[];
  maxTokens: number;
};

export const toModelInputs = (models: ModelDraft[]): IGlobalModelInput[] =>
  models
    .filter((model) => model.name.trim() && model.types.length > 0)
    .map((model) => ({
      model_name: model.name.trim(),
      model_type: model.types.map(toRequestModelType),
      max_tokens: model.maxTokens,
    }));

let nextDraftKey = 1;
export const newModelDraft = (): ModelDraft => ({
  key: nextDraftKey++,
  name: '',
  types: ['chat'],
  maxTokens: DefaultMaxTokens,
});

/** Edits one model of the dialog's model list. */
export function ModelDraftRow({
  draft,
  removable,
  onChange,
  onRemove,
}: {
  draft: ModelDraft;
  removable: boolean;
  onChange: (draft: ModelDraft) => void;
  onRemove?: (key: number) => void;
}) {
  const { t } = useTranslation();
  const handleName = useCallback(
    (event: ChangeEvent<HTMLInputElement>) =>
      onChange({ ...draft, name: event.target.value }),
    [draft, onChange],
  );
  const handleMaxTokens = useCallback(
    (event: ChangeEvent<HTMLInputElement>) =>
      onChange({ ...draft, maxTokens: Number(event.target.value) || 0 }),
    [draft, onChange],
  );
  const handleTypes = useCallback(
    (types: GlobalModelTypeKey[]) => onChange({ ...draft, types }),
    [draft, onChange],
  );
  const handleRemove = useCallback(
    () => onRemove?.(draft.key),
    [draft.key, onRemove],
  );
  return (
    <div className="space-y-3 rounded-lg bg-bg-card p-3">
      <div className="flex items-center gap-2">
        <Input
          aria-label={t('globalModel.modelName')}
          placeholder={t('globalModel.modelNamePlaceholder')}
          value={draft.name}
          onChange={handleName}
        />
        <Input
          className="w-32"
          type="number"
          min={1}
          aria-label={t('globalModel.maxTokens')}
          title={t('globalModel.maxTokens')}
          value={draft.maxTokens}
          onChange={handleMaxTokens}
        />
        {removable && (
          <Button
            variant="ghost"
            size="icon"
            aria-label={t('globalModel.removeModel')}
            onClick={handleRemove}
          >
            <LucideTrash2 />
          </Button>
        )}
      </div>
      <ModelTypeCheckboxes value={draft.types} onChange={handleTypes} />
    </div>
  );
}

/** Adds an instance (endpoint + optional key) with its models to a provider. */
export function InstanceDialog({
  provider,
  onClose,
}: {
  provider?: IGlobalProvider;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const formId = useId();
  const [name, setName] = useState('default');
  const [baseURL, setBaseURL] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [region, setRegion] = useState('');
  const [models, setModels] = useState<ModelDraft[]>(() => [newModelDraft()]);
  const { mutate: createInstance, loading: creating } =
    useCreateGlobalInstance();
  const { testConnection, loading: testing } = useTestGlobalConnection();

  const modelInputs = useMemo(() => toModelInputs(models), [models]);

  const handleOpenChange = useCallback(
    (open: boolean) => {
      if (!open) onClose();
    },
    [onClose],
  );
  const handleName = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => setName(event.target.value),
    [],
  );
  const handleBaseURL = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => setBaseURL(event.target.value),
    [],
  );
  const handleAPIKey = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => setAPIKey(event.target.value),
    [],
  );
  const handleRegion = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => setRegion(event.target.value),
    [],
  );
  const handleModelChange = useCallback(
    (draft: ModelDraft) =>
      setModels((previous) =>
        previous.map((item) => (item.key === draft.key ? draft : item)),
      ),
    [],
  );
  const handleModelRemove = useCallback(
    (key: number) =>
      setModels((previous) => previous.filter((item) => item.key !== key)),
    [],
  );
  const handleAddModel = useCallback(
    () => setModels((previous) => [...previous, newModelDraft()]),
    [],
  );

  const handleTest = useCallback(async () => {
    if (!provider) return;
    try {
      await testConnection({
        providerId: provider.id,
        body: {
          api_key: apiKey.trim(),
          base_url: baseURL.trim(),
          region: region.trim(),
          model_info: modelInputs,
        },
      });
    } catch {
      // The admin request client already shows the error.
      return;
    }
    message.success(t('globalModel.connectionSucceeded'));
  }, [apiKey, baseURL, modelInputs, provider, region, t, testConnection]);

  const handleSubmit = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (!provider) return;
      try {
        await createInstance({
          providerId: provider.id,
          body: {
            instance_name: name.trim(),
            api_key: apiKey.trim(),
            base_url: baseURL.trim(),
            region: region.trim(),
            models: modelInputs,
          },
        });
      } catch {
        // The admin request client already shows the error; keep the dialog.
        return;
      }
      message.success(t('globalModel.instanceCreated'));
      onClose();
    },
    [
      apiKey,
      baseURL,
      createInstance,
      modelInputs,
      name,
      onClose,
      provider,
      region,
      t,
    ],
  );

  return (
    <Dialog open={Boolean(provider)} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {t('globalModel.addInstanceTo', {
              provider: provider?.provider_name,
            })}
          </DialogTitle>
        </DialogHeader>
        <form
          id={formId}
          className="max-h-[60vh] space-y-4 overflow-auto px-6"
          onSubmit={handleSubmit}
        >
          <div className="grid gap-3 md:grid-cols-2">
            <label className="space-y-1.5 text-sm">
              <span>{t('globalModel.instanceName')}</span>
              <Input required value={name} onChange={handleName} />
            </label>
            <label className="space-y-1.5 text-sm">
              <span>{t('globalModel.region')}</span>
              <Input value={region} onChange={handleRegion} />
            </label>
          </div>
          <label className="block space-y-1.5 text-sm">
            <span>{t('globalModel.baseUrl')}</span>
            <Input
              placeholder="http://host:8000/v1"
              value={baseURL}
              onChange={handleBaseURL}
            />
          </label>
          <label className="block space-y-1.5 text-sm">
            <span>{t('globalModel.apiKey')}</span>
            <Input
              type="password"
              autoComplete="new-password"
              placeholder={t('globalModel.apiKeyPlaceholder')}
              value={apiKey}
              onChange={handleAPIKey}
            />
          </label>
          <div className="space-y-2">
            <div className="text-sm">{t('globalModel.models')}</div>
            {models.map((draft) => (
              <ModelDraftRow
                key={draft.key}
                draft={draft}
                removable={models.length > 1}
                onChange={handleModelChange}
                onRemove={handleModelRemove}
              />
            ))}
            <Button variant="outline" size="sm" onClick={handleAddModel}>
              <LucidePlus />
              {t('globalModel.addAnotherModel')}
            </Button>
          </div>
        </form>
        <DialogFooter className="gap-4 px-6 py-4">
          <Button
            variant="outline"
            loading={testing}
            disabled={modelInputs.length === 0}
            onClick={handleTest}
          >
            {t('globalModel.testConnection')}
          </Button>
          <Button
            type="submit"
            form={formId}
            loading={creating}
            disabled={!name.trim() || modelInputs.length === 0}
          >
            {t('globalModel.create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
