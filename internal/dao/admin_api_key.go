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

package dao

import (
	"context"
	"errors"

	"ragflow/internal/entity"

	"gorm.io/gorm"
)

// AdminAPIKeyDAO reads and writes API keys that a super admin manages for a
// user, together with their api_token rows.
type AdminAPIKeyDAO struct{}

// NewAdminAPIKeyDAO creates an admin API key DAO.
func NewAdminAPIKeyDAO() *AdminAPIKeyDAO {
	return &AdminAPIKeyDAO{}
}

// Create stores the api_token row and its admin record in one transaction.
func (d *AdminAPIKeyDAO) Create(ctx context.Context, db *gorm.DB, token *entity.APIToken, key *entity.AdminAPIKey) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(token).Error; err != nil {
			return err
		}
		return tx.Create(key).Error
	})
}

// ListByUser returns the user's admin-managed keys, newest first.
func (d *AdminAPIKeyDAO) ListByUser(ctx context.Context, db *gorm.DB, userID string) ([]entity.AdminAPIKey, error) {
	var keys []entity.AdminAPIKey
	err := db.WithContext(ctx).Where("user_id = ?", userID).Order("create_time DESC").Find(&keys).Error
	return keys, err
}

// GetForUser returns the key with the given ID when it belongs to the user, or
// nil when there is none.
func (d *AdminAPIKeyDAO) GetForUser(ctx context.Context, db *gorm.DB, userID, id string) (*entity.AdminAPIKey, error) {
	var key entity.AdminAPIKey
	err := db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Take(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// GetByToken returns the admin record for an API token, or nil when the token
// is not admin-managed.
func (d *AdminAPIKeyDAO) GetByToken(ctx context.Context, db *gorm.DB, token string) (*entity.AdminAPIKey, error) {
	var key entity.AdminAPIKey
	err := db.WithContext(ctx).Where("token = ?", token).Take(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// TokensForTenant returns the set of admin-managed tokens in a tenant.
func (d *AdminAPIKeyDAO) TokensForTenant(ctx context.Context, db *gorm.DB, tenantID string) (map[string]struct{}, error) {
	var tokens []string
	err := db.WithContext(ctx).Model(&entity.AdminAPIKey{}).Where("tenant_id = ?", tenantID).Pluck("token", &tokens).Error
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}
	return set, nil
}

// SetEnabled switches a key on or off.
func (d *AdminAPIKeyDAO) SetEnabled(ctx context.Context, db *gorm.DB, id string, enabled bool) error {
	value := "0"
	if enabled {
		value = "1"
	}
	return db.WithContext(ctx).Model(&entity.AdminAPIKey{}).Where("id = ?", id).Update("enabled", value).Error
}

// Delete removes the key's api_token row and its admin record.
func (d *AdminAPIKeyDAO) Delete(ctx context.Context, db *gorm.DB, key *entity.AdminAPIKey) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND token = ?", key.TenantID, key.Token).Delete(&entity.APIToken{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", key.ID).Delete(&entity.AdminAPIKey{}).Error
	})
}
