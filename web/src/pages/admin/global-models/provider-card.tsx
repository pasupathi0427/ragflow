import { ConfirmDeleteDialog } from '@/components/confirm-delete-dialog';
import { LlmIcon } from '@/components/svg-icon';
import { TruncatedText } from '@/components/truncated-text';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import message from '@/components/ui/message';
import { Switch } from '@/components/ui/switch';
import {
  GlobalModelStatus,
  IGlobalInstance,
  IGlobalModel,
  IGlobalProvider,
} from '@/interfaces/database/global-model';
import { cn } from '@/lib/utils';
import {
  LucideChevronRight,
  LucidePlugZap,
  LucidePlus,
  LucideTrash2,
} from 'lucide-react';
import { useCallback, useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  useAddGlobalModel,
  useDeleteGlobalInstance,
  useDeleteGlobalModel,
  useSetGlobalModelStatus,
  useTestGlobalConnection,
  useUpdateGlobalInstance,
} from './hooks';
import {
  ModelDraft,
  ModelDraftRow,
  newModelDraft,
  toModelInputs,
} from './instance-dialog';

const isActive = (status: string) => status === GlobalModelStatus.Active;

const ignoreError = () => {
  // The admin request client already shows the error.
};

function ModelRow({ model }: { model: IGlobalModel }) {
  const { t } = useTranslation();
  const labelId = useId();
  const { mutate: setStatus, loading } = useSetGlobalModelStatus();
  const { mutate: deleteModel } = useDeleteGlobalModel();

  const handleStatus = useCallback(
    (checked: boolean) =>
      setStatus({
        modelId: model.id,
        status: checked ? GlobalModelStatus.Active : GlobalModelStatus.Inactive,
      }).catch(ignoreError),
    [model.id, setStatus],
  );
  const handleDelete = useCallback(
    () => deleteModel(model.id).catch(ignoreError),
    [deleteModel, model.id],
  );

  return (
    <li className="flex min-w-0 items-center gap-3 py-2">
      <div className="min-w-0 flex-1">
        <TruncatedText
          as="span"
          className="block truncate text-sm"
          tooltip={model.model_name}
          testId={`global-model-${model.id}`}
        >
          <span id={labelId}>{model.model_name}</span>
        </TruncatedText>
        <span className="text-xs text-text-secondary">
          {model.model_types
            .map((type) => t(`globalModel.type.${type}`))
            .join(', ')}
        </span>
      </div>
      <Switch
        aria-labelledby={labelId}
        checked={isActive(model.status)}
        disabled={loading}
        onCheckedChange={handleStatus}
      />
      <ConfirmDeleteDialog
        title={t('globalModel.deleteModel')}
        content={{
          title: model.model_name,
          node: t('globalModel.deleteModelConfirmation'),
        }}
        onOk={handleDelete}
      >
        <Button
          variant="ghost"
          size="icon"
          aria-label={t('globalModel.deleteModel')}
        >
          <LucideTrash2 />
        </Button>
      </ConfirmDeleteDialog>
    </li>
  );
}

function AddModelForm({ instanceId }: { instanceId: string }) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState<ModelDraft | null>(null);
  const { mutate: addModel, loading } = useAddGlobalModel();
  const input = useMemo(
    () => (draft ? toModelInputs([draft])[0] : undefined),
    [draft],
  );

  const handleOpen = useCallback(() => setDraft(newModelDraft()), []);
  const handleCancel = useCallback(() => setDraft(null), []);
  const handleAdd = useCallback(async () => {
    if (!input) return;
    try {
      await addModel({ instanceId, model: input });
    } catch {
      return;
    }
    setDraft(null);
  }, [addModel, input, instanceId]);

  if (!draft) {
    return (
      <Button variant="outline" size="sm" onClick={handleOpen}>
        <LucidePlus />
        {t('globalModel.addModel')}
      </Button>
    );
  }
  return (
    <div className="space-y-2">
      <ModelDraftRow draft={draft} removable={false} onChange={setDraft} />
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" onClick={handleCancel}>
          {t('admin.cancel')}
        </Button>
        <Button
          size="sm"
          loading={loading}
          disabled={!input}
          onClick={handleAdd}
        >
          {t('globalModel.addModel')}
        </Button>
      </div>
    </div>
  );
}

function InstanceSection({
  providerId,
  instance,
}: {
  providerId: string;
  instance: IGlobalInstance;
}) {
  const { t } = useTranslation();
  const statusLabelId = useId();
  const { mutate: updateInstance, loading: updating } =
    useUpdateGlobalInstance();
  const { mutate: deleteInstance } = useDeleteGlobalInstance();
  const { testConnection, loading: testing } = useTestGlobalConnection();

  const handleStatus = useCallback(
    (checked: boolean) =>
      updateInstance({
        instanceId: instance.id,
        body: {
          status: checked
            ? GlobalModelStatus.Active
            : GlobalModelStatus.Inactive,
        },
      }).catch(ignoreError),
    [instance.id, updateInstance],
  );
  const handleDelete = useCallback(
    () => deleteInstance(instance.id).catch(ignoreError),
    [deleteInstance, instance.id],
  );
  const handleTest = useCallback(async () => {
    try {
      await testConnection({
        providerId,
        body: {
          instance_id: instance.id,
          model_info: instance.models
            .filter((model) => isActive(model.status))
            .map((model) => ({
              model_name: model.model_name,
              model_type: model.model_types,
              max_tokens: 0,
            })),
        },
      });
    } catch {
      return;
    }
    message.success(t('globalModel.connectionSucceeded'));
  }, [instance.id, instance.models, providerId, t, testConnection]);

  return (
    <section className="space-y-3 rounded-lg border-0.5 border-border-button p-4">
      <div className="flex min-w-0 flex-wrap items-center gap-3">
        <div className="min-w-0 flex-1">
          <div className="text-sm font-medium" id={statusLabelId}>
            {instance.instance_name}
          </div>
          <TruncatedText
            className="truncate text-xs text-text-secondary"
            tooltip={instance.base_url}
          >
            {instance.base_url || t('globalModel.providerDefaultEndpoint')}
          </TruncatedText>
          <div className="text-xs text-text-secondary">
            {instance.has_api_key
              ? t('globalModel.apiKeySet', { masked: instance.api_key_masked })
              : t('globalModel.noApiKey')}
          </div>
        </div>
        <span className="text-xs text-text-secondary">
          {isActive(instance.status)
            ? t('globalModel.enabled')
            : t('globalModel.disabled')}
        </span>
        <Switch
          aria-labelledby={statusLabelId}
          checked={isActive(instance.status)}
          disabled={updating}
          onCheckedChange={handleStatus}
        />
        <Button
          variant="outline"
          size="sm"
          loading={testing}
          onClick={handleTest}
        >
          <LucidePlugZap />
          {t('globalModel.test')}
        </Button>
        <ConfirmDeleteDialog
          title={t('globalModel.deleteInstance')}
          content={{
            title: instance.instance_name,
            node: t('globalModel.deleteInstanceConfirmation'),
          }}
          onOk={handleDelete}
        >
          <Button
            variant="ghost"
            size="icon"
            aria-label={t('globalModel.deleteInstance')}
          >
            <LucideTrash2 />
          </Button>
        </ConfirmDeleteDialog>
      </div>
      <ul className="divide-y-0.5 divide-border-button">
        {instance.models.map((model) => (
          <ModelRow key={model.id} model={model} />
        ))}
      </ul>
      <AddModelForm instanceId={instance.id} />
    </section>
  );
}

export function ProviderCard({
  provider,
  expanded,
  onExpandedChange,
  onAddInstance,
  onDelete,
}: {
  provider: IGlobalProvider;
  expanded: boolean;
  onExpandedChange: (provider: IGlobalProvider, expanded: boolean) => void;
  onAddInstance: (provider: IGlobalProvider) => void;
  onDelete: (provider: IGlobalProvider) => void;
}) {
  const { t } = useTranslation();
  const modelCount = provider.instances.reduce(
    (total, instance) => total + instance.models.length,
    0,
  );
  const handleOpenChange = useCallback(
    (open: boolean) => onExpandedChange(provider, open),
    [onExpandedChange, provider],
  );
  const handleAddInstance = useCallback(
    () => onAddInstance(provider),
    [onAddInstance, provider],
  );
  const handleDelete = useCallback(
    () => onDelete(provider),
    [onDelete, provider],
  );

  return (
    <Collapsible open={expanded} onOpenChange={handleOpenChange}>
      <Card className="border-0.5 border-border-default bg-transparent">
        <CardHeader
          className={cn(
            'flex flex-row items-center gap-4 space-y-0',
            expanded && 'border-b-0.5 border-border-button',
          )}
        >
          <CollapsibleTrigger asChild>
            <Button
              variant="transparent"
              className="h-auto min-w-0 flex-1 justify-start gap-3 whitespace-normal border-0 p-0 text-left hover:bg-transparent"
            >
              <LucideChevronRight
                className={cn(
                  'size-4 shrink-0 text-text-secondary transition-transform',
                  expanded && 'rotate-90',
                )}
              />
              <LlmIcon
                name={provider.provider_name}
                width={24}
                height={24}
                imgClass="size-6 flex-shrink-0"
              />
              <div className="min-w-0">
                <TruncatedText
                  className="truncate text-lg font-normal text-text-primary"
                  tooltip={provider.provider_name}
                >
                  {provider.provider_name}
                </TruncatedText>
                <div className="text-xs font-normal text-text-secondary">
                  {t('globalModel.providerSummary', {
                    instances: provider.instances.length,
                    models: modelCount,
                  })}
                </div>
              </div>
            </Button>
          </CollapsibleTrigger>
          <Button variant="outline" size="sm" onClick={handleAddInstance}>
            <LucidePlus />
            {t('globalModel.addInstance')}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            aria-label={t('globalModel.deleteProvider')}
            onClick={handleDelete}
          >
            <LucideTrash2 />
          </Button>
        </CardHeader>
        <CollapsibleContent>
          <CardContent className="space-y-4 p-6">
            {provider.instances.length === 0 ? (
              <p className="text-sm text-text-secondary">
                {t('globalModel.noInstances')}
              </p>
            ) : (
              provider.instances.map((instance) => (
                <InstanceSection
                  key={instance.id}
                  providerId={provider.id}
                  instance={instance}
                />
              ))
            )}
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}
