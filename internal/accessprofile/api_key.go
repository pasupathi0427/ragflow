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
	"ragflow/internal/dao"
	"ragflow/internal/entity"

	"go.uber.org/zap"
)

// Errors returned by CheckAPIKeyAccess.
var (
	ErrAPIAccessDisabled    = errors.New("API access is disabled for this account. Contact your administrator.")
	ErrAPIAccountDisabled   = errors.New("This account has been disabled. Contact your administrator.")
	ErrAPIKeyDisabled       = errors.New("This API key is disabled. Contact your administrator.")
	ErrAPIAccessCheckFailed = errors.New("API access check failed")
)

// CheckAPIKeyAccess rejects a request authenticated with an API key when the
// key owner is inactive, when the owner's profile turns API access off, or when
// the key is one a super admin manages and has switched off. Embed (beta)
// tokens are not API keys and are not checked.
func CheckAPIKeyAccess(ctx context.Context, user *entity.User, token string) error {
	if user.IsActive != "1" {
		return ErrAPIAccountDisabled
	}
	enabled, err := DefaultService().IsEnabled(ctx, user.ID, FeatureAPI)
	if err != nil {
		common.Error("access profile lookup failed", err, zap.String("user_id", user.ID))
		return ErrAPIAccessCheckFailed
	}
	if !enabled {
		return ErrAPIAccessDisabled
	}
	adminKey, err := dao.NewAdminAPIKeyDAO().GetByToken(ctx, dao.DB, token)
	if err != nil {
		common.Error("admin API key lookup failed", err, zap.String("user_id", user.ID))
		return ErrAPIAccessCheckFailed
	}
	if adminKey != nil && adminKey.Enabled != "1" {
		return ErrAPIKeyDisabled
	}
	return nil
}
