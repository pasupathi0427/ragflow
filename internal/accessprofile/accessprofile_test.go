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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"
	"ragflow/internal/ingestion/testutil"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupAccessProfileTest gives each test its own database and a fresh service
// so cached lookups never leak between tests.
func setupAccessProfileTest(t *testing.T) *Service {
	t.Helper()
	db := testutil.SetupTestDB(t, &entity.User{}, &entity.AccessProfile{}, &entity.UserAccessProfile{})
	t.Cleanup(testutil.ReplaceDBForTest(t, db))
	previous := defaultService
	defaultService = NewService()
	t.Cleanup(func() { defaultService = previous })
	for _, user := range []entity.User{
		{ID: "user-a", Email: "a@example.com", Nickname: "A", IsActive: "1"},
		{ID: "user-b", Email: "b@example.com", Nickname: "B", IsActive: "1"},
	} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	return defaultService
}

func mustCreate(t *testing.T, svc *Service, input ProfileInput) *Profile {
	t.Helper()
	profile, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	return profile
}

func mustEnabled(t *testing.T, svc *Service, userID string, feature Feature) bool {
	t.Helper()
	enabled, err := svc.IsEnabled(context.Background(), userID, feature)
	if err != nil {
		t.Fatalf("IsEnabled: %v", err)
	}
	return enabled
}

func TestEverythingEnabledBeforeAnyProfileExists(t *testing.T) {
	svc := setupAccessProfileTest(t)
	for _, feature := range AllFeatures {
		if !mustEnabled(t, svc, "user-a", feature) {
			t.Fatalf("feature %s disabled with no profiles configured", feature)
		}
	}
}

func TestDefaultProfileAppliesToUnassignedUsers(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	if _, err := svc.Update(ctx, DefaultProfileID, ProfileInput{Name: "Default", Features: map[string]bool{"mcp": false}}); err != nil {
		t.Fatalf("update default: %v", err)
	}
	if mustEnabled(t, svc, "user-a", FeatureMCP) {
		t.Fatal("mcp enabled although the default profile disables it")
	}
	if !mustEnabled(t, svc, "user-a", FeatureTeam) {
		t.Fatal("team disabled although the default profile does not mention it")
	}
}

func TestAssignedProfileOverridesDefault(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	restricted := mustCreate(t, svc, ProfileInput{Name: "Interns", Features: map[string]bool{"api": false, "data_sources": false}})
	if _, err := svc.AssignUser(ctx, "a@example.com", restricted.ID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if mustEnabled(t, svc, "user-a", FeatureAPI) || mustEnabled(t, svc, "user-a", FeatureDataSources) {
		t.Fatal("assigned profile switches were not applied")
	}
	if !mustEnabled(t, svc, "user-b", FeatureAPI) {
		t.Fatal("unassigned user lost API access")
	}
	features, err := svc.FeaturesForUser(ctx, "user-a")
	if err != nil {
		t.Fatal(err)
	}
	if features.ProfileID != restricted.ID || features.ProfileName != "Interns" {
		t.Fatalf("unexpected profile in features: %+v", features)
	}
}

func TestAssigningDefaultRemovesAssignment(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	restricted := mustCreate(t, svc, ProfileInput{Name: "No API", Features: map[string]bool{"api": false}})
	if _, err := svc.AssignUser(ctx, "a@example.com", restricted.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AssignUser(ctx, "a@example.com", DefaultProfileID); err != nil {
		t.Fatal(err)
	}
	assignments, err := svc.ListAssignments(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 0 {
		t.Fatalf("assignment kept after assigning default: %+v", assignments)
	}
	if !mustEnabled(t, svc, "user-a", FeatureAPI) {
		t.Fatal("user still restricted after moving to default")
	}
}

func TestReassignmentReplacesEarlierProfile(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	first := mustCreate(t, svc, ProfileInput{Name: "First", Features: map[string]bool{"mcp": false}})
	second := mustCreate(t, svc, ProfileInput{Name: "Second", Features: map[string]bool{"team": false}})
	if _, err := svc.AssignUser(ctx, "a@example.com", first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AssignUser(ctx, "a@example.com", second.ID); err != nil {
		t.Fatal(err)
	}
	assignments, err := svc.ListAssignments(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 1 || assignments[0].ProfileID != second.ID || assignments[0].Email != "a@example.com" {
		t.Fatalf("unexpected assignments: %+v", assignments)
	}
	if !mustEnabled(t, svc, "user-a", FeatureMCP) || mustEnabled(t, svc, "user-a", FeatureTeam) {
		t.Fatal("second profile did not replace the first")
	}
}

func TestDeletingProfileMovesMembersToDefault(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	restricted := mustCreate(t, svc, ProfileInput{Name: "Temp", Features: map[string]bool{"chat_channels": false}})
	if _, err := svc.AssignUser(ctx, "a@example.com", restricted.ID); err != nil {
		t.Fatal(err)
	}
	profiles, err := svc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 || !profiles[0].IsDefault || profiles[1].MemberCount != 1 {
		t.Fatalf("unexpected list before delete: %+v", profiles)
	}
	if err := svc.Delete(ctx, restricted.ID); err != nil {
		t.Fatal(err)
	}
	if !mustEnabled(t, svc, "user-a", FeatureChatChannels) {
		t.Fatal("member of a deleted profile is still restricted")
	}
	var remaining int64
	dao.DB.Model(&entity.UserAccessProfile{}).Count(&remaining)
	if remaining != 0 {
		t.Fatalf("assignments of deleted profile remain: %d", remaining)
	}
}

func TestDefaultProfileIsProtected(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	if err := svc.Delete(ctx, DefaultProfileID); !errors.Is(err, ErrDefaultProfileReadOnly) {
		t.Fatalf("delete default: got %v", err)
	}
	if _, err := svc.Update(ctx, DefaultProfileID, ProfileInput{Name: "Renamed"}); !errors.Is(err, ErrDefaultProfileReadOnly) {
		t.Fatalf("rename default: got %v", err)
	}
}

func TestProfileInputValidation(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	mustCreate(t, svc, ProfileInput{Name: "Engineers"})
	for name, tt := range map[string]struct {
		input ProfileInput
		want  error
	}{
		"empty name":      {ProfileInput{Name: "   "}, ErrInvalidInput},
		"unknown feature": {ProfileInput{Name: "X", Features: map[string]bool{"billing": false}}, ErrInvalidInput},
		"duplicate name":  {ProfileInput{Name: " Engineers "}, ErrProfileNameTaken},
		"default name":    {ProfileInput{Name: "Default"}, ErrProfileNameTaken},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Create(ctx, tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
	if _, err := svc.AssignUser(ctx, "missing@example.com", DefaultProfileID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("assign unknown user: got %v", err)
	}
	if _, err := svc.AssignUser(ctx, "a@example.com", "no-such-profile"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("assign unknown profile: got %v", err)
	}
}

func TestAdminChangesInvalidateCache(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	if !mustEnabled(t, svc, "user-a", FeatureMCP) {
		t.Fatal("expected mcp enabled")
	}
	if _, err := svc.Update(ctx, DefaultProfileID, ProfileInput{Name: "Default", Features: map[string]bool{"mcp": false}}); err != nil {
		t.Fatal(err)
	}
	if mustEnabled(t, svc, "user-a", FeatureMCP) {
		t.Fatal("cached value survived an admin change")
	}
}

func TestCachedValueExpires(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	mustEnabled(t, svc, "user-a", FeatureMCP)
	// A change made by another process does not invalidate this cache.
	restricted := mustCreate(t, NewService(), ProfileInput{Name: "No MCP", Features: map[string]bool{"mcp": false}})
	if err := dao.NewAccessProfileDAO().SetAssignment(ctx, dao.DB, "user-a", restricted.ID); err != nil {
		t.Fatal(err)
	}
	if !mustEnabled(t, svc, "user-a", FeatureMCP) {
		t.Fatal("expected cached value before expiry")
	}
	svc.mu.Lock()
	entry := svc.cache["user-a"]
	entry.expires = time.Now().Add(-time.Second)
	svc.cache["user-a"] = entry
	svc.mu.Unlock()
	if mustEnabled(t, svc, "user-a", FeatureMCP) {
		t.Fatal("expired cache entry was reused")
	}
}

type testResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func serve(t *testing.T, engine *gin.Engine, method, path string, body any) (*httptest.ResponseRecorder, testResponse) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	var response testResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode %s: %v", recorder.Body.String(), err)
	}
	return recorder, response
}

// userEngine mimics the API auth middleware by setting user_id.
func userEngine(userID string) *gin.Engine {
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	})
	engine.GET("/mcp", RequireFeature(FeatureMCP), func(c *gin.Context) {
		common.RespondSuccess(c, nil, "")
	})
	engine.GET("/features", MyFeatures)
	return engine
}

func TestRequireFeature(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	restricted := mustCreate(t, svc, ProfileInput{Name: "No MCP", Features: map[string]bool{"mcp": false}})
	if _, err := svc.AssignUser(ctx, "a@example.com", restricted.ID); err != nil {
		t.Fatal(err)
	}

	recorder, response := serve(t, userEngine("user-a"), http.MethodGet, "/mcp", nil)
	if recorder.Code != http.StatusForbidden || response.Code != int(common.CodeForbidden) || response.Message != MessageFeatureDisabled {
		t.Fatalf("restricted user: %d %+v", recorder.Code, response)
	}
	recorder, response = serve(t, userEngine("user-b"), http.MethodGet, "/mcp", nil)
	if recorder.Code != http.StatusOK || response.Code != int(common.CodeSuccess) {
		t.Fatalf("allowed user: %d %+v", recorder.Code, response)
	}
	recorder, _ = serve(t, userEngine(""), http.MethodGet, "/mcp", nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous request: %d", recorder.Code)
	}
}

func TestMyFeatures(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	restricted := mustCreate(t, svc, ProfileInput{Name: "No Team", Features: map[string]bool{"team": false}})
	if _, err := svc.AssignUser(ctx, "a@example.com", restricted.ID); err != nil {
		t.Fatal(err)
	}
	_, response := serve(t, userEngine("user-a"), http.MethodGet, "/features", nil)
	var features UserFeatures
	if err := json.Unmarshal(response.Data, &features); err != nil {
		t.Fatal(err)
	}
	if features.ProfileName != "No Team" || features.Features["team"] || !features.Features["mcp"] || len(features.Features) != len(AllFeatures) {
		t.Fatalf("unexpected features: %+v", features)
	}
}

func TestCheckAPIKeyAccess(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	if err := CheckAPIKeyAccess(ctx, "user-a"); err != nil {
		t.Fatalf("API key rejected for unrestricted user: %v", err)
	}
	restricted := mustCreate(t, svc, ProfileInput{Name: "No API", Features: map[string]bool{"api": false}})
	if _, err := svc.AssignUser(ctx, "a@example.com", restricted.ID); err != nil {
		t.Fatal(err)
	}
	if err := CheckAPIKeyAccess(ctx, "user-a"); !errors.Is(err, ErrAPIAccessDisabled) {
		t.Fatalf("API key accepted for restricted user: %v", err)
	}
}

func TestAdminRoutes(t *testing.T) {
	setupAccessProfileTest(t)
	engine := gin.New()
	RegisterAdminRoutes(engine.Group("/api/v1/admin"))

	_, response := serve(t, engine, http.MethodPost, "/api/v1/admin/access-profiles", ProfileInput{Name: "Analysts", Features: map[string]bool{"mcp": false}})
	if response.Code != int(common.CodeSuccess) {
		t.Fatalf("create: %+v", response)
	}
	var created Profile
	if err := json.Unmarshal(response.Data, &created); err != nil {
		t.Fatal(err)
	}

	_, response = serve(t, engine, http.MethodPost, "/api/v1/admin/access-profiles", ProfileInput{Name: "Analysts"})
	if response.Code != int(common.CodeConflict) {
		t.Fatalf("duplicate create: %+v", response)
	}

	_, response = serve(t, engine, http.MethodPut, "/api/v1/admin/users/a@example.com/access-profile", map[string]string{"profile_id": created.ID})
	if response.Code != int(common.CodeSuccess) {
		t.Fatalf("assign: %+v", response)
	}

	_, response = serve(t, engine, http.MethodGet, "/api/v1/admin/access-profile-assignments", nil)
	var assignments []Assignment
	if err := json.Unmarshal(response.Data, &assignments); err != nil || len(assignments) != 1 || assignments[0].ProfileID != created.ID {
		t.Fatalf("assignments: %v %s", err, response.Data)
	}

	_, response = serve(t, engine, http.MethodGet, "/api/v1/admin/access-profiles", nil)
	var profiles []Profile
	if err := json.Unmarshal(response.Data, &profiles); err != nil || len(profiles) != 2 || profiles[1].MemberCount != 1 {
		t.Fatalf("list: %v %s", err, response.Data)
	}

	_, response = serve(t, engine, http.MethodDelete, "/api/v1/admin/access-profiles/"+DefaultProfileID, nil)
	if response.Code != int(common.CodeForbidden) {
		t.Fatalf("delete default: %+v", response)
	}

	_, response = serve(t, engine, http.MethodDelete, "/api/v1/admin/access-profiles/"+created.ID, nil)
	if response.Code != int(common.CodeSuccess) {
		t.Fatalf("delete: %+v", response)
	}

	_, response = serve(t, engine, http.MethodGet, "/api/v1/admin/access-profiles/"+created.ID, nil)
	if response.Code != int(common.CodeNotFound) {
		t.Fatalf("get deleted: %+v", response)
	}
}

func TestProfileNamesAreUniqueIgnoringCase(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	mustCreate(t, svc, ProfileInput{Name: "Sales"})
	if _, err := svc.Create(ctx, ProfileInput{Name: "sales"}); !errors.Is(err, ErrProfileNameTaken) {
		t.Fatalf("create with different case: got %v", err)
	}
	other := mustCreate(t, svc, ProfileInput{Name: "Support"})
	if _, err := svc.Update(ctx, other.ID, ProfileInput{Name: "SALES"}); !errors.Is(err, ErrProfileNameTaken) {
		t.Fatalf("rename to different case: got %v", err)
	}
	if _, err := svc.Update(ctx, other.ID, ProfileInput{Name: "SUPPORT"}); err != nil {
		t.Fatalf("recasing own name: %v", err)
	}
}

func TestSuperusersCannotBeAssigned(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	isSuperuser := true
	admin := entity.User{ID: "admin", Email: "admin@example.com", Nickname: "Admin", IsActive: "1", IsSuperuser: &isSuperuser}
	if err := dao.DB.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	restricted := mustCreate(t, svc, ProfileInput{Name: "Restricted"})
	if _, err := svc.AssignUser(ctx, "admin@example.com", restricted.ID); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("assign superuser: got %v", err)
	}
}

func TestDeletedUsersAreNotCountedAsMembers(t *testing.T) {
	svc := setupAccessProfileTest(t)
	ctx := context.Background()
	restricted := mustCreate(t, svc, ProfileInput{Name: "Restricted"})
	for _, email := range []string{"a@example.com", "b@example.com"} {
		if _, err := svc.AssignUser(ctx, email, restricted.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := dao.DB.Where("id = ?", "user-b").Delete(&entity.User{}).Error; err != nil {
		t.Fatal(err)
	}
	profile, err := svc.Get(ctx, restricted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if profile.MemberCount != 1 {
		t.Fatalf("member count includes deleted user: %d", profile.MemberCount)
	}
}

func TestLookupStartedBeforeInvalidationIsNotCached(t *testing.T) {
	svc := setupAccessProfileTest(t)
	now := time.Now()
	svc.mu.Lock()
	startedAt := svc.generation
	svc.mu.Unlock()

	// An admin change lands while the lookup is in flight.
	svc.invalidateCache()
	svc.cacheIfCurrent("user-a", UserFeatures{ProfileID: "stale"}, startedAt, now)
	svc.mu.Lock()
	_, cached := svc.cache["user-a"]
	current := svc.generation
	svc.mu.Unlock()
	if cached {
		t.Fatal("stale lookup was cached after invalidation")
	}

	svc.cacheIfCurrent("user-a", UserFeatures{ProfileID: "fresh"}, current, now)
	svc.mu.Lock()
	entry, cached := svc.cache["user-a"]
	svc.mu.Unlock()
	if !cached || entry.features.ProfileID != "fresh" {
		t.Fatal("current lookup was not cached")
	}
}

func TestCacheDropsExpiredEntriesWhenFull(t *testing.T) {
	svc := setupAccessProfileTest(t)
	now := time.Now()
	svc.mu.Lock()
	for i := 0; i < maxCacheEntries; i++ {
		svc.cache[fmt.Sprintf("expired-%d", i)] = cacheEntry{expires: now.Add(-time.Second)}
	}
	svc.storeLocked("fresh", cacheEntry{expires: now.Add(time.Minute)}, now)
	size := len(svc.cache)
	svc.mu.Unlock()
	if size != 1 {
		t.Fatalf("expired entries kept: cache size %d", size)
	}
}
