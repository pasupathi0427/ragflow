import { ConfirmDeleteDialog } from '@/components/confirm-delete-dialog';
import Empty from '@/components/empty/empty';
import Spotlight from '@/components/spotlight';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import message from '@/components/ui/message';
import { ScrollArea } from '@/components/ui/scroll-area';
import { IGlobalProvider } from '@/interfaces/database/global-model';
import { LucidePlus } from 'lucide-react';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AddProviderDialog } from './add-provider-dialog';
import { DefaultsCard } from './defaults-card';
import { useDeleteGlobalProvider, useFetchGlobalProviders } from './hooks';
import { InstanceDialog } from './instance-dialog';
import { ProviderCard } from './provider-card';

/**
 * Admin console page for global model providers: models the admin configures
 * once and every user may call. Users can use them but never change them.
 */
function AdminGlobalModels() {
  const { t } = useTranslation();
  const { data: providers } = useFetchGlobalProviders();
  const { mutate: deleteProvider } = useDeleteGlobalProvider();
  const [addingProvider, setAddingProvider] = useState(false);
  const [instanceProvider, setInstanceProvider] = useState<IGlobalProvider>();
  const [deletingProvider, setDeletingProvider] = useState<IGlobalProvider>();
  const [expandedIds, setExpandedIds] = useState<ReadonlySet<string>>(
    () => new Set(),
  );

  const existingNames = useMemo(
    () => providers.map((provider) => provider.provider_name),
    [providers],
  );

  const openAddProvider = useCallback(() => setAddingProvider(true), []);
  const closeAddProvider = useCallback(() => setAddingProvider(false), []);
  const closeInstanceDialog = useCallback(
    () => setInstanceProvider(undefined),
    [],
  );

  const expand = useCallback(
    (providerId: string) =>
      setExpandedIds((previous) => new Set(previous).add(providerId)),
    [],
  );

  const handleProviderAdded = useCallback(
    (added: { id: string; provider_name: string }) => {
      expand(added.id);
      // Go straight on to the provider's first instance.
      setInstanceProvider({ ...added, instances: [] });
    },
    [expand],
  );

  const handleExpandedChange = useCallback(
    (provider: IGlobalProvider, expanded: boolean) =>
      setExpandedIds((previous) => {
        const next = new Set(previous);
        if (expanded) {
          next.add(provider.id);
        } else {
          next.delete(provider.id);
        }
        return next;
      }),
    [],
  );

  const handleAddInstance = useCallback(
    (provider: IGlobalProvider) => {
      expand(provider.id);
      setInstanceProvider(provider);
    },
    [expand],
  );

  const handleDeleteOpenChange = useCallback((open: boolean) => {
    if (!open) setDeletingProvider(undefined);
  }, []);

  const handleConfirmDelete = useCallback(async () => {
    if (!deletingProvider) return;
    try {
      await deleteProvider(deletingProvider.id);
    } catch {
      // The admin request client already shows why, e.g. what still uses it.
      return;
    }
    message.success(t('globalModel.providerDeleted'));
    setDeletingProvider(undefined);
  }, [deleteProvider, deletingProvider, t]);

  return (
    <>
      <Card className="relative size-full rounded-xl border-0.5 border-border-button bg-transparent !shadow-none">
        <Spotlight />
        <ScrollArea className="size-full">
          <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
            <div className="min-w-0 space-y-1.5">
              <CardTitle>{t('globalModel.title')}</CardTitle>
              <p className="text-sm text-text-secondary">
                {t('globalModel.description')}
              </p>
            </div>
            <Button className="h-10 shrink-0 px-4" onClick={openAddProvider}>
              <LucidePlus />
              {t('globalModel.addProvider')}
            </Button>
          </CardHeader>

          <CardContent className="space-y-6">
            <DefaultsCard providers={providers} />
            {providers.length ? (
              providers.map((provider) => (
                <ProviderCard
                  key={provider.id}
                  provider={provider}
                  expanded={expandedIds.has(provider.id)}
                  onExpandedChange={handleExpandedChange}
                  onAddInstance={handleAddInstance}
                  onDelete={setDeletingProvider}
                />
              ))
            ) : (
              <Empty className="py-24" text={t('globalModel.empty')} />
            )}
          </CardContent>
        </ScrollArea>
      </Card>

      {addingProvider && (
        <AddProviderDialog
          existingNames={existingNames}
          onClose={closeAddProvider}
          onAdded={handleProviderAdded}
        />
      )}

      {instanceProvider && (
        <InstanceDialog
          provider={instanceProvider}
          onClose={closeInstanceDialog}
        />
      )}

      <ConfirmDeleteDialog
        open={Boolean(deletingProvider)}
        onOpenChange={handleDeleteOpenChange}
        title={t('globalModel.deleteProvider')}
        content={{
          title: deletingProvider?.provider_name,
          node: t('globalModel.deleteProviderConfirmation'),
        }}
        onOk={handleConfirmDelete}
      />
    </>
  );
}

export default AdminGlobalModels;
