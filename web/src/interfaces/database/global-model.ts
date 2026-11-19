export enum GlobalModelStatus {
  Active = 'active',
  Inactive = 'inactive',
}

export interface IGlobalModel {
  id: string;
  model_name: string;
  model_types: string[];
  status: GlobalModelStatus | string;
}

export interface IGlobalInstance {
  id: string;
  instance_name: string;
  status: GlobalModelStatus | string;
  base_url: string;
  region: string;
  has_api_key: boolean;
  /** Last four characters only, e.g. "••••1234". The key is never returned. */
  api_key_masked: string;
  models: IGlobalModel[];
}

export interface IGlobalProvider {
  id: string;
  provider_name: string;
  instances: IGlobalInstance[];
}

/** A provider from the model catalog that can be added globally. */
export interface IGlobalCatalogProvider {
  name: string;
  [key: string]: unknown;
}

/** Global default model ID per model type name (chat, embedding, ...). */
export type GlobalModelDefaults = Partial<Record<string, string>>;
