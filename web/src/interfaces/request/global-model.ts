export interface IGlobalModelInput {
  model_name: string;
  model_type: string[];
  max_tokens: number;
}

export interface IGlobalInstanceRequestBody {
  instance_name?: string;
  /** Plain key on create or replace; omit to keep the saved key. */
  api_key?: string;
  clear_api_key?: boolean;
  base_url?: string;
  region?: string;
  status?: string;
  models?: IGlobalModelInput[];
}

export interface IGlobalConnectionRequestBody {
  instance_id?: string;
  api_key?: string;
  base_url?: string;
  region?: string;
  model_info?: IGlobalModelInput[];
}
