import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { LucideTrash2 } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { ConfirmDeleteDialog } from '@/components/confirm-delete-dialog';
import CopyToClipboard from '@/components/copy-to-clipboard';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import message from '@/components/ui/message';
import { Switch } from '@/components/ui/switch';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { IAdminAPIKey } from '@/interfaces/database/access-profile';
import {
  createUserAPIKey,
  deleteUserAPIKey,
  listUserAPIKeys,
  setUserAPIKeyEnabled,
} from '@/services/admin-service';
import { formatDate } from '@/utils/date';

function unwrap<T>(response: {
  data: { code: number; message: string; data: T };
}) {
  if (response.data.code !== 0) {
    throw new Error(response.data.message);
  }
  return response.data.data;
}

const userAPIKeysQueryKey = (email: string) =>
  ['adminAccessProfile', 'apiKeys', email] as const;

/**
 * Lists, creates, switches and deletes the API keys a super admin manages for
 * one user. A new key is shown in full once, right after it is created.
 */
export function UserAPIKeysDialog({
  email,
  open,
  onOpenChange,
}: {
  email: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const queryKey = userAPIKeysQueryKey(email);
  const [name, setName] = useState('');
  const [createdKey, setCreatedKey] = useState<IAdminAPIKey | null>(null);

  const { data: keys = [], isFetching } = useQuery<IAdminAPIKey[]>({
    queryKey,
    queryFn: async () => unwrap(await listUserAPIKeys(email)) ?? [],
    enabled: open,
    retry: false,
  });

  const refresh = () => queryClient.invalidateQueries({ queryKey });

  const createMutation = useMutation({
    mutationFn: async (keyName: string) =>
      unwrap(await createUserAPIKey(email, keyName)),
    onSuccess: async (key) => {
      setCreatedKey(key);
      setName('');
      await refresh();
    },
    retry: false,
  });

  const enabledMutation = useMutation({
    mutationFn: async (input: { id: string; enabled: boolean }) =>
      unwrap(await setUserAPIKeyEnabled(email, input.id, input.enabled)),
    onSuccess: refresh,
    retry: false,
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => unwrap(await deleteUserAPIKey(email, id)),
    onSuccess: async () => {
      message.success(t('accessProfile.apiKeys.deleted'));
      await refresh();
    },
    retry: false,
  });

  const handleCreate = (event: FormEvent) => {
    event.preventDefault();
    const trimmed = name.trim();
    if (trimmed) {
      createMutation.mutate(trimmed);
    }
  };

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setCreatedKey(null);
      setName('');
    }
    onOpenChange(next);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>
            {t('accessProfile.apiKeys.open', { email })}
          </DialogTitle>
          <DialogDescription>
            {t('accessProfile.apiKeys.description')}
          </DialogDescription>
        </DialogHeader>

        {createdKey?.token ? (
          <div
            role="status"
            className="rounded-md border border-state-success p-3 space-y-2"
          >
            <p className="text-sm">{t('accessProfile.apiKeys.created_once')}</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 break-all text-sm">
                {createdKey.token}
              </code>
              <CopyToClipboard
                text={createdKey.token}
                aria-label={t('accessProfile.apiKeys.copy')}
              />
            </div>
          </div>
        ) : (
          <form className="flex items-center gap-2" onSubmit={handleCreate}>
            <Input
              aria-label={t('accessProfile.apiKeys.name')}
              placeholder={t('accessProfile.apiKeys.namePlaceholder')}
              maxLength={128}
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
            <Button
              type="submit"
              disabled={!name.trim() || createMutation.isPending}
            >
              {t('accessProfile.apiKeys.create')}
            </Button>
          </form>
        )}

        {keys.length === 0 ? (
          <p className="text-text-secondary text-sm py-4">
            {isFetching ? '' : t('accessProfile.apiKeys.empty')}
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('accessProfile.apiKeys.name')}</TableHead>
                <TableHead>{t('accessProfile.apiKeys.key')}</TableHead>
                <TableHead>{t('accessProfile.apiKeys.created')}</TableHead>
                <TableHead>{t('accessProfile.apiKeys.enabled')}</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell className="max-w-40 truncate">
                    {key.name}
                  </TableCell>
                  <TableCell>
                    <code className="text-sm">{key.masked_token}</code>
                  </TableCell>
                  <TableCell>{formatDate(key.create_time)}</TableCell>
                  <TableCell>
                    <Switch
                      aria-label={t('accessProfile.apiKeys.enabledToggle', {
                        name: key.name,
                      })}
                      checked={key.enabled}
                      disabled={enabledMutation.isPending}
                      onCheckedChange={(enabled) =>
                        enabledMutation.mutate({ id: key.id, enabled })
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <ConfirmDeleteDialog
                      title={t('accessProfile.apiKeys.deleteConfirm', {
                        name: key.name,
                      })}
                      onOk={() => deleteMutation.mutate(key.id)}
                    >
                      <Button
                        variant="danger"
                        size="icon"
                        className="border-0"
                        aria-label={t('accessProfile.apiKeys.delete', {
                          name: key.name,
                        })}
                      >
                        <LucideTrash2 />
                      </Button>
                    </ConfirmDeleteDialog>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        <DialogFooter>
          {createdKey?.token ? (
            <Button onClick={() => setCreatedKey(null)}>
              {t('accessProfile.apiKeys.done')}
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
