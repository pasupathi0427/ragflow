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

// Package globalmodel lets every user call model providers that a super admin
// configures once. Global providers live in the regular provider tables under
// the reserved owner entity.GlobalModelOwnerID. Users may use a global model
// when the admin has it enabled and their access profile allows it; users can
// never manage one, because the provider management paths only accept
// providers of the caller's own tenant.
package globalmodel

import (
	"context"
	"errors"
	"strings"

	"ragflow/internal/accessprofile"
	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"
	"ragflow/internal/permission"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Errors returned when a global model may not be used. Their messages are shown
// to users.
var (
	ErrNotAccessible = errors.New("model not found or inaccessible")
	ErrDisabled      = errors.New("this model has been disabled by your administrator")
	ErrNotAllowed    = errors.New("this model is not available for your account; contact your administrator")
)

const statusActive = "active"

// IsGlobal reports whether a provider is a global provider.
func IsGlobal(provider *entity.TenantModelProvider) bool {
	return provider != nil && provider.TenantID == entity.GlobalModelOwnerID
}

// ProviderByName returns the tenant owner's provider with the given name, or
// the global provider of that name when the owner has none. The tenant's own
// provider always wins. Callers must still pass the result to CheckUse.
func ProviderByName(ctx context.Context, db *gorm.DB, tenantID, ownerUserID, providerName string) (*entity.TenantModelProvider, error) {
	providerDAO := dao.NewTenantModelProviderDAO()
	provider, err := providerDAO.GetByTenantIDAndUserIDAndProviderName(ctx, db, tenantID, ownerUserID, providerName)
	if err == nil && provider != nil {
		return provider, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return providerDAO.GetByTenantIDAndUserIDAndProviderName(ctx, db, entity.GlobalModelOwnerID, entity.GlobalModelOwnerID, providerName)
}

// OwnedOrGlobalProviderIDs scopes a provider query to the tenant owner's
// providers plus the global ones, for lookups by model ID.
func OwnedOrGlobalProviderIDs(db *gorm.DB, tenantID, ownerUserID string) *gorm.DB {
	return db.Session(&gorm.Session{}).Model(&entity.TenantModelProvider{}).
		Select("id").
		Where("(tenant_id = ? AND user_id = ?) OR (tenant_id = ? AND user_id = ?)", tenantID, ownerUserID, entity.GlobalModelOwnerID, entity.GlobalModelOwnerID)
}

// CheckUse decides whether the current caller may call a model of the
// provider. A provider of the tenant itself is always usable here; the
// caller's own checks still apply. A global model must exist as an enabled
// model of an enabled instance, and the calling user's access profile must
// allow it. Any other provider is reported as inaccessible.
func CheckUse(ctx context.Context, db *gorm.DB, tenantID string, provider *entity.TenantModelProvider, instance *entity.TenantModelInstance, modelName string) error {
	if done, err := checkProvider(tenantID, provider, instance); done {
		return err
	}
	models, err := dao.NewTenantModelDAO().GetModelsByProviderIDAndInstanceIDAndModelName(ctx, db, provider.ID, instance.ID, modelName)
	if err != nil {
		return err
	}
	var model *entity.TenantModel
	for _, candidate := range models {
		if strings.EqualFold(candidate.Status, statusActive) {
			model = candidate
			break
		}
	}
	return checkGlobalModel(ctx, db, tenantID, provider, instance, model)
}

// CheckUseModel is CheckUse for a caller that already holds the model row.
func CheckUseModel(ctx context.Context, db *gorm.DB, tenantID string, provider *entity.TenantModelProvider, instance *entity.TenantModelInstance, model *entity.TenantModel) error {
	if done, err := checkProvider(tenantID, provider, instance); done {
		return err
	}
	return checkGlobalModel(ctx, db, tenantID, provider, instance, model)
}

// checkProvider settles every case except an enabled global instance: done is
// true when the result is final.
func checkProvider(tenantID string, provider *entity.TenantModelProvider, instance *entity.TenantModelInstance) (done bool, err error) {
	if provider == nil {
		return true, ErrNotAccessible
	}
	if provider.TenantID == tenantID && tenantID != "" {
		return true, nil
	}
	if !IsGlobal(provider) {
		return true, ErrNotAccessible
	}
	if instance == nil || instance.ProviderID != provider.ID {
		return true, ErrNotAccessible
	}
	if !strings.EqualFold(instance.Status, statusActive) {
		return true, ErrDisabled
	}
	return false, nil
}

// checkGlobalModel requires an enabled model of the instance that the calling
// user's access profile allows.
func checkGlobalModel(ctx context.Context, db *gorm.DB, tenantID string, provider *entity.TenantModelProvider, instance *entity.TenantModelInstance, model *entity.TenantModel) error {
	if model == nil || model.ProviderID != provider.ID || model.InstanceID != instance.ID || !strings.EqualFold(model.Status, statusActive) {
		// The admin did not enable this model, or disabled it.
		return ErrDisabled
	}
	userID, err := callerUserID(ctx, db, tenantID)
	if err != nil {
		return err
	}
	allowed, err := accessprofile.DefaultService().AllowsGlobalModel(ctx, userID, model.ID)
	if err != nil {
		common.Error("global model access check failed", err, zap.String("model_id", model.ID))
		return ErrNotAllowed
	}
	if !allowed {
		return ErrNotAllowed
	}
	return nil
}

// callerUserID is the user making the call: the authenticated user when the
// request carries one, otherwise (background ingestion, scheduled agents) the
// owner of the tenant the work runs in.
func callerUserID(ctx context.Context, db *gorm.DB, tenantID string) (string, error) {
	if userID := strings.TrimSpace(permission.UserIDFromContext(ctx)); userID != "" {
		return userID, nil
	}
	owner, err := dao.NewUserTenantDAO().GetOwnerByTenantID(ctx, db, tenantID)
	if err != nil {
		return "", err
	}
	return owner.UserID, nil
}
