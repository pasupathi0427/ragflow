import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  IAccessProfile,
  IAccessProfileAssignment,
} from '@/interfaces/database/access-profile';
import { IAccessProfileRequestBody } from '@/interfaces/request/access-profile';
import {
  assignAccessProfile,
  createAccessProfile,
  deleteAccessProfile,
  listAccessProfileAssignments,
  listAccessProfiles,
  updateAccessProfile,
} from '@/services/admin-service';

export const AdminAccessProfileKeys = {
  all: ['adminAccessProfile'] as const,
  profiles: () => [...AdminAccessProfileKeys.all, 'profiles'] as const,
  assignments: () => [...AdminAccessProfileKeys.all, 'assignments'] as const,
};

function unwrapAdminResponse<T>(response: {
  data: { code: number; message: string; data: T };
}) {
  if (response.data.code !== 0) {
    throw new Error(response.data.message);
  }
  return response.data.data;
}

export const useFetchAccessProfiles = () => {
  const { data, isFetching: loading } = useQuery<IAccessProfile[]>({
    queryKey: AdminAccessProfileKeys.profiles(),
    initialData: [],
    queryFn: async () => unwrapAdminResponse(await listAccessProfiles()) ?? [],
    retry: false,
  });

  return { data, loading };
};

export const useFetchAccessProfileAssignments = () => {
  const { data, isFetching: loading } = useQuery<IAccessProfileAssignment[]>({
    queryKey: AdminAccessProfileKeys.assignments(),
    initialData: [],
    queryFn: async () =>
      unwrapAdminResponse(await listAccessProfileAssignments()) ?? [],
    retry: false,
  });

  return { data, loading };
};

// Returned from onSuccess so the mutation stays pending until the lists have
// refetched; a follow-up edit then starts from the saved state.
const useInvalidateAdminAccessProfiles = () => {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({
        queryKey: AdminAccessProfileKeys.profiles(),
      }),
      queryClient.invalidateQueries({
        queryKey: AdminAccessProfileKeys.assignments(),
      }),
    ]);
};

export const useCreateAccessProfile = () => {
  const invalidate = useInvalidateAdminAccessProfiles();
  const { mutateAsync, isPending: loading } = useMutation({
    mutationFn: async (body: IAccessProfileRequestBody) =>
      unwrapAdminResponse(await createAccessProfile(body)),
    onSuccess: invalidate,
    retry: false,
  });

  return { createAccessProfile: mutateAsync, loading };
};

export const useUpdateAccessProfile = () => {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateAdminAccessProfiles();
  const { mutateAsync, isPending: loading } = useMutation({
    mutationFn: async ({
      profileId,
      body,
    }: {
      profileId: string;
      body: IAccessProfileRequestBody;
    }) => unwrapAdminResponse(await updateAccessProfile(profileId, body)),
    onSuccess: (updated) => {
      // Show the saved switches right away, before the refetch lands.
      queryClient.setQueryData<IAccessProfile[]>(
        AdminAccessProfileKeys.profiles(),
        (profiles) =>
          profiles?.map((profile) =>
            profile.id === updated.id ? updated : profile,
          ),
      );
      return invalidate();
    },
    retry: false,
  });

  return { updateAccessProfile: mutateAsync, loading };
};

export const useDeleteAccessProfile = () => {
  const invalidate = useInvalidateAdminAccessProfiles();
  const { mutateAsync, isPending: loading } = useMutation({
    mutationFn: async (profileId: string) =>
      unwrapAdminResponse(await deleteAccessProfile(profileId)),
    onSuccess: invalidate,
    retry: false,
  });

  return { deleteAccessProfile: mutateAsync, loading };
};

export const useAssignAccessProfile = () => {
  const invalidate = useInvalidateAdminAccessProfiles();
  const { mutateAsync, isPending: loading } = useMutation({
    mutationFn: async ({
      email,
      profileId,
    }: {
      email: string;
      profileId: string;
    }) => unwrapAdminResponse(await assignAccessProfile(email, profileId)),
    onSuccess: invalidate,
    retry: false,
  });

  return { assignAccessProfile: mutateAsync, loading };
};
