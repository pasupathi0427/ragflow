import { GlobalModelBadge } from '@/components/global-model-badge';
import { LlmIcon } from '@/components/svg-icon';
import { TruncatedText } from '@/components/truncated-text';
import { useFetchAllAddedModels } from '@/hooks/use-llm-request';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

/**
 * Read-only list of the models the organization's admin provides to every
 * user. Users can pick them anywhere a model is chosen but cannot change them.
 */
export function OrganizationModels() {
  const { t } = useTranslation();
  const { data } = useFetchAllAddedModels();
  const globalModels = useMemo(
    () => data.filter((model) => model.is_global),
    [data],
  );

  if (globalModels.length === 0) {
    return null;
  }

  return (
    <section
      className="mx-6 mb-6 rounded-lg border-0.5 border-border-button p-4"
      data-testid="organization-models"
    >
      <h3 className="text-base font-medium text-text-primary">
        {t('globalModel.organizationModels')}
      </h3>
      <p className="mt-1 text-sm text-text-secondary">
        {t('globalModel.organizationModelsDescription')}
      </p>
      <ul className="mt-4 divide-y-0.5 divide-border-button">
        {globalModels.map((model) => (
          <li
            key={model.model_id}
            className="flex min-w-0 items-center gap-3 py-2"
          >
            <LlmIcon
              name={model.provider_name}
              width={22}
              height={22}
              imgClass="size-[22px] flex-shrink-0"
            />
            <div className="min-w-0 flex-1">
              <TruncatedText className="truncate text-sm" tooltip={model.name}>
                {model.name}
              </TruncatedText>
              <TruncatedText
                className="truncate text-xs text-text-secondary"
                tooltip={`${model.provider_name} / ${model.instance_name}`}
              >
                {model.provider_name} / {model.instance_name}
              </TruncatedText>
            </div>
            <span className="shrink-0 text-xs text-text-secondary">
              {model.model_type.join(', ')}
            </span>
            <GlobalModelBadge />
          </li>
        ))}
      </ul>
    </section>
  );
}
