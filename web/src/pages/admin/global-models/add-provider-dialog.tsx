import { SelectWithSearch } from '@/components/originui/select-with-search';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAddGlobalProvider, useFetchGlobalCatalog } from './hooks';

/** Adds a provider from the model catalog as a global provider. */
export function AddProviderDialog({
  existingNames,
  onClose,
  onAdded,
}: {
  existingNames: string[];
  onClose: () => void;
  onAdded: (provider: { id: string; provider_name: string }) => void;
}) {
  const { t } = useTranslation();
  const [providerName, setProviderName] = useState('');
  const { data: catalog } = useFetchGlobalCatalog(true);
  const { mutate: addProvider, loading } = useAddGlobalProvider();

  const options = useMemo(
    () =>
      catalog
        .filter((provider) => !existingNames.includes(provider.name))
        .map((provider) => ({ label: provider.name, value: provider.name }))
        .sort((a, b) => a.label.localeCompare(b.label)),
    [catalog, existingNames],
  );

  const handleOpenChange = useCallback(
    (open: boolean) => {
      if (!open) onClose();
    },
    [onClose],
  );

  const handleAdd = useCallback(async () => {
    let added: { id: string; provider_name: string } | undefined;
    try {
      added = await addProvider(providerName);
    } catch {
      // The admin request client already shows the error.
      return;
    }
    if (added) onAdded(added);
    onClose();
  }, [addProvider, onAdded, onClose, providerName]);

  return (
    <Dialog open onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('globalModel.addProvider')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-2 px-6">
          <p className="text-sm text-text-secondary">
            {t('globalModel.addProviderDescription')}
          </p>
          <SelectWithSearch
            options={options}
            value={providerName}
            onChange={setProviderName}
            placeholder={t('globalModel.chooseProvider')}
          />
        </div>
        <DialogFooter className="gap-4 px-6 py-4">
          <Button variant="outline" onClick={onClose}>
            {t('admin.cancel')}
          </Button>
          <Button
            loading={loading}
            disabled={!providerName}
            onClick={handleAdd}
          >
            {t('globalModel.addProvider')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
