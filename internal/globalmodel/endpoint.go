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
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"ragflow/internal/entity"
)

// ErrEndpointOverride is returned when a caller points a global model at
// another endpoint than the one the admin configured.
var ErrEndpointOverride = errors.New("a global model's endpoint cannot be overridden")

// KeyMayGoTo reports whether a global instance's API key may be sent to the
// endpoint a caller asked for. Only the admin-configured endpoint qualifies: a
// caller-supplied base URL of another origin would receive the admin's key.
// An empty override keeps the configured endpoint and is always allowed.
func KeyMayGoTo(instance *entity.TenantModelInstance, override string) bool {
	override = strings.TrimSpace(override)
	if override == "" {
		return true
	}
	if instance == nil {
		return false
	}
	var extra map[string]string
	if strings.TrimSpace(instance.Extra) != "" {
		_ = json.Unmarshal([]byte(instance.Extra), &extra)
	}
	return sameOrigin(extra["base_url"], override)
}

func sameOrigin(a, b string) bool {
	ua, errA := url.Parse(strings.TrimSpace(a))
	ub, errB := url.Parse(strings.TrimSpace(b))
	if errA != nil || errB != nil || ua.Hostname() == "" || ub.Hostname() == "" {
		return false
	}
	return strings.EqualFold(ua.Scheme, ub.Scheme) &&
		strings.EqualFold(ua.Hostname(), ub.Hostname()) &&
		effectivePort(ua) == effectivePort(ub)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	}
	return ""
}
