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

	"github.com/gin-gonic/gin"
)

func TestAccessProfileRoutesRequireAdminAuth(t *testing.T) {
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
