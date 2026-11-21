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

package accessprofile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"

	"github.com/gin-gonic/gin"
)

func mustCreateAPIKey(t *testing.T, svc *Service, email, name string) *AdminAPIKey {
	t.Helper()
	key, err := svc.CreateAPIKey(context.Background(), email, name, "admin@example.com")
	if err != nil {
		t.Fatalf("create API key: %v", err)
	}
	return key
}

func TestCreateAPIKeyStoresTokenInUserTenant(t *testing.T) {
	svc := setupAccessProfileTest(t)
	key := mustCreateAPIKey(t, svc, "a@example.com", "Open WebUI")

	if !strings.HasPrefix(key.Token, "ragflow-") || !key.Enabled || key.CreatedBy != "admin@example.com" {
		t.Fatalf("unexpected key: %+v", key)
	}
	if strings.Contains(key.MaskedToken, key.Token[12:len(key.Token)-4]) {
		t.Fatalf("masked token %q exposes the key", key.MaskedToken)
	}
	token, err := dao.NewAPITokenDAO().GetByAPIToken(context.Background(), dao.DB, key.Token)
	if err != nil || token == nil {
		t.Fatalf("api_token row missing: %v", err)
	}
	if token.TenantID != "user-a" || token.UserID == nil || *token.UserID != "user-a" {
		t.Fatalf("token stored for wrong owner: %+v", token)
	}
	if token.Beta != nil && *token.Beta != "" {
		t.Fatalf("admin key must not carry an embed token, got %q", *token.Beta)
	}
}

func TestListAPIKeysNeverReturnsTheFullKey(t *testing.T) {
	svc := setupAccessProfileTest(t)
	created := mustCreateAPIKey(t, svc, "a@example.com", "Open WebUI")
	mustCreateAPIKey(t, svc, "b@example.com", "Other user")

	keys, err := svc.ListAPIKeys(context.Background(), "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].ID != created.ID || keys[0].Token != "" || keys[0].MaskedToken == "" {
		t.Fatalf("unexpected keys: %+v", keys)
	}
}

func TestCreateAPIKeyValidation(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	if _, err := svc.CreateAPIKey(ctx, "a@example.com", "   ", "admin"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("blank name accepted: %v", err)
	}
	if _, err := svc.CreateAPIKey(ctx, "a@example.com", strings.Repeat("x", maxAPIKeyNameLength+1), "admin"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("long name accepted: %v", err)
	}
	if _, err := svc.CreateAPIKey(ctx, "nobody@example.com", "Key", "admin"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unknown user accepted: %v", err)
	}
	isSuperuser := true
	if err := dao.DB.Create(&entity.User{ID: "root", Email: "root@example.com", Nickname: "Root", IsActive: "1", IsSuperuser: &isSuperuser}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateAPIKey(ctx, "root@example.com", "Key", "admin"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("superuser key accepted: %v", err)
	}
}

func TestCheckAPIKeyAccessForAdminKeys(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	userA := &entity.User{ID: "user-a", IsActive: "1"}
	key := mustCreateAPIKey(t, svc, "a@example.com", "Open WebUI")

	if err := CheckAPIKeyAccess(ctx, userA, key.Token); err != nil {
		t.Fatalf("enabled admin key rejected: %v", err)
	}

	if _, err := svc.SetAPIKeyEnabled(ctx, "a@example.com", key.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := CheckAPIKeyAccess(ctx, userA, key.Token); !errors.Is(err, ErrAPIKeyDisabled) {
		t.Fatalf("disabled admin key accepted: %v", err)
	}
	if err := CheckAPIKeyAccess(ctx, userA, "ragflow-own-key"); err != nil {
		t.Fatalf("disabling an admin key blocked the user's own key: %v", err)
	}

	if _, err := svc.SetAPIKeyEnabled(ctx, "a@example.com", key.ID, true); err != nil {
		t.Fatal(err)
	}
	restricted := mustCreate(t, svc, ProfileInput{Name: "No API", Features: map[string]bool{"api": false}})
	if _, err := svc.AssignUser(ctx, "a@example.com", restricted.ID); err != nil {
		t.Fatal(err)
	}
	if err := CheckAPIKeyAccess(ctx, userA, key.Token); !errors.Is(err, ErrAPIAccessDisabled) {
		t.Fatalf("admin key accepted while the profile turns API off: %v", err)
	}
}

func TestCheckAPIKeyAccessRejectsInactiveUser(t *testing.T) {
	setupAccessProfileTest(t)
	inactive := &entity.User{ID: "user-a", IsActive: "0"}
	if err := CheckAPIKeyAccess(context.Background(), inactive, "ragflow-own-key"); !errors.Is(err, ErrAPIAccountDisabled) {
		t.Fatalf("inactive user's key accepted: %v", err)
	}
}

func TestDeleteAPIKeyRemovesToken(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	key := mustCreateAPIKey(t, svc, "a@example.com", "Open WebUI")

	if err := svc.DeleteAPIKey(ctx, "b@example.com", key.ID); !errors.Is(err, ErrAPIKeyNotFound) {
		t.Fatalf("deleted another user's key: %v", err)
	}
	if err := svc.DeleteAPIKey(ctx, "a@example.com", key.ID); err != nil {
		t.Fatal(err)
	}
	token, err := dao.NewAPITokenDAO().GetByAPIToken(ctx, dao.DB, key.Token)
	if err != nil || token != nil {
		t.Fatalf("api_token row still present: %+v %v", token, err)
	}
	if keys, _ := svc.ListAPIKeys(ctx, "a@example.com"); len(keys) != 0 {
		t.Fatalf("admin record still present: %+v", keys)
	}
}

func TestAdminAPIKeyRoutes(t *testing.T) {
	setupAccessProfileTest(t)
	engine := gin.New()
	group := engine.Group("/api/v1/admin")
	group.Use(func(c *gin.Context) { c.Set("email", "admin@example.com") })
	RegisterAdminRoutes(group)

	_, response := serve(t, engine, http.MethodPost, "/api/v1/admin/users/a@example.com/api-keys", map[string]string{"name": "Open WebUI"})
	if response.Code != int(common.CodeSuccess) {
		t.Fatalf("create failed: %+v", response)
	}
	var created AdminAPIKey
	if err := json.Unmarshal(response.Data, &created); err != nil {
		t.Fatal(err)
	}
	if created.Token == "" || created.CreatedBy != "admin@example.com" {
		t.Fatalf("unexpected create response: %+v", created)
	}

	_, response = serve(t, engine, http.MethodGet, "/api/v1/admin/users/a@example.com/api-keys", nil)
	if response.Code != int(common.CodeSuccess) || strings.Contains(string(response.Data), created.Token) {
		t.Fatalf("list failed or exposed the key: %+v", response)
	}

	_, response = serve(t, engine, http.MethodPut, "/api/v1/admin/users/a@example.com/api-keys/"+created.ID, map[string]any{})
	if response.Code != int(common.CodeBadRequest) {
		t.Fatalf("missing enabled accepted: %+v", response)
	}
	_, response = serve(t, engine, http.MethodPut, "/api/v1/admin/users/a@example.com/api-keys/"+created.ID, map[string]bool{"enabled": false})
	if response.Code != int(common.CodeSuccess) || !strings.Contains(string(response.Data), `"enabled":false`) {
		t.Fatalf("disable failed: %+v", response)
	}

	_, response = serve(t, engine, http.MethodDelete, "/api/v1/admin/users/a@example.com/api-keys/missing", nil)
	if response.Code != int(common.CodeNotFound) {
		t.Fatalf("missing key delete: %+v", response)
	}
	_, response = serve(t, engine, http.MethodDelete, "/api/v1/admin/users/a@example.com/api-keys/"+created.ID, nil)
	if response.Code != int(common.CodeSuccess) {
		t.Fatalf("delete failed: %+v", response)
	}
}
