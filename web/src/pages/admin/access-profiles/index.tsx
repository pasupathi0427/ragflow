import { ConfirmDeleteDialog } from '@/components/confirm-delete-dialog';
import Empty from '@/components/empty/empty';
import Spotlight from '@/components/spotlight';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { SearchInput } from '@/components/ui/input';
import message from '@/components/ui/message';
import { ScrollArea } from '@/components/ui/scroll-area';
import { AccessFeature } from '@/constants/access-profile';
import {
  useCreateAccessProfile,
  useDeleteAccessProfile,
  useFetchAccessProfiles,
  useUpdateAccessProfile,
} from './hooks';
import {
  IAccessProfile,
  IGlobalModelAccess,
} from '@/interfaces/database/access-profile';
import { IAccessProfileRequestBody } from '@/interfaces/request/access-profile';
import {
  LucideChevronsDownUp,
  LucideChevronsUpDown,
  LucidePlus,
} from 'lucide-react';
import { ChangeEvent, useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ProfileCard } from './profile-card';
import { ProfileFormDialog } from './profile-form-dialog';
import { filterAccessProfiles } from './utils';

enum DialogMode {
  Closed = 'closed',
  Create = 'create',
  Edit = 'edit',
}

function AdminAccessProfiles() {
  const { t } = useTranslation();
  const { data: profiles } = useFetchAccessProfiles();
  const { createAccessProfile, loading: creating } = useCreateAccessProfile();
  const { updateAccessProfile, loading: updating } = useUpdateAccessProfile();
  const { deleteAccessProfile } = useDeleteAccessProfile();

  const [dialogMode, setDialogMode] = useState(DialogMode.Closed);
  const [editingProfile, setEditingProfile] = useState<IAccessProfile>();
  const [deletingProfile, setDeletingProfile] = useState<IAccessProfile>();
  const [searchKeyword, setSearchKeyword] = useState('');
  // Profiles start collapsed so a long list stays scannable.
  const [expandedIds, setExpandedIds] = useState<ReadonlySet<string>>(
    () => new Set(),
  );

  const visibleProfiles = useMemo(
    () => filterAccessProfiles(profiles, searchKeyword),
    [profiles, searchKeyword],
  );

  const handleSearchChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) =>
      setSearchKeyword(event.target.value),
    [],
  );

  const handleExpandedChange = useCallback(
    (profile: IAccessProfile, expanded: boolean) =>
      setExpandedIds((previous) => {
        const next = new Set(previous);
        if (expanded) {
          next.add(profile.id);
        } else {
          next.delete(profile.id);
        }
        return next;
      }),
    [],
  );

  const expandAll = useCallback(
    () => setExpandedIds(new Set(visibleProfiles.map((profile) => profile.id))),
    [visibleProfiles],
  );

  const collapseAll = useCallback(() => setExpandedIds(new Set()), []);

  const openCreateDialog = useCallback(() => {
    setEditingProfile(undefined);
    setDialogMode(DialogMode.Create);
  }, []);

  const openEditDialog = useCallback((profile: IAccessProfile) => {
    setEditingProfile(profile);
    setDialogMode(DialogMode.Edit);
  }, []);

  const handleDialogOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setDialogMode(DialogMode.Closed);
    }
  }, []);

  const handleDialogSubmit = useCallback(
    async (body: IAccessProfileRequestBody) => {
      try {
        if (editingProfile) {
          await updateAccessProfile({
            profileId: editingProfile.id,
            body: { ...body, features: editingProfile.features },
          });
        } else {
          const created = await createAccessProfile(body);
          // Open the new profile so its switches can be set right away.
          setExpandedIds((previous) => new Set(previous).add(created.id));
        }
      } catch {
        // The admin request client already shows the error; keep the dialog.
        return;
      }
      message.success(t('accessProfile.saved'));
      setDialogMode(DialogMode.Closed);
    },
    [createAccessProfile, editingProfile, t, updateAccessProfile],
  );

  const handleFeatureChange = useCallback(
    (profile: IAccessProfile, feature: AccessFeature, enabled: boolean) =>
      updateAccessProfile({
        profileId: profile.id,
        body: {
          name: profile.name,
          description: profile.description,
          features: { ...profile.features, [feature]: enabled },
        },
      }).catch(() => {
        // The admin request client already shows the error.
      }),
    [updateAccessProfile],
  );

  const handleGlobalModelsChange = useCallback(
    (profile: IAccessProfile, access: IGlobalModelAccess) =>
      updateAccessProfile({
        profileId: profile.id,
        body: {
          name: profile.name,
          description: profile.description,
          features: profile.features,
          global_models: access,
        },
      }).catch(() => {
        // The admin request client already shows the error.
      }),
    [updateAccessProfile],
  );

  const handleDeleteOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setDeletingProfile(undefined);
    }
  }, []);

  const handleConfirmDelete = useCallback(async () => {
    if (!deletingProfile) {
      return;
    }
    try {
      await deleteAccessProfile(deletingProfile.id);
    } catch {
      // The admin request client already shows the error.
      return;
    }
    message.success(t('accessProfile.deleted'));
    setDeletingProfile(undefined);
  }, [deleteAccessProfile, deletingProfile, t]);

  return (
    <>
      <Card className="relative size-full rounded-xl border-0.5 border-border-button bg-transparent !shadow-none">
        <Spotlight />

        <ScrollArea className="size-full">
          <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
            <div className="min-w-0 space-y-1.5">
              <CardTitle>{t('accessProfile.title')}</CardTitle>
              <p className="text-sm text-text-secondary">
                {t('accessProfile.description')}
              </p>
            </div>

            <Button className="h-10 shrink-0 px-4" onClick={openCreateDialog}>
              <LucidePlus />
              {t('accessProfile.newProfile')}
            </Button>
          </CardHeader>

          <CardContent className="space-y-6">
            <div className="flex flex-wrap items-center gap-2">
              <SearchInput
                className="h-10 w-72 max-w-full border-border-button bg-bg-input"
                placeholder={t('accessProfile.searchPlaceholder')}
                aria-label={t('accessProfile.searchPlaceholder')}
                value={searchKeyword}
                onChange={handleSearchChange}
              />
              <div className="ml-auto flex gap-2">
                <Button
                  variant="outline"
                  className="h-10 dark:border-border-button"
                  disabled={visibleProfiles.length === 0}
                  onClick={expandAll}
                >
                  <LucideChevronsUpDown />
                  {t('accessProfile.expandAll')}
                </Button>
                <Button
                  variant="outline"
                  className="h-10 dark:border-border-button"
                  disabled={expandedIds.size === 0}
                  onClick={collapseAll}
                >
                  <LucideChevronsDownUp />
                  {t('accessProfile.collapseAll')}
                </Button>
              </div>
            </div>

            {visibleProfiles.length ? (
              visibleProfiles.map((profile) => (
                <ProfileCard
                  key={profile.id}
                  profile={profile}
                  saving={updating}
                  expanded={expandedIds.has(profile.id)}
                  onExpandedChange={handleExpandedChange}
                  onFeatureChange={handleFeatureChange}
                  onGlobalModelsChange={handleGlobalModelsChange}
                  onEdit={openEditDialog}
                  onDelete={setDeletingProfile}
                />
              ))
            ) : (
              <Empty
                className="py-24"
                text={
                  profiles.length ? t('accessProfile.noMatches') : undefined
                }
              />
            )}
          </CardContent>
        </ScrollArea>
      </Card>

      <ProfileFormDialog
        open={dialogMode !== DialogMode.Closed}
        profile={dialogMode === DialogMode.Edit ? editingProfile : undefined}
        loading={creating || updating}
        onOpenChange={handleDialogOpenChange}
        onSubmit={handleDialogSubmit}
      />

      <ConfirmDeleteDialog
        open={Boolean(deletingProfile)}
        onOpenChange={handleDeleteOpenChange}
        title={t('accessProfile.deleteProfile')}
        content={{
          title: deletingProfile?.name,
          node: t('accessProfile.deleteConfirmation'),
        }}
        onOk={handleConfirmDelete}
      />
    </>
  );
}

export default AdminAccessProfiles;
