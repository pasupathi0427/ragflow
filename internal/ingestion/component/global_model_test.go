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
	"ragflow/internal/dao"
	"ragflow/internal/entity"
	"ragflow/internal/globalmodel"
	"ragflow/internal/ingestion/testutil"
	"ragflow/internal/permission"

	"gorm.io/gorm"
)

// setupIngestionGlobalModel seeds tenant-1, whose owner has no providers of
// their own, and a global OpenAI embedding model whose key is stored encrypted.
func setupIngestionGlobalModel(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv(common.EnvRAGFlowConnectorKey, base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	db := testutil.SetupTestDB(t,
		&entity.User{}, &entity.UserTenant{}, &entity.AccessProfile{}, &entity.UserAccessProfile{},
		&entity.TenantModelProvider{}, &entity.TenantModelInstance{}, &entity.TenantModel{}, &entity.SystemSettings{},
	)
	t.Cleanup(testutil.ReplaceDBForTest(t, db))
	key, err := common.EncryptModelAPIKey("sk-global-embed")
	if err != nil {
		t.Fatal(err)
	}
	status := "1"
	for _, row := range []interface{}{
		&entity.User{ID: "owner", Email: "owner@example.com", Nickname: "Owner", IsActive: "1"},
		&entity.UserTenant{ID: "ut-owner", UserID: "owner", TenantID: "tenant-1", Role: "owner", InvitedBy: "owner", Status: &status},
		&entity.TenantModelProvider{ID: "gp", TenantID: entity.GlobalModelOwnerID, UserID: entity.GlobalModelOwnerID, ProviderName: "OpenAI"},
		&entity.TenantModelInstance{ID: "gi", ProviderID: "gp", InstanceName: "shared", APIKey: key, Status: "active", Extra: `{"base_url":"https://example.invalid/v1"}`},
		&entity.TenantModel{ID: "gm", ProviderID: "gp", InstanceID: "gi", ModelName: "text-embedding-3-small", ModelType: int(entity.ModelTypeEmbedding), Status: "active"},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed %T: %v", row, err)
		}
	}
	return db
}

func TestIngestionResolvesGlobalModelWithDecryptedKey(t *testing.T) {
	db := setupIngestionGlobalModel(t)
	ctx := permission.WithIdentity(context.Background(), "owner", "tenant-1")

	resolved, err := resolveModelConfigByID(ctx, db, "tenant-1", entity.ModelTypeEmbedding, "gm")
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	if resolved.model.ID != "gm" || resolved.apiConfig == nil || resolved.apiConfig.ApiKey == nil || *resolved.apiConfig.ApiKey != "sk-global-embed" {
		t.Fatalf("by id: resolved %+v", resolved)
	}

	_, _, apiConfig, _, err := resolveModelConfigFromProviderInstance(ctx, db, "tenant-1", entity.ModelTypeEmbedding, "text-embedding-3-small@shared@OpenAI")
	if err != nil || apiConfig == nil || apiConfig.ApiKey == nil || *apiConfig.ApiKey != "sk-global-embed" {
		t.Fatalf("by name: %+v (%v)", apiConfig, err)
	}
}

func TestIngestionRefusesDisabledGlobalModel(t *testing.T) {
	db := setupIngestionGlobalModel(t)
	ctx := permission.WithIdentity(context.Background(), "owner", "tenant-1")
	if err := db.Model(&entity.TenantModelInstance{}).Where("id = ?", "gi").Update("status", "inactive").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := resolveModelConfigByID(ctx, db, "tenant-1", entity.ModelTypeEmbedding, "gm"); !errors.Is(err, globalmodel.ErrDisabled) {
		t.Fatalf("by id: %v, want disabled", err)
	}
	if _, _, _, _, err := resolveModelConfigFromProviderInstance(ctx, db, "tenant-1", entity.ModelTypeEmbedding, "text-embedding-3-small@shared@OpenAI"); !errors.Is(err, globalmodel.ErrDisabled) {
		t.Fatalf("by name: %v, want disabled", err)
	}
}

func TestIngestionUsesGlobalDefaultEmbedding(t *testing.T) {
	db := setupIngestionGlobalModel(t)
	ctx := permission.WithIdentity(context.Background(), "owner", "tenant-1")
	if _, _, _, _, err := defaultResolveTenantModelByType(ctx, db, "tenant-1", entity.ModelTypeEmbedding); err == nil {
		t.Fatal("resolved a default although none is set")
	}
	if err := dao.NewSystemSettingsDAO().SaveOrCreate(context.Background(), db, globalmodel.DefaultsSetting, `{"embedding":"gm"}`, "admin", "json"); err != nil {
		t.Fatal(err)
	}
	_, modelName, apiConfig, _, err := defaultResolveTenantModelByType(ctx, db, "tenant-1", entity.ModelTypeEmbedding)
	if err != nil || modelName != "text-embedding-3-small" || apiConfig == nil || *apiConfig.ApiKey != "sk-global-embed" {
		t.Fatalf("global default = %q %+v (%v)", modelName, apiConfig, err)
	}
}

func TestIngestionIDOnlyDefaultDoesNotFallBackToGlobal(t *testing.T) {
	db := setupIngestionGlobalModel(t)
	ctx := permission.WithIdentity(context.Background(), "owner", "tenant-1")
	if err := dao.NewSystemSettingsDAO().SaveOrCreate(context.Background(), db, globalmodel.DefaultsSetting, `{"embedding":"gm"}`, "admin", "json"); err != nil {
		t.Fatal(err)
	}
	// The member chose a default by ID only; that choice must not be replaced
	// by the global default.
	if err := db.Model(&entity.UserTenant{}).Where("id = ?", "ut-owner").Update("tenant_embd_id", "members-own-model").Error; err != nil {
		t.Fatal(err)
	}
	_, modelName, _, _, err := defaultResolveTenantModelByType(ctx, db, "tenant-1", entity.ModelTypeEmbedding)
	if err == nil || modelName == "text-embedding-3-small" {
		t.Fatalf("ID-only default fell back to the global model %q (%v)", modelName, err)
	}
}
