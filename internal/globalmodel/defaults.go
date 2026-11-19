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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"

	"gorm.io/gorm"
)

// DefaultsSetting is the system_settings row holding the global default model
// ID per model type name, e.g. {"chat": "<tenant_model id>"}.
const DefaultsSetting = "global_models.defaults"

// LoadDefaults reads the global default model ID per model type name.
func LoadDefaults(ctx context.Context, db *gorm.DB) (map[string]string, error) {
	settings, err := dao.NewSystemSettingsDAO().GetByName(ctx, db, DefaultsSetting)
	if err != nil {
		return nil, err
	}
	defaults := map[string]string{}
	if len(settings) == 0 || strings.TrimSpace(settings[0].Value) == "" {
		return defaults, nil
	}
	if err := json.Unmarshal([]byte(settings[0].Value), &defaults); err != nil {
		return nil, fmt.Errorf("decode global default models: %w", err)
	}
	return defaults, nil
}

// DefaultModelID returns the global default model for a type, or "" when none
// is set. Callers resolve it like any model ID, so a disabled or disallowed
// default is still refused.
func DefaultModelID(ctx context.Context, db *gorm.DB, modelType entity.ModelType) string {
	defaults, err := LoadDefaults(ctx, db)
	if err != nil {
		common.Error("load global default models failed", err)
		return ""
	}
	return defaults[modelType.String()]
}
