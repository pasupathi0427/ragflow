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

package entity

import (
	"ragflow/internal/common"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GlobalModelOwnerID is the tenant_id and user_id of model providers a super
// admin configures for every user. No real tenant or user has this ID.
const GlobalModelOwnerID = "global"

// AfterFind decrypts an API key stored encrypted. Only global providers store
// encrypted keys; other keys pass through unchanged. A key that cannot be
// decrypted is cleared, so the model call fails instead of the whole query.
func (i *TenantModelInstance) AfterFind(*gorm.DB) error {
	if !common.IsEncryptedModelAPIKey(i.APIKey) {
		return nil
	}
	plaintext, err := common.DecryptModelAPIKey(i.APIKey)
	if err != nil {
		common.Error("model API key decryption failed", err, zap.String("instance_id", i.ID))
		i.APIKey = ""
		return nil
	}
	i.APIKey = plaintext
	return nil
}

// BeforeSave keeps a global provider's key encrypted when code that loaded the
// instance (and so holds the decrypted key) saves it back. The global model
// admin service encrypts keys itself; this is a safety net, so a failed
// provider lookup leaves the save alone.
func (i *TenantModelInstance) BeforeSave(tx *gorm.DB) error {
	if i.APIKey == "" || common.IsEncryptedModelAPIKey(i.APIKey) || i.ProviderID == "" {
		return nil
	}
	var ownerTenantIDs []string
	err := tx.Session(&gorm.Session{NewDB: true}).
		Model(&TenantModelProvider{}).
		Where("id = ?", i.ProviderID).
		Limit(1).
		Pluck("tenant_id", &ownerTenantIDs).Error
	if err != nil || len(ownerTenantIDs) == 0 || ownerTenantIDs[0] != GlobalModelOwnerID {
		return nil
	}
	encrypted, err := common.EncryptModelAPIKey(i.APIKey)
	if err != nil {
		return err
	}
	i.APIKey = encrypted
	return nil
}
