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
	"fmt"
	"strings"

	"ragflow/internal/entity"
)

// maxGlobalModelIDs bounds the allow list of one profile.
const maxGlobalModelIDs = 1000

// GlobalModelAccess says which global models a profile may use: every global
// model when All is set, otherwise only the tenant_model IDs in ModelIDs.
type GlobalModelAccess struct {
	All      bool     `json:"all"`
	ModelIDs []string `json:"model_ids"`
}

func allGlobalModels() GlobalModelAccess {
	return GlobalModelAccess{All: true, ModelIDs: []string{}}
}

// globalModelAccessFromStored reads a profile's stored access. A profile saved
// before global models existed allows all of them.
func globalModelAccessFromStored(stored entity.JSONMap) GlobalModelAccess {
	if stored == nil {
		return allGlobalModels()
	}
	access := GlobalModelAccess{ModelIDs: []string{}}
	if all, ok := stored["all"].(bool); ok {
		access.All = all
	}
	if ids, ok := stored["model_ids"].([]interface{}); ok {
		for _, id := range ids {
			if value, ok := id.(string); ok {
				access.ModelIDs = append(access.ModelIDs, value)
			}
		}
	}
	return access
}

func (a GlobalModelAccess) toStored() entity.JSONMap {
	ids := make([]interface{}, 0, len(a.ModelIDs))
	for _, id := range a.ModelIDs {
		ids = append(ids, id)
	}
	return entity.JSONMap{"all": a.All, "model_ids": ids}
}

// normalizeGlobalModelAccess trims and de-duplicates the allow list. With All
// set the list is dropped, so a later switch to "selected" starts empty.
func normalizeGlobalModelAccess(input GlobalModelAccess) (GlobalModelAccess, error) {
	if input.All {
		return allGlobalModels(), nil
	}
	seen := make(map[string]bool, len(input.ModelIDs))
	ids := make([]string, 0, len(input.ModelIDs))
	for _, id := range input.ModelIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) > maxGlobalModelIDs {
		return GlobalModelAccess{}, fmt.Errorf("%w: at most %d global models per profile", ErrInvalidInput, maxGlobalModelIDs)
	}
	return GlobalModelAccess{All: false, ModelIDs: ids}, nil
}

// Allows reports whether the access covers a global model.
func (a GlobalModelAccess) Allows(modelID string) bool {
	if a.All {
		return true
	}
	for _, id := range a.ModelIDs {
		if id == modelID {
			return true
		}
	}
	return false
}

// AllowsGlobalModel reports whether a user's profile lets them use a global
// model, identified by its tenant_model ID.
func (s *Service) AllowsGlobalModel(ctx context.Context, userID, modelID string) (bool, error) {
	features, err := s.FeaturesForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return features.GlobalModels.Allows(modelID), nil
}
