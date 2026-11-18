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
	"errors"

	"ragflow/internal/common"

	"go.uber.org/zap"
)

// Errors returned by CheckAPIKeyAccess.
var (
	ErrAPIAccessDisabled    = errors.New("API access is disabled for this account. Contact your administrator.")
	ErrAPIAccessCheckFailed = errors.New("API access check failed")
)

// CheckAPIKeyAccess rejects a request authenticated with an API key when the
// key owner's profile turns API access off. Embed (beta) tokens are not API
// keys and are not checked.
func CheckAPIKeyAccess(ctx context.Context, userID string) error {
	enabled, err := DefaultService().IsEnabled(ctx, userID, FeatureAPI)
	if err != nil {
		common.Error("access profile lookup failed", err, zap.String("user_id", userID))
		return ErrAPIAccessCheckFailed
	}
	if !enabled {
		return ErrAPIAccessDisabled
	}
	return nil
}
