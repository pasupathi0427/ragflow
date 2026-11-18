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

// Package accessprofile lets a super admin decide which user-settings features
// each user can see and use. A profile is a named set of feature switches; a
// user has at most one assigned profile and otherwise uses the built-in
// default profile.
package accessprofile

import (
	"fmt"

	"ragflow/internal/entity"
)

// Feature identifies a user-settings feature that a profile can switch off.
type Feature string

// Features a profile controls. Profile is always available and is not listed.
const (
	FeatureModelProviders Feature = "model_providers"
	FeatureDataSources    Feature = "data_sources"
	FeatureChatChannels   Feature = "chat_channels"
	FeatureMCP            Feature = "mcp"
	FeatureTeam           Feature = "team"
	FeatureAPI            Feature = "api"
)

// AllFeatures lists every feature in display order.
var AllFeatures = []Feature{
	FeatureModelProviders,
	FeatureDataSources,
	FeatureChatChannels,
	FeatureMCP,
	FeatureTeam,
	FeatureAPI,
}

func isKnownFeature(feature Feature) bool {
	for _, known := range AllFeatures {
		if known == feature {
			return true
		}
	}
	return false
}

// FeatureSet maps every known feature to whether it is enabled.
type FeatureSet map[Feature]bool

// allEnabled returns a feature set with every feature on.
func allEnabled() FeatureSet {
	set := make(FeatureSet, len(AllFeatures))
	for _, feature := range AllFeatures {
		set[feature] = true
	}
	return set
}

// featureSetFromStored reads the stored switches of a profile. A feature with
// no stored switch is enabled, so features added later stay available until an
// admin turns them off.
func featureSetFromStored(stored entity.JSONMap) FeatureSet {
	set := allEnabled()
	for _, feature := range AllFeatures {
		if enabled, ok := stored[string(feature)].(bool); ok {
			set[feature] = enabled
		}
	}
	return set
}

// featureSetFromInput validates switches sent by an admin. Unknown features are
// rejected and missing features are enabled.
func featureSetFromInput(input map[string]bool) (FeatureSet, error) {
	set := allEnabled()
	for key, enabled := range input {
		feature := Feature(key)
		if !isKnownFeature(feature) {
			return nil, fmt.Errorf("%w: unknown feature %q", ErrInvalidInput, key)
		}
		set[feature] = enabled
	}
	return set, nil
}

func (s FeatureSet) toStored() entity.JSONMap {
	stored := make(entity.JSONMap, len(s))
	for feature, enabled := range s {
		stored[string(feature)] = enabled
	}
	return stored
}

func (s FeatureSet) toResponse() map[string]bool {
	response := make(map[string]bool, len(s))
	for feature, enabled := range s {
		response[string(feature)] = enabled
	}
	return response
}
