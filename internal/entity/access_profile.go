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

// AccessProfile is a named set of feature switches that a super admin assigns
// to users. Features maps a feature key to whether it is enabled.
type AccessProfile struct {
	ID          string  `gorm:"column:id;primaryKey;size:32" json:"id"`
	Name        string  `gorm:"column:name;size:128;not null;uniqueIndex:idx_access_profile_name" json:"name"`
	Description string  `gorm:"column:description;size:512;not null;default:''" json:"description"`
	Features    JSONMap `gorm:"column:features;type:longtext" json:"features"`
	// GlobalModels is {"all": bool, "model_ids": [...]}; nil allows every
	// global model.
	GlobalModels JSONMap `gorm:"column:global_models;type:longtext" json:"global_models"`
	BaseModel
}

// TableName specify table name
func (AccessProfile) TableName() string {
	return "access_profile"
}

// UserAccessProfile assigns one access profile to a user. A user without a row
// uses the built-in default profile.
type UserAccessProfile struct {
	UserID    string `gorm:"column:user_id;primaryKey;size:32" json:"user_id"`
	ProfileID string `gorm:"column:profile_id;size:32;not null;index" json:"profile_id"`
	BaseModel
}

// TableName specify table name
func (UserAccessProfile) TableName() string {
	return "user_access_profile"
}
