import { TruncatedText } from '@/components/truncated-text';
import message from '@/components/ui/message';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { DefaultAccessProfileId } from '@/constants/access-profile';
import {
  useAssignAccessProfile,
  useFetchAccessProfileAssignments,
  useFetchAccessProfiles,
} from './hooks';
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';

/**
 * Shows and changes a user's access profile in the admin user table. The
 * profile and assignment lists are shared queries, so every row reuses one
 * request of each.
 */
export function UserAccessProfileSelect({
  email,
  isSuperuser,
}: {
  email: string;
  isSuperuser?: boolean;
}) {
  const { t } = useTranslation();
  const { data: profiles } = useFetchAccessProfiles();
  const { data: assignments } = useFetchAccessProfileAssignments();
  const { assignAccessProfile, loading } = useAssignAccessProfile();

  const profileId = useMemo(
    () =>
      assignments.find((assignment) => assignment.email === email)
        ?.profile_id ?? DefaultAccessProfileId,
    [assignments, email],
  );

  const handleValueChange = useCallback(
    async (value: string) => {
      try {
        await assignAccessProfile({ email, profileId: value });
      } catch {
        // The admin request client already shows the error.
        return;
      }
      message.success(t('accessProfile.assigned'));
    },
    [assignAccessProfile, email, t],
  );

  // Superusers only use the admin console, where profiles do not apply.
  if (isSuperuser || profiles.length === 0) {
    return <span className="text-text-secondary">-</span>;
  }

  return (
    <Select
      disabled={loading}
      value={profileId}
      onValueChange={handleValueChange}
    >
      <SelectTrigger
        className="max-w-48"
        aria-label={t('accessProfile.changeProfileFor', { email })}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {profiles.map((profile) => (
          <SelectItem key={profile.id} value={profile.id}>
            <TruncatedText className="max-w-40 truncate" tooltip={profile.name}>
              {profile.name}
            </TruncatedText>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
