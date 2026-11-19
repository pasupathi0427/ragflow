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

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"ragflow/internal/accessprofile"
	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"
	"ragflow/internal/globalmodel"

	"gorm.io/gorm"
)

const (
	testGlobalProviderID = "global-provider-1"
	testGlobalInstanceID = "global-instance-1"
	testGlobalModelID    = "global-model-1"
)

// setupGlobalModelTest seeds user-1's own OpenAI configuration (model-1) and a
// global "VLLM" provider with instance "shared" and chat model "qwen".
func setupGlobalModelTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupModelProviderServiceTestDB(t)
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.AccessProfile{},
		&entity.UserAccessProfile{},
		&entity.SystemSettings{},
		&entity.Knowledgebase{},
		&entity.Chat{},
		&entity.Memory{},
		&entity.UserCanvas{},
		&entity.SkillSpace{},
		&entity.SkillSearchConfig{},
		&entity.Search{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	useModelProviderServiceTestDB(t, db)
	seedModelProviderServiceScope(t, db)
	for _, row := range []interface{}{
		&entity.User{ID: "user-1", Email: "user-1@example.com", Nickname: "User", IsActive: "1"},
		&entity.TenantModelProvider{ID: testGlobalProviderID, UserID: entity.GlobalModelOwnerID, TenantID: entity.GlobalModelOwnerID, ProviderName: "VLLM"},
		&entity.TenantModelInstance{ID: testGlobalInstanceID, ProviderID: testGlobalProviderID, InstanceName: "shared", Status: "active", Extra: `{"base_url":"http://vllm:8000/v1"}`},
		&entity.TenantModel{ID: testGlobalModelID, ProviderID: testGlobalProviderID, InstanceID: testGlobalInstanceID, ModelName: "qwen", ModelType: int(entity.ModelTypeChat), Status: "active"},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed %T: %v", row, err)
		}
	}
	return db
}

func setTestConnectorKey(t *testing.T) {
	t.Helper()
	t.Setenv(common.EnvRAGFlowConnectorKey, base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
}

func restrictUserToGlobalModels(t *testing.T, modelIDs ...string) {
	t.Helper()
	profiles := accessprofile.DefaultService()
	profile, err := profiles.Create(context.Background(), accessprofile.ProfileInput{
		Name:         "Restricted",
		GlobalModels: &accessprofile.GlobalModelAccess{All: false, ModelIDs: modelIDs},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AssignUser(context.Background(), "user-1@example.com", profile.ID); err != nil {
		t.Fatal(err)
	}
}

func TestGlobalModelResolvesForEveryUserByIDAndName(t *testing.T) {
	setupGlobalModelTest(t)
	ctx := serviceTestContext(t, "user-1", "user-1")
	factory := NewModelFactory()
	for _, ref := range []string{testGlobalModelID, "qwen@shared@VLLM"} {
		identity, err := factory.resolveModelIdentity(ctx, entity.ModelTypeChat, ref)
		if err != nil {
			t.Fatalf("%s: %v", ref, err)
		}
		if identity.providerEntity.ID != testGlobalProviderID || identity.modelEntity.ID != testGlobalModelID {
			t.Fatalf("%s: resolved %s/%s", ref, identity.providerEntity.ID, identity.modelEntity.ID)
		}
	}
}

func TestOwnProviderStillResolves(t *testing.T) {
	setupGlobalModelTest(t)
	ctx := serviceTestContext(t, "user-1", "user-1")
	for _, ref := range []string{"model-1", "gpt-test@default@OpenAI"} {
		identity, err := NewModelFactory().resolveModelIdentity(ctx, entity.ModelTypeChat, ref)
		if err != nil || identity.providerEntity.ID != "provider-1" {
			t.Fatalf("%s: %v", ref, err)
		}
	}
}

func TestOwnProviderWinsOverSameNamedGlobal(t *testing.T) {
	db := setupGlobalModelTest(t)
	for _, row := range []interface{}{
		&entity.TenantModelProvider{ID: "global-openai", UserID: entity.GlobalModelOwnerID, TenantID: entity.GlobalModelOwnerID, ProviderName: "OpenAI"},
		&entity.TenantModelInstance{ID: "global-openai-default", ProviderID: "global-openai", InstanceName: "default", Status: "active", Extra: "{}"},
		&entity.TenantModel{ID: "global-gpt", ProviderID: "global-openai", InstanceID: "global-openai-default", ModelName: "gpt-test", ModelType: int(entity.ModelTypeChat), Status: "active"},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	identity, err := NewModelFactory().resolveModelIdentity(serviceTestContext(t, "user-1", "user-1"), entity.ModelTypeChat, "gpt-test@default@OpenAI")
	if err != nil || identity.providerEntity.ID != "provider-1" {
		t.Fatalf("name reference reached %v (%v), want the user's own provider", identity, err)
	}
}

func TestGlobalModelHardStops(t *testing.T) {
	for name, tt := range map[string]struct {
		table   string
		id      string
		updates map[string]interface{}
	}{
		"instance disabled": {"tenant_model_instance", testGlobalInstanceID, map[string]interface{}{"status": "inactive"}},
		"model disabled":    {"tenant_model", testGlobalModelID, map[string]interface{}{"status": "inactive"}},
	} {
		t.Run(name, func(t *testing.T) {
			db := setupGlobalModelTest(t)
			if err := db.Table(tt.table).Where("id = ?", tt.id).Updates(tt.updates).Error; err != nil {
				t.Fatal(err)
			}
			ctx := serviceTestContext(t, "user-1", "user-1")
			for _, ref := range []string{testGlobalModelID, "qwen@shared@VLLM"} {
				_, err := NewModelFactory().resolveModelIdentity(ctx, entity.ModelTypeChat, ref)
				if !errors.Is(err, errModelConfigUnavailable) {
					t.Fatalf("%s: error = %v, want unavailable", ref, err)
				}
			}
		})
	}
}

func TestGlobalModelRefusedOutsideProfile(t *testing.T) {
	db := setupGlobalModelTest(t)
	if err := db.Create(&entity.TenantModel{ID: "global-model-2", ProviderID: testGlobalProviderID, InstanceID: testGlobalInstanceID, ModelName: "llama", ModelType: int(entity.ModelTypeChat), Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	restrictUserToGlobalModels(t, "global-model-2")
	ctx := serviceTestContext(t, "user-1", "user-1")
	if _, err := NewModelFactory().resolveModelIdentity(ctx, entity.ModelTypeChat, testGlobalModelID); !errors.Is(err, globalmodel.ErrNotAllowed) {
		t.Fatalf("disallowed model: error = %v, want ErrNotAllowed", err)
	}
	if _, err := NewModelFactory().resolveModelIdentity(ctx, entity.ModelTypeChat, "global-model-2"); err != nil {
		t.Fatalf("allowed model: %v", err)
	}
	if _, err := NewModelFactory().resolveModelIdentity(ctx, entity.ModelTypeChat, "model-1"); err != nil {
		t.Fatalf("own model must not be limited by the profile: %v", err)
	}
}

func TestGlobalProvidersAreReadOnlyForUsers(t *testing.T) {
	setupGlobalModelTest(t)
	ctx := serviceTestContext(t, "user-1", "user-1")
	providers := NewModelProviderService()
	for _, operation := range []string{testGlobalProviderID, "VLLM"} {
		if _, err := providers.findProviderByIDOrNameInCurrentTenant(ctx, operation, true); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("%s: management lookup error = %v, want not found", operation, err)
		}
	}
	if code, err := providers.CreateProviderInstance(ctx, testGlobalProviderID, "mine", "sk", "", "", nil); err == nil || code == common.CodeSuccess {
		t.Fatal("user created an instance on a global provider")
	}
	if code, err := providers.DropProviderInstances(ctx, testGlobalProviderID, []string{testGlobalInstanceID}); err == nil || code == common.CodeSuccess {
		t.Fatal("user dropped a global instance")
	}
	var instance entity.TenantModelInstance
	if err := dao.DB.Where("id = ?", testGlobalInstanceID).Take(&instance).Error; err != nil {
		t.Fatalf("global instance changed: %v", err)
	}
	listed, _, err := providers.ListProvidersOfTenant(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, provider := range listed {
		if provider["id"] == testGlobalProviderID || provider["name"] == "VLLM" || provider["provider_name"] == "VLLM" {
			t.Fatalf("global provider listed among the user's own providers: %v", provider)
		}
	}
}

func TestGlobalAPIKeyEncryption(t *testing.T) {
	db := setupGlobalModelTest(t)
	admin := NewGlobalModelAdminService()
	input := GlobalInstanceInput{InstanceName: "keyed", APIKey: json.RawMessage(`"sk-global-secret-1234"`)}

	if _, err := admin.CreateInstance(context.Background(), testGlobalProviderID, input); !errors.Is(err, common.ErrModelAPIKeyEncryptionUnavailable) {
		t.Fatalf("without a connector key: error = %v, want refusal", err)
	}

	setTestConnectorKey(t)
	instance, err := admin.CreateInstance(context.Background(), testGlobalProviderID, input)
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := db.Raw("SELECT api_key FROM tenant_model_instance WHERE id = ?", instance.ID).Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if !common.IsEncryptedModelAPIKey(raw) || strings.Contains(raw, "secret") {
		t.Fatalf("key stored as %q, want encrypted", raw)
	}
	var loaded entity.TenantModelInstance
	if err := db.Where("id = ?", instance.ID).Take(&loaded).Error; err != nil || loaded.APIKey != "sk-global-secret-1234" {
		t.Fatalf("loaded key = %q (%v), want plaintext for the model call", loaded.APIKey, err)
	}

	views, err := admin.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(views)
	if strings.Contains(string(encoded), "secret") || !strings.Contains(string(encoded), "••••1234") {
		t.Fatalf("admin listing exposes the key or lacks the mask: %s", encoded)
	}

	// Saving the loaded (decrypted) instance back must keep the key encrypted.
	loaded.Status = "active"
	if err := db.Save(&loaded).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Raw("SELECT api_key FROM tenant_model_instance WHERE id = ?", instance.ID).Scan(&raw).Error; err != nil || !common.IsEncryptedModelAPIKey(raw) {
		t.Fatalf("re-saved key stored as %q (%v), want encrypted", raw, err)
	}
}

func TestGlobalModelDeleteIsBlockedWhileInUse(t *testing.T) {
	db := setupGlobalModelTest(t)
	admin := NewGlobalModelAdminService()
	embd := testGlobalModelID
	if err := db.Create(&entity.Knowledgebase{ID: "kb-1", TenantID: "user-1", Name: "kb", EmbdID: "qwen@shared@VLLM", TenantEmbdID: &embd}).Error; err != nil {
		t.Fatal(err)
	}
	usage, err := admin.DeleteModel(context.Background(), testGlobalModelID)
	if !errors.Is(err, ErrGlobalModelInUse) || usage == nil || usage.Datasets != 1 {
		t.Fatalf("delete in-use model: usage=%+v err=%v", usage, err)
	}
	if _, err := admin.DeleteProvider(context.Background(), testGlobalProviderID); !errors.Is(err, ErrGlobalModelInUse) {
		t.Fatalf("delete provider of in-use model: %v", err)
	}
	if err := db.Where("id = ?", "kb-1").Delete(&entity.Knowledgebase{}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := admin.DeleteProvider(context.Background(), testGlobalProviderID); err != nil {
		t.Fatalf("delete unused provider: %v", err)
	}
	var remaining int64
	db.Model(&entity.TenantModel{}).Where("provider_id = ?", testGlobalProviderID).Count(&remaining)
	if remaining != 0 {
		t.Fatalf("models of deleted provider remain: %d", remaining)
	}
}

func TestGlobalDefaultModelFallback(t *testing.T) {
	setupGlobalModelTest(t)
	admin := NewGlobalModelAdminService()
	ctx := serviceTestContext(t, "user-1", "user-1")

	if _, err := admin.SetDefaults(context.Background(), map[string]string{"embedding": testGlobalModelID}); !errors.Is(err, ErrGlobalModelInvalidInput) {
		t.Fatalf("chat model as embedding default: %v", err)
	}
	if _, err := admin.SetDefaults(context.Background(), map[string]string{"chat": "model-1"}); !errors.Is(err, ErrGlobalModelNotFound) {
		t.Fatalf("a user's own model as global default: %v", err)
	}
	if _, err := admin.SetDefaults(context.Background(), map[string]string{"chat": testGlobalModelID}); err != nil {
		t.Fatal(err)
	}

	ref, err := NewModelFactory().defaultModelReference(ctx, entity.ModelTypeChat)
	if err != nil || ref != testGlobalModelID {
		t.Fatalf("user without a chat default got %q (%v), want the global default", ref, err)
	}

	// A user's own default still wins.
	if err := dao.DB.Model(&entity.UserTenant{}).Where("id = ?", "user-tenant-1").Updates(map[string]interface{}{"llm_id": "gpt-test@default@OpenAI", "tenant_llm_id": "model-1"}).Error; err != nil {
		t.Fatal(err)
	}
	if ref, err := NewModelFactory().defaultModelReference(ctx, entity.ModelTypeChat); err != nil || ref != "model-1" {
		t.Fatalf("own default = %q (%v), want model-1", ref, err)
	}
}

func TestGlobalModelStatusSwitch(t *testing.T) {
	setupGlobalModelTest(t)
	admin := NewGlobalModelAdminService()
	ctx := serviceTestContext(t, "user-1", "user-1")
	if err := admin.SetModelStatus(context.Background(), testGlobalModelID, "inactive"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewModelFactory().resolveModelIdentity(ctx, entity.ModelTypeChat, testGlobalModelID); !errors.Is(err, errModelConfigUnavailable) {
		t.Fatalf("disabled model resolved: %v", err)
	}
	if err := admin.SetModelStatus(context.Background(), testGlobalModelID, "active"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewModelFactory().resolveModelIdentity(ctx, entity.ModelTypeChat, testGlobalModelID); err != nil {
		t.Fatalf("re-enabled model: %v", err)
	}
	if err := admin.SetModelStatus(context.Background(), "model-1", "inactive"); !errors.Is(err, ErrGlobalModelNotFound) {
		t.Fatalf("admin changed a user's own model: %v", err)
	}
}

func TestGlobalModelUsageIgnoresTenantsOwnSameNamedProvider(t *testing.T) {
	db := setupGlobalModelTest(t)
	admin := NewGlobalModelAdminService()
	// tenant-2 has its own VLLM provider, so its "qwen@shared@VLLM" chat is not
	// a use of the global model.
	for _, row := range []interface{}{
		&entity.TenantModelProvider{ID: "own-vllm", TenantID: "tenant-2", UserID: "user-2", ProviderName: "VLLM"},
		&entity.Chat{ID: "chat-own", TenantID: "tenant-2", LLMID: "qwen@shared@VLLM", LLMSetting: entity.JSONMap{}, PromptConfig: entity.JSONMap{}, DoRefer: "1", KBIDs: entity.JSONSlice{}},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if _, err := admin.DeleteModel(context.Background(), testGlobalModelID); err != nil {
		t.Fatalf("delete blocked by another tenant's own model: %v", err)
	}
}

func TestGlobalModelUsageCountsNameBasedUserDefaults(t *testing.T) {
	db := setupGlobalModelTest(t)
	if err := db.Model(&entity.UserTenant{}).Where("id = ?", "user-tenant-1").Update("llm_id", "qwen@shared@VLLM").Error; err != nil {
		t.Fatal(err)
	}
	usage, err := NewGlobalModelAdminService().DeleteModel(context.Background(), testGlobalModelID)
	if !errors.Is(err, ErrGlobalModelInUse) || usage == nil || usage.UserDefaults != 1 {
		t.Fatalf("name-based default not counted: usage=%+v err=%v", usage, err)
	}
}

func TestGlobalDefaultMustBeEnabled(t *testing.T) {
	db := setupGlobalModelTest(t)
	if err := db.Model(&entity.TenantModelInstance{}).Where("id = ?", testGlobalInstanceID).Update("status", "inactive").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewGlobalModelAdminService().SetDefaults(context.Background(), map[string]string{"chat": testGlobalModelID}); !errors.Is(err, ErrGlobalModelInvalidInput) {
		t.Fatalf("disabled model accepted as default: %v", err)
	}
}

func TestAppendGlobalModelsListsWhatTheUserMayCall(t *testing.T) {
	db := setupGlobalModelTest(t)
	if err := db.Create(&entity.TenantModel{ID: "global-embed", ProviderID: testGlobalProviderID, InstanceID: testGlobalInstanceID, ModelName: "bge", ModelType: int(entity.ModelTypeEmbedding), Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	ctx := serviceTestContext(t, "user-1", "user-1")
	providers := NewModelProviderService()

	listed, err := providers.AppendGlobalModels(ctx, nil, "")
	if err != nil || len(listed) != 2 {
		t.Fatalf("all types: %d models (%v)", len(listed), err)
	}
	for _, model := range listed {
		if model["is_global"] != true || model["tenant_id"] != entity.GlobalModelOwnerID {
			t.Fatalf("global model not marked: %v", model)
		}
		for _, field := range []string{"api_key", "base_url", "extra"} {
			if _, ok := model[field]; ok {
				t.Fatalf("listing exposes %s: %v", field, model)
			}
		}
	}

	chatOnly, err := providers.AppendGlobalModels(ctx, nil, "chat")
	if err != nil || len(chatOnly) != 1 || chatOnly[0]["name"] != "qwen" {
		t.Fatalf("chat filter: %v (%v)", chatOnly, err)
	}

	if err := db.Model(&entity.TenantModel{}).Where("id = ?", "global-embed").Update("status", "inactive").Error; err != nil {
		t.Fatal(err)
	}
	restrictUserToGlobalModels(t)
	hidden, err := providers.AppendGlobalModels(ctx, []map[string]interface{}{{"model_id": "own"}}, "")
	if err != nil || len(hidden) != 1 || hidden[0]["model_id"] != "own" {
		t.Fatalf("disabled and disallowed models listed: %v (%v)", hidden, err)
	}
}

func TestDefaultModelListFallsBackToGlobalDefault(t *testing.T) {
	setupGlobalModelTest(t)
	if _, err := NewGlobalModelAdminService().SetDefaults(context.Background(), map[string]string{"chat": testGlobalModelID}); err != nil {
		t.Fatal(err)
	}
	items, err := NewUserTenantService().ListDefaultModels(serviceTestContext(t, "user-1", "user-1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ModelID != testGlobalModelID || items[0].ModelType != "chat" {
		t.Fatalf("defaults = %+v, want the global chat default", items)
	}
}
