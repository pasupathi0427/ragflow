//
// Copyright 2026 The InfiniFlow Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package service

import (
	"errors"
	"testing"

	"ragflow/internal/dao"
	"ragflow/internal/entity"
	"ragflow/internal/permission"
)

func TestCreateAPIKeyStoresUserAndTenantContext(t *testing.T) {
	ctx := permission.WithIdentity(t.Context(), "user-1", "tenant-1")
	db := setupServiceTestDB(t)
	if err := db.AutoMigrate(&entity.APIToken{}, &entity.UserTenant{}); err != nil {
		t.Fatalf("failed to migrate api_token and user_tenant: %v", err)
	}
	if err := db.Create(&entity.User{ID: "user-1", Email: "user-1@example.com", Nickname: "user-1"}).Error; err != nil {
		t.Fatalf("failed to create API key owner: %v", err)
	}
	if err := db.Create(&entity.Tenant{ID: "tenant-1", ParserIDs: ""}).Error; err != nil {
		t.Fatalf("failed to create Tenant: %v", err)
	}
	status := entity.TenantMembershipStatusJoined
	if err := db.Create(&entity.UserTenant{ID: "membership-1", UserID: "user-1", TenantID: "tenant-1", Role: "owner", Status: &status}).Error; err != nil {
		t.Fatalf("failed to create Tenant membership: %v", err)
	}
	pushServiceDB(t, db)

	key, err := (&SystemService{}).CreateAPIKey(ctx, &CreateAPIKeyRequest{})
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if key.UserID != "user-1" || key.TenantID != "tenant-1" {
		t.Fatalf("CreateAPIKey() context = (%q, %q), want (user-1, tenant-1)", key.UserID, key.TenantID)
	}

	stored, err := dao.NewAPITokenDAO().GetByAPIToken(ctx, dao.DB, key.Token)
	if err != nil {
		t.Fatalf("GetByAPIToken() error = %v", err)
	}
	if stored == nil || stored.UserID == nil || *stored.UserID != "user-1" || stored.TenantID != "tenant-1" {
		t.Fatalf("stored API token context = %+v, want user-1 in tenant-1", stored)
	}
}

func setupAdminKeyTenant(t *testing.T) {
	t.Helper()
	db := setupServiceTestDB(t)
	if err := db.AutoMigrate(&entity.APIToken{}, &entity.UserTenant{}, &entity.AdminAPIKey{}); err != nil {
		t.Fatalf("failed to migrate API key tables: %v", err)
	}
	if err := db.Create(&entity.User{ID: "user-1", Email: "user-1@example.com", Nickname: "user-1"}).Error; err != nil {
		t.Fatalf("failed to create API key owner: %v", err)
	}
	if err := db.Create(&entity.Tenant{ID: "tenant-1", ParserIDs: ""}).Error; err != nil {
		t.Fatalf("failed to create Tenant: %v", err)
	}
	status := entity.TenantMembershipStatusJoined
	if err := db.Create(&entity.UserTenant{ID: "membership-1", UserID: "user-1", TenantID: "tenant-1", Role: "owner", Status: &status}).Error; err != nil {
		t.Fatalf("failed to create Tenant membership: %v", err)
	}
	userID := "user-1"
	if err := db.Create(&entity.APIToken{TenantID: "tenant-1", UserID: &userID, Token: "ragflow-admin-managed"}).Error; err != nil {
		t.Fatalf("failed to create admin token: %v", err)
	}
	if err := db.Create(&entity.AdminAPIKey{ID: "key-1", TenantID: "tenant-1", UserID: "user-1", Token: "ragflow-admin-managed", Name: "Open WebUI", Enabled: "1"}).Error; err != nil {
		t.Fatalf("failed to create admin key record: %v", err)
	}
	pushServiceDB(t, db)
}

func TestListAPIKeysHidesAdminManagedKeys(t *testing.T) {
	ctx := permission.WithIdentity(t.Context(), "user-1", "tenant-1")
	setupAdminKeyTenant(t)
	own, err := (&SystemService{}).CreateAPIKey(ctx, &CreateAPIKeyRequest{})
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	keys, err := (&SystemService{}).ListAPIKeys(ctx)
	if err != nil {
		t.Fatalf("ListAPIKeys() error = %v", err)
	}
	if len(keys) != 1 || keys[0].Token != own.Token {
		t.Fatalf("ListAPIKeys() = %+v, want only the user's own key", keys)
	}
}

func TestDeleteAPIKeyRefusesAdminManagedKey(t *testing.T) {
	ctx := permission.WithIdentity(t.Context(), "user-1", "tenant-1")
	setupAdminKeyTenant(t)

	if err := (&SystemService{}).DeleteAPIKey(ctx, "ragflow-admin-managed"); !errors.Is(err, ErrAPIKeyManagedByAdmin) {
		t.Fatalf("DeleteAPIKey() error = %v, want ErrAPIKeyManagedByAdmin", err)
	}
	stored, err := dao.NewAPITokenDAO().GetByAPIToken(ctx, dao.DB, "ragflow-admin-managed")
	if err != nil || stored == nil {
		t.Fatalf("admin-managed key was removed: %+v %v", stored, err)
	}
}
