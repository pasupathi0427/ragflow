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

// AdminAPIKey marks an api_token row that a super admin created for a user.
// The key itself stays in api_token so every API route accepts it; this row
// hides it from the user's own key list and lets the admin switch it off.
type AdminAPIKey struct {
	ID       string `gorm:"column:id;primaryKey;size:32" json:"id"`
	TenantID string `gorm:"column:tenant_id;size:32;not null;index" json:"tenant_id"`
	UserID   string `gorm:"column:user_id;size:32;not null;index" json:"user_id"`
	Token    string `gorm:"column:token;size:255;not null;uniqueIndex:idx_admin_api_key_token" json:"-"`
	Name     string `gorm:"column:name;size:128;not null;default:''" json:"name"`
	// Enabled is "1" or "0". A string avoids GORM skipping a false bool on
	// insert when the column has a default.
	Enabled   string `gorm:"column:enabled;size:1;not null;default:1" json:"enabled"`
	CreatedBy string `gorm:"column:created_by;size:255;not null;default:''" json:"created_by"`
	BaseModel
}

// TableName specify table name
func (AdminAPIKey) TableName() string {
	return "admin_api_key"
}
