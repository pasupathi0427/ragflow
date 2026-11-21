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
	"fmt"
	"strings"
	"unicode/utf8"

	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"

	"gorm.io/gorm"
)

const maxAPIKeyNameLength = 128

// ErrAPIKeyNotFound is returned when an admin-managed key does not exist for
// the user.
var ErrAPIKeyNotFound = errors.New("API key not found")

// AdminAPIKey is the admin view of a key. Token is set only in the response
// that creates the key; afterwards only MaskedToken is returned.
type AdminAPIKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MaskedToken string `json:"masked_token"`
	Token       string `json:"token,omitempty"`
	Enabled     bool   `json:"enabled"`
	CreatedBy   string `json:"created_by"`
	CreateTime  *int64 `json:"create_time,omitempty"`
}

func maskToken(token string) string {
	if len(token) <= 16 {
		return strings.Repeat("•", 8)
	}
	return token[:12] + "…" + token[len(token)-4:]
}

func toAdminAPIKey(key *entity.AdminAPIKey) AdminAPIKey {
	return AdminAPIKey{
		ID:          key.ID,
		Name:        key.Name,
		MaskedToken: maskToken(key.Token),
		Enabled:     key.Enabled == "1",
		CreatedBy:   key.CreatedBy,
		CreateTime:  key.CreateTime,
	}
}

// keyOwner returns the user that admin keys are managed for. Superusers sign in
// to the admin console only and cannot hold tenant API keys.
func (s *Service) keyOwner(ctx context.Context, email string) (*entity.User, error) {
	user, err := s.userDAO.GetByEmail(ctx, dao.DB, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if user.IsSuperuser != nil && *user.IsSuperuser {
		return nil, fmt.Errorf("%w: superusers cannot hold API keys", ErrInvalidInput)
	}
	return user, nil
}

// ListAPIKeys returns the keys the admin manages for a user.
func (s *Service) ListAPIKeys(ctx context.Context, email string) ([]AdminAPIKey, error) {
	user, err := s.keyOwner(ctx, email)
	if err != nil {
		return nil, err
	}
	keys, err := dao.NewAdminAPIKeyDAO().ListByUser(ctx, dao.DB, user.ID)
	if err != nil {
		return nil, err
	}
	views := make([]AdminAPIKey, len(keys))
	for i := range keys {
		views[i] = toAdminAPIKey(&keys[i])
	}
	return views, nil
}

// CreateAPIKey creates an enabled key in the user's own tenant. It is created
// without an embed (beta) token, so it cannot be used to embed chat pages.
func (s *Service) CreateAPIKey(ctx context.Context, email, name, createdBy string) (*AdminAPIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(name) > maxAPIKeyNameLength {
		return nil, fmt.Errorf("%w: name must be at most %d characters", ErrInvalidInput, maxAPIKeyNameLength)
	}
	user, err := s.keyOwner(ctx, email)
	if err != nil {
		return nil, err
	}
	userID := user.ID
	token := &entity.APIToken{
		TenantID: user.ID,
		UserID:   &userID,
		Token:    common.GenerateAPIToken(),
	}
	key := &entity.AdminAPIKey{
		ID:        common.GenerateToken(),
		TenantID:  token.TenantID,
		UserID:    user.ID,
		Token:     token.Token,
		Name:      name,
		Enabled:   "1",
		CreatedBy: createdBy,
	}
	if err := dao.NewAdminAPIKeyDAO().Create(ctx, dao.DB, token, key); err != nil {
		return nil, err
	}
	view := toAdminAPIKey(key)
	view.Token = key.Token
	return &view, nil
}

// SetAPIKeyEnabled switches one of the user's admin-managed keys on or off.
func (s *Service) SetAPIKeyEnabled(ctx context.Context, email, keyID string, enabled bool) (*AdminAPIKey, error) {
	key, err := s.adminKey(ctx, email, keyID)
	if err != nil {
		return nil, err
	}
	if err := dao.NewAdminAPIKeyDAO().SetEnabled(ctx, dao.DB, key.ID, enabled); err != nil {
		return nil, err
	}
	key.Enabled = "0"
	if enabled {
		key.Enabled = "1"
	}
	view := toAdminAPIKey(key)
	return &view, nil
}

// DeleteAPIKey removes one of the user's admin-managed keys.
func (s *Service) DeleteAPIKey(ctx context.Context, email, keyID string) error {
	key, err := s.adminKey(ctx, email, keyID)
	if err != nil {
		return err
	}
	return dao.NewAdminAPIKeyDAO().Delete(ctx, dao.DB, key)
}

func (s *Service) adminKey(ctx context.Context, email, keyID string) (*entity.AdminAPIKey, error) {
	user, err := s.keyOwner(ctx, email)
	if err != nil {
		return nil, err
	}
	key, err := dao.NewAdminAPIKeyDAO().GetForUser(ctx, dao.DB, user.ID, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrAPIKeyNotFound
	}
	return key, nil
}
