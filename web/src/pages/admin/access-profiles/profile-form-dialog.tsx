import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { AccessFeature } from '@/constants/access-profile';
import {
  AccessFeatureSwitches,
  IAccessProfile,
} from '@/interfaces/database/access-profile';
import { IAccessProfileRequestBody } from '@/interfaces/request/access-profile';
import { ChangeEvent, FormEvent, useCallback, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FeatureSwitchList } from './feature-switch-list';

const NameMaxLength = 128;
const DescriptionMaxLength = 512;

type ProfileFormProps = {
  profile?: IAccessProfile;
  loading: boolean;
  onCancel: () => void;
  onSubmit: (body: IAccessProfileRequestBody) => void;
};

/**
 * Creates a profile (`profile` unset) or edits the name and description of an
 * existing one. Switches of an existing profile are edited on its card.
 */
export function ProfileFormDialog({
  open,
  profile,
  loading,
  onOpenChange,
  onSubmit,
}: Omit<ProfileFormProps, 'onCancel'> & {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();
  const handleCancel = useCallback(() => onOpenChange(false), [onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {profile
              ? t('accessProfile.editProfile')
              : t('accessProfile.createProfile')}
          </DialogTitle>
        </DialogHeader>
        {/* Dialog content unmounts when closed, so each open starts from the
            given profile. */}
        <ProfileForm
          profile={profile}
          loading={loading}
          onCancel={handleCancel}
          onSubmit={onSubmit}
        />
      </DialogContent>
    </Dialog>
  );
}

function ProfileForm({
  profile,
  loading,
  onCancel,
  onSubmit,
}: ProfileFormProps) {
  const { t } = useTranslation();
  const formId = useId();
  const nameId = useId();
  const descriptionId = useId();
  const [name, setName] = useState(profile?.name ?? '');
  const [description, setDescription] = useState(profile?.description ?? '');
  const [features, setFeatures] = useState<AccessFeatureSwitches>(
    profile?.features ?? {},
  );

  const handleNameChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => setName(event.target.value),
    [],
  );

  const handleDescriptionChange = useCallback(
    (event: ChangeEvent<HTMLTextAreaElement>) =>
      setDescription(event.target.value),
    [],
  );

  const handleFeatureChange = useCallback(
    (feature: AccessFeature, enabled: boolean) =>
      setFeatures((previous) => ({ ...previous, [feature]: enabled })),
    [],
  );

  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      onSubmit({
        name: name.trim(),
        description: description.trim(),
        features,
      });
    },
    [description, features, name, onSubmit],
  );

  const isEdit = Boolean(profile);

  return (
    <>
      <form id={formId} className="space-y-5 px-6" onSubmit={handleSubmit}>
        <div className="space-y-2">
          <Label htmlFor={nameId}>{t('accessProfile.name')}</Label>
          <Input
            id={nameId}
            required
            maxLength={NameMaxLength}
            disabled={profile?.is_default}
            placeholder={t('accessProfile.namePlaceholder')}
            value={name}
            onChange={handleNameChange}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor={descriptionId}>
            {t('accessProfile.descriptionLabel')}
          </Label>
          <Textarea
            id={descriptionId}
            maxLength={DescriptionMaxLength}
            value={description}
            onChange={handleDescriptionChange}
          />
        </div>

        {!isEdit && (
          <div className="space-y-2">
            <div className="text-sm font-medium">
              {t('accessProfile.features')}
            </div>
            <FeatureSwitchList
              features={features}
              onFeatureChange={handleFeatureChange}
            />
          </div>
        )}
      </form>

      <DialogFooter className="gap-4 px-6 py-4">
        <Button variant="outline" onClick={onCancel}>
          {t('admin.cancel')}
        </Button>
        <Button
          type="submit"
          form={formId}
          loading={loading}
          disabled={!name.trim()}
        >
          {t('admin.confirm')}
        </Button>
      </DialogFooter>
    </>
  );
}
