//
//  Copyright 2026 The InfiniFlow Authors. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package component

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"ragflow/internal/common"
	"ragflow/internal/entity"
	"ragflow/internal/globalmodel"
	"ragflow/internal/ingestion/testutil"
	"ragflow/internal/permission"
)

func TestAgentResolvesAndStopsGlobalModels(t *testing.T) {
	// The global instance below is seeded with a plaintext key; the entity's
	// BeforeSave hook encrypts it, and AfterFind decrypts it for the call.
	t.Setenv(common.EnvRAGFlowConnectorKey, base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	db := testutil.SetupTestDB(t,
		&entity.User{}, &entity.UserTenant{}, &entity.AccessProfile{}, &entity.UserAccessProfile{},
		&entity.TenantModelProvider{}, &entity.TenantModelInstance{}, &entity.TenantModel{},
	)
	t.Cleanup(testutil.ReplaceDBForTest(t, db))
	status := "1"
	for _, row := range []interface{}{
		&entity.User{ID: "owner", Email: "owner@example.com", Nickname: "Owner", IsActive: "1"},
		&entity.UserTenant{ID: "ut-owner", UserID: "owner", TenantID: "tenant-1", Role: "owner", InvitedBy: "owner", Status: &status},
		&entity.TenantModelProvider{ID: "gp", TenantID: entity.GlobalModelOwnerID, UserID: entity.GlobalModelOwnerID, ProviderName: "VLLM"},
		&entity.TenantModelInstance{ID: "gi", ProviderID: "gp", InstanceName: "shared", APIKey: "sk-vllm", Status: "active", Extra: `{"base_url":"http://vllm:8000/v1"}`},
		&entity.TenantModel{ID: "gm", ProviderID: "gp", InstanceID: "gi", ModelName: "qwen", ModelType: int(entity.ModelTypeChat), Status: "active"},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed %T: %v", row, err)
		}
	}
	var stored string
	if err := db.Raw("SELECT api_key FROM tenant_model_instance WHERE id = ?", "gi").Scan(&stored).Error; err != nil || !common.IsEncryptedModelAPIKey(stored) {
		t.Fatalf("global key stored as %q (%v), want encrypted", stored, err)
	}
	ctx := permission.WithIdentity(context.Background(), "owner", "tenant-1")

	modelName, provider, apiKey, baseURL, _, ok, err := resolveTenantModelByTenantModelID(ctx, db, "tenant-1", "gm", "", "", entity.ModelTypeChat)
	if err != nil || !ok || modelName != "qwen" || provider != "VLLM" || apiKey != "sk-vllm" || baseURL != "http://vllm:8000/v1" {
		t.Fatalf("global model = %q %q %q %q ok=%v (%v)", modelName, provider, apiKey, baseURL, ok, err)
	}

	// The admin's key only goes to the endpoint the admin configured.
	if _, _, _, _, _, _, err := resolveTenantModelByTenantModelID(ctx, db, "tenant-1", "gm", "", "https://attacker.example/v1", entity.ModelTypeChat); !errors.Is(err, globalmodel.ErrEndpointOverride) {
		t.Fatalf("base_url override: %v, want refusal", err)
	}
	if _, _, _, _, _, _, err := resolveTenantModelByInstanceID(ctx, db, "tenant-1", "gi", "", "https://attacker.example/v1", entity.ModelTypeChat); !errors.Is(err, globalmodel.ErrEndpointOverride) {
		t.Fatalf("base_url override by instance id: %v, want refusal", err)
	}
	if key, _, ok := resolveTenantModelInstanceCredentials(ctx, db, "tenant-1", "qwen@shared@VLLM", "https://attacker.example/v1"); ok || key != "" {
		t.Fatalf("fallback sent the global key to an override: %q", key)
	}
	if _, _, key, _, _, ok, err := resolveTenantModelByTenantModelID(ctx, db, "tenant-1", "gm", "", "http://vllm:8000/v1/", entity.ModelTypeChat); err != nil || !ok || key != "sk-vllm" {
		t.Fatalf("same-origin base_url: key=%q ok=%v (%v)", key, ok, err)
	}

	if err := db.Model(&entity.TenantModelInstance{}).Where("id = ?", "gi").Update("status", "inactive").Error; err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, _, err := resolveTenantModelByTenantModelID(ctx, db, "tenant-1", "gm", "", "", entity.ModelTypeChat); !errors.Is(err, globalmodel.ErrDisabled) {
		t.Fatalf("disabled instance: %v, want disabled", err)
	}
	if _, _, _, _, _, _, err := resolveTenantModelByInstanceID(ctx, db, "tenant-1", "gi", "", "", entity.ModelTypeChat); !errors.Is(err, globalmodel.ErrDisabled) {
		t.Fatalf("disabled instance by instance id: %v, want disabled", err)
	}
	if key, _, ok := resolveTenantModelInstanceCredentials(ctx, db, "tenant-1", "qwen@shared@VLLM", ""); ok || key != "" {
		t.Fatalf("fallback handed out credentials of a disabled model: %q", key)
	}
}
