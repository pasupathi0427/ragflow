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

package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ragflow/internal/common"
	"ragflow/internal/service"

	"github.com/gin-gonic/gin"
)

func TestCustomAdminRoutesRequireAdminAuth(t *testing.T) {
	engine := gin.New()
	NewRouter(NewHandler(NewService())).Setup(engine)

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/access-profiles"},
		{http.MethodPost, "/api/v1/admin/access-profiles"},
		{http.MethodGet, "/api/v1/admin/access-profiles/default"},
		{http.MethodPut, "/api/v1/admin/access-profiles/default"},
		{http.MethodDelete, "/api/v1/admin/access-profiles/p1"},
		{http.MethodGet, "/api/v1/admin/access-profile-assignments"},
		{http.MethodPut, "/api/v1/admin/users/a@example.com/access-profile"},
		{http.MethodGet, "/api/v1/admin/global-models"},
		{http.MethodGet, "/api/v1/admin/global-models/catalog"},
		{http.MethodPost, "/api/v1/admin/global-models/providers"},
		{http.MethodDelete, "/api/v1/admin/global-models/providers/p1"},
		{http.MethodPost, "/api/v1/admin/global-models/providers/p1/instances"},
		{http.MethodPost, "/api/v1/admin/global-models/providers/p1/connection"},
		{http.MethodPut, "/api/v1/admin/global-models/instances/i1"},
		{http.MethodDelete, "/api/v1/admin/global-models/instances/i1"},
		{http.MethodPost, "/api/v1/admin/global-models/instances/i1/models"},
		{http.MethodPut, "/api/v1/admin/global-models/models/m1/status"},
		{http.MethodDelete, "/api/v1/admin/global-models/models/m1"},
		{http.MethodGet, "/api/v1/admin/global-models/defaults"},
		{http.MethodPut, "/api/v1/admin/global-models/defaults"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(route.method, route.path, strings.NewReader("{}")))
			var response struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode %q: %v", recorder.Body.String(), err)
			}
			if response.Code != int(common.CodeUnauthorized) {
				t.Fatalf("route reachable without admin token: %s", recorder.Body.String())
			}
		})
	}
}

func TestDescribeGlobalModelUsage(t *testing.T) {
	got := describeGlobalModelUsage(&service.GlobalModelUsage{Datasets: 2, Chats: 1, GlobalDefault: []string{"chat"}})
	want := "Still in use by 2 datasets, 1 chat and the global default for chat. Disable it instead, or move them to another model first."
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
