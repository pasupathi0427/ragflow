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

package globalmodel

import (
	"context"
	"errors"
	"testing"

	"ragflow/internal/accessprofile"
	"ragflow/internal/dao"
	"ragflow/internal/entity"
	"ragflow/internal/ingestion/testutil"
	"ragflow/internal/permission"

	"gorm.io/gorm"
)

func setupGlobalModelPackageTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.SetupTestDB(t,
		&entity.User{}, &entity.UserTenant{}, &entity.AccessProfile{}, &entity.UserAccessProfile{},
		&entity.TenantModelProvider{}, &entity.TenantModelInstance{}, &entity.TenantModel{}, &entity.SystemSettings{},
	)
	t.Cleanup(testutil.ReplaceDBForTest(t, db))
	status := "1"
	for _, row := range []interface{}{
		&entity.User{ID: "owner", Email: "owner@example.com", Nickname: "Owner", IsActive: "1"},
		&entity.User{ID: "member", Email: "member@example.com", Nickname: "Member", IsActive: "1"},
		&entity.UserTenant{ID: "ut-owner", UserID: "owner", TenantID: "tenant-1", Role: "owner", InvitedBy: "owner", Status: &status},
		&entity.TenantModelProvider{ID: "own-provider", TenantID: "tenant-1", UserID: "owner", ProviderName: "VLLM"},
		&entity.TenantModelProvider{ID: "foreign-provider", TenantID: "tenant-2", UserID: "other", ProviderName: "Ollama"},
		&entity.TenantModelProvider{ID: "global-provider", TenantID: entity.GlobalModelOwnerID, UserID: entity.GlobalModelOwnerID, ProviderName: "VLLM"},
		&entity.TenantModelProvider{ID: "global-only", TenantID: entity.GlobalModelOwnerID, UserID: entity.GlobalModelOwnerID, ProviderName: "OpenAI"},
		&entity.TenantModelInstance{ID: "global-instance", ProviderID: "global-only", InstanceName: "shared", Status: "active", Extra: "{}"},
		&entity.TenantModel{ID: "global-model", ProviderID: "global-only", InstanceID: "global-instance", ModelName: "gpt", ModelType: int(entity.ModelTypeChat), Status: "active"},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed %T: %v", row, err)
		}
	}
	return db
}

func globalFixtures(t *testing.T) (*entity.TenantModelProvider, *entity.TenantModelInstance) {
	t.Helper()
	var provider entity.TenantModelProvider
	var instance entity.TenantModelInstance
	if err := dao.DB.Where("id = ?", "global-only").Take(&provider).Error; err != nil {
		t.Fatal(err)
	}
	if err := dao.DB.Where("id = ?", "global-instance").Take(&instance).Error; err != nil {
		t.Fatal(err)
	}
	return &provider, &instance
}

func TestProviderByNamePrefersTheTenant(t *testing.T) {
	setupGlobalModelPackageTest(t)
	ctx := context.Background()
	own, err := ProviderByName(ctx, dao.DB, "tenant-1", "owner", "VLLM")
	if err != nil || own.ID != "own-provider" {
		t.Fatalf("VLLM = %+v (%v), want the tenant's own provider", own, err)
	}
	global, err := ProviderByName(ctx, dao.DB, "tenant-1", "owner", "OpenAI")
	if err != nil || global.ID != "global-only" {
		t.Fatalf("OpenAI = %+v (%v), want the global provider", global, err)
	}
	if _, err := ProviderByName(ctx, dao.DB, "tenant-1", "owner", "Ollama"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("another tenant's provider was reachable: %v", err)
	}
}

func TestCheckUse(t *testing.T) {
	db := setupGlobalModelPackageTest(t)
	provider, instance := globalFixtures(t)
	ctx := permission.WithIdentity(context.Background(), "member", "tenant-1")

	var own, foreign entity.TenantModelProvider
	db.Where("id = ?", "own-provider").Take(&own)
	db.Where("id = ?", "foreign-provider").Take(&foreign)
	if err := CheckUse(ctx, db, "tenant-1", &own, nil, "anything"); err != nil {
		t.Fatalf("own provider: %v", err)
	}
	if err := CheckUse(ctx, db, "tenant-1", &foreign, nil, "anything"); !errors.Is(err, ErrNotAccessible) {
		t.Fatalf("foreign provider: %v", err)
	}
	if err := CheckUse(ctx, db, "tenant-1", provider, instance, "gpt"); err != nil {
		t.Fatalf("enabled global model: %v", err)
	}
	if err := CheckUse(ctx, db, "tenant-1", provider, instance, "not-enabled"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("model the admin never enabled: %v", err)
	}

	db.Model(&entity.TenantModel{}).Where("id = ?", "global-model").Update("status", "inactive")
	if err := CheckUse(ctx, db, "tenant-1", provider, instance, "gpt"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled model: %v", err)
	}
	db.Model(&entity.TenantModel{}).Where("id = ?", "global-model").Update("status", "active")

	stopped := *instance
	stopped.Status = "inactive"
	if err := CheckUse(ctx, db, "tenant-1", provider, &stopped, "gpt"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled instance: %v", err)
	}
}

func TestCheckUseFollowsTheCallersProfile(t *testing.T) {
	db := setupGlobalModelPackageTest(t)
	provider, instance := globalFixtures(t)
	profiles := accessprofile.DefaultService()
	restricted, err := profiles.Create(context.Background(), accessprofile.ProfileInput{
		Name:         "No global models",
		GlobalModels: &accessprofile.GlobalModelAccess{All: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AssignUser(context.Background(), "member@example.com", restricted.ID); err != nil {
		t.Fatal(err)
	}

	member := permission.WithIdentity(context.Background(), "member", "tenant-1")
	if err := CheckUse(member, db, "tenant-1", provider, instance, "gpt"); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("restricted member: %v", err)
	}
	owner := permission.WithIdentity(context.Background(), "owner", "tenant-1")
	if err := CheckUse(owner, db, "tenant-1", provider, instance, "gpt"); err != nil {
		t.Fatalf("unrestricted owner in the same tenant: %v", err)
	}
	// Background work without a user runs as the tenant owner.
	if err := CheckUse(context.Background(), db, "tenant-1", provider, instance, "gpt"); err != nil {
		t.Fatalf("background run as owner: %v", err)
	}
}

func TestDefaultModelID(t *testing.T) {
	db := setupGlobalModelPackageTest(t)
	if got := DefaultModelID(context.Background(), db, entity.ModelTypeChat); got != "" {
		t.Fatalf("no defaults set, got %q", got)
	}
	if err := dao.NewSystemSettingsDAO().SaveOrCreate(context.Background(), db, DefaultsSetting, `{"chat":"global-model"}`, "admin", "json"); err != nil {
		t.Fatal(err)
	}
	if got := DefaultModelID(context.Background(), db, entity.ModelTypeChat); got != "global-model" {
		t.Fatalf("chat default = %q", got)
	}
	if got := DefaultModelID(context.Background(), db, entity.ModelTypeEmbedding); got != "" {
		t.Fatalf("embedding default = %q, want none", got)
	}
}
