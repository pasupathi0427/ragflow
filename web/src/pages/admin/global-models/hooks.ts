import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  GlobalModelDefaults,
  IGlobalCatalogProvider,
  IGlobalProvider,
} from '@/interfaces/database/global-model';
import {
  IGlobalConnectionRequestBody,
  IGlobalInstanceRequestBody,
  IGlobalModelInput,
} from '@/interfaces/request/global-model';
import {
  addGlobalModel,
  addGlobalProvider,
  createGlobalInstance,
  deleteGlobalInstance,
  deleteGlobalModel,
  deleteGlobalProvider,
  getGlobalModelDefaults,
  listGlobalModelCatalog,
  listGlobalModels,
  setGlobalModelDefaults,
  setGlobalModelStatus,
  testGlobalConnection,
  updateGlobalInstance,
} from '@/services/admin-service';

export const AdminGlobalModelKeys = {
  all: ['adminGlobalModel'] as const,
  providers: () => [...AdminGlobalModelKeys.all, 'providers'] as const,
  catalog: () => [...AdminGlobalModelKeys.all, 'catalog'] as const,
  defaults: () => [...AdminGlobalModelKeys.all, 'defaults'] as const,
};

function unwrapAdminResponse<T>(response: {
  data: { code: number; message: string; data: T };
}) {
  if (response.data.code !== 0) {
    throw new Error(response.data.message);
  }
  return response.data.data;
}

export const useFetchGlobalProviders = () => {
  const { data, isFetching: loading } = useQuery<IGlobalProvider[]>({
    queryKey: AdminGlobalModelKeys.providers(),
    initialData: [],
    queryFn: async () => unwrapAdminResponse(await listGlobalModels()) ?? [],
    retry: false,
  });
  return { data, loading };
};

export const useFetchGlobalCatalog = (enabled: boolean) => {
  const { data, isFetching: loading } = useQuery<IGlobalCatalogProvider[]>({
    queryKey: AdminGlobalModelKeys.catalog(),
    initialData: [],
    enabled,
    queryFn: async () =>
      unwrapAdminResponse(await listGlobalModelCatalog()) ?? [],
    retry: false,
  });
  return { data, loading };
};

export const useFetchGlobalDefaults = () => {
  const { data, isFetching: loading } = useQuery<GlobalModelDefaults>({
    queryKey: AdminGlobalModelKeys.defaults(),
    initialData: {},
    queryFn: async () =>
      unwrapAdminResponse(await getGlobalModelDefaults()) ?? {},
    retry: false,
  });
  return { data, loading };
};

// Returned from onSuccess so a mutation stays pending until the lists have
// refetched; a follow-up edit then starts from the saved state.
const useInvalidateGlobalModels = () => {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({
        queryKey: AdminGlobalModelKeys.providers(),
      }),
      queryClient.invalidateQueries({
        queryKey: AdminGlobalModelKeys.defaults(),
      }),
    ]);
};

function useGlobalModelMutation<TVariables, TResult>(
  mutationFn: (variables: TVariables) => Promise<TResult>,
) {
  const invalidate = useInvalidateGlobalModels();
  const { mutateAsync, isPending } = useMutation({
    mutationFn,
    onSuccess: invalidate,
    retry: false,
  });
  return { mutate: mutateAsync, loading: isPending };
}

export const useAddGlobalProvider = () =>
  useGlobalModelMutation(async (providerName: string) =>
    unwrapAdminResponse(await addGlobalProvider(providerName)),
  );

export const useDeleteGlobalProvider = () =>
  useGlobalModelMutation(async (providerId: string) =>
    unwrapAdminResponse(await deleteGlobalProvider(providerId)),
  );

export const useCreateGlobalInstance = () =>
  useGlobalModelMutation(
    async ({
      providerId,
      body,
    }: {
      providerId: string;
      body: IGlobalInstanceRequestBody;
    }) => unwrapAdminResponse(await createGlobalInstance(providerId, body)),
  );

export const useUpdateGlobalInstance = () =>
  useGlobalModelMutation(
    async ({
      instanceId,
      body,
    }: {
      instanceId: string;
      body: IGlobalInstanceRequestBody;
    }) => unwrapAdminResponse(await updateGlobalInstance(instanceId, body)),
  );

export const useDeleteGlobalInstance = () =>
  useGlobalModelMutation(async (instanceId: string) =>
    unwrapAdminResponse(await deleteGlobalInstance(instanceId)),
  );

export const useAddGlobalModel = () =>
  useGlobalModelMutation(
    async ({
      instanceId,
      model,
    }: {
      instanceId: string;
      model: IGlobalModelInput;
    }) => unwrapAdminResponse(await addGlobalModel(instanceId, model)),
  );

export const useSetGlobalModelStatus = () =>
  useGlobalModelMutation(
    async ({ modelId, status }: { modelId: string; status: string }) =>
      unwrapAdminResponse(await setGlobalModelStatus(modelId, status)),
  );

export const useDeleteGlobalModel = () =>
  useGlobalModelMutation(async (modelId: string) =>
    unwrapAdminResponse(await deleteGlobalModel(modelId)),
  );

export const useSetGlobalDefaults = () =>
  useGlobalModelMutation(async (defaults: GlobalModelDefaults) =>
    unwrapAdminResponse(await setGlobalModelDefaults(defaults)),
  );

export const useTestGlobalConnection = () => {
  const { mutateAsync, isPending } = useMutation({
    mutationFn: async ({
      providerId,
      body,
    }: {
      providerId: string;
      body: IGlobalConnectionRequestBody;
    }) => unwrapAdminResponse(await testGlobalConnection(providerId, body)),
    retry: false,
  });
  return { testConnection: mutateAsync, loading: isPending };
};
