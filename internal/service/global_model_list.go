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

package service

import (
	"context"
	"sort"
	"strings"

	"ragflow/internal/accessprofile"
	"ragflow/internal/dao"
	"ragflow/internal/entity"
	"ragflow/internal/permission"
)

// AppendGlobalModels adds the global models the signed-in user may call to a
// model list, in the list's format, marked "is_global". Disabled models and
// models outside the user's access profile are left out, matching what a call
// would accept. No key or endpoint is included.
func (m *ModelProviderService) AppendGlobalModels(ctx context.Context, models []map[string]interface{}, modelTypeFilter string) ([]map[string]interface{}, error) {
	userID := strings.TrimSpace(permission.UserIDFromContext(ctx))
	if userID == "" {
		return models, nil
	}
	features, err := accessprofile.DefaultService().FeaturesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	filter := entity.ModelTypeFromString(strings.ToLower(strings.TrimSpace(modelTypeFilter)))

	var providers []entity.TenantModelProvider
	if err := dao.DB.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", entity.GlobalModelOwnerID, entity.GlobalModelOwnerID).
		Find(&providers).Error; err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return models, nil
	}
	providerNames := make(map[string]string, len(providers))
	providerIDs := make([]string, 0, len(providers))
	for _, provider := range providers {
		providerNames[provider.ID] = provider.ProviderName
		providerIDs = append(providerIDs, provider.ID)
	}
	var instances []entity.TenantModelInstance
	if err := dao.DB.WithContext(ctx).Select("id", "instance_name", "provider_id", "status").
		Where("provider_id IN ? AND status = ?", providerIDs, globalStatusActive).
		Find(&instances).Error; err != nil {
		return nil, err
	}
	instanceNames := make(map[string]string, len(instances))
	instanceIDs := make([]string, 0, len(instances))
	for _, instance := range instances {
		instanceNames[instance.ID] = instance.InstanceName
		instanceIDs = append(instanceIDs, instance.ID)
	}
	if len(instanceIDs) == 0 {
		return models, nil
	}
	var rows []entity.TenantModel
	if err := dao.DB.WithContext(ctx).
		Where("instance_id IN ? AND status = ?", instanceIDs, globalStatusActive).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	global := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		modelType := entity.ModelType(row.ModelType)
		if filter != 0 && !modelType.Has(filter) {
			continue
		}
		if !features.GlobalModels.Allows(row.ID) {
			continue
		}
		global = append(global, map[string]interface{}{
			"model_id":        row.ID,
			"status":          row.Status,
			"instance_status": globalStatusActive,
			"tenant_id":       entity.GlobalModelOwnerID,
			"tenant_name":     "",
			"model_type":      modelType.HumanReadable(),
			"name":            row.ModelName,
			"provider_id":     row.ProviderID,
			"provider_name":   providerNames[row.ProviderID],
			"instance_id":     row.InstanceID,
			"instance_name":   instanceNames[row.InstanceID],
			"rank":            500,
			"is_global":       true,
		})
	}
	sort.SliceStable(global, func(i, j int) bool {
		pi, pj := global[i]["provider_name"].(string), global[j]["provider_name"].(string)
		if pi != pj {
			return pi < pj
		}
		ii, ij := global[i]["instance_name"].(string), global[j]["instance_name"].(string)
		if ii != ij {
			return ii < ij
		}
		return global[i]["name"].(string) < global[j]["name"].(string)
	})
	return append(models, global...), nil
}
