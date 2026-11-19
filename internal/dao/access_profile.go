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
	"gorm.io/gorm/clause"
)

// AccessProfileDAO reads and writes access profiles and their user
// assignments.
type AccessProfileDAO struct{}

// NewAccessProfileDAO creates an access profile DAO.
func NewAccessProfileDAO() *AccessProfileDAO {
	return &AccessProfileDAO{}
}

// List returns every access profile ordered by name.
func (d *AccessProfileDAO) List(ctx context.Context, db *gorm.DB) ([]entity.AccessProfile, error) {
	var profiles []entity.AccessProfile
	err := db.WithContext(ctx).Order("name ASC").Find(&profiles).Error
	return profiles, err
}

// GetByID returns the profile with the given ID, or nil when it does not exist.
func (d *AccessProfileDAO) GetByID(ctx context.Context, db *gorm.DB, id string) (*entity.AccessProfile, error) {
	var profile entity.AccessProfile
	err := db.WithContext(ctx).Where("id = ?", id).Take(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetByName returns the profile whose name matches case-insensitively, or nil
// when none does. MySQL's default collation compares the unique name index the
// same way.
func (d *AccessProfileDAO) GetByName(ctx context.Context, db *gorm.DB, name string) (*entity.AccessProfile, error) {
	var profile entity.AccessProfile
	err := db.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name).Take(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Create inserts a profile.
func (d *AccessProfileDAO) Create(ctx context.Context, db *gorm.DB, profile *entity.AccessProfile) error {
	return db.WithContext(ctx).Create(profile).Error
}

// Update writes the name, description, and features of an existing profile.
func (d *AccessProfileDAO) Update(ctx context.Context, db *gorm.DB, profile *entity.AccessProfile) error {
	return db.WithContext(ctx).Model(&entity.AccessProfile{}).
		Where("id = ?", profile.ID).
		Updates(map[string]interface{}{
			"name":          profile.Name,
			"description":   profile.Description,
			"features":      profile.Features,
			"global_models": profile.GlobalModels,
		}).Error
}

// Delete removes a profile and every assignment that points to it, so its
// users fall back to the default profile.
func (d *AccessProfileDAO) Delete(ctx context.Context, db *gorm.DB, id string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("profile_id = ?", id).Delete(&entity.UserAccessProfile{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&entity.AccessProfile{}).Error
	})
}

// CountMembers returns the number of existing users assigned to each profile ID.
// Assignments of deleted users are not counted.
func (d *AccessProfileDAO) CountMembers(ctx context.Context, db *gorm.DB) (map[string]int64, error) {
	var rows []struct {
		ProfileID string
		Total     int64
	}
	existingUsers := db.WithContext(ctx).Model(&entity.User{}).Select("id")
	err := db.WithContext(ctx).Model(&entity.UserAccessProfile{}).
		Where("user_id IN (?)", existingUsers).
		Select("profile_id, COUNT(*) AS total").
		Group("profile_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.ProfileID] = row.Total
	}
	return counts, nil
}

// GetAssignment returns the profile ID assigned to a user, or "" when the user
// has no assignment.
func (d *AccessProfileDAO) GetAssignment(ctx context.Context, db *gorm.DB, userID string) (string, error) {
	var assignment entity.UserAccessProfile
	err := db.WithContext(ctx).Where("user_id = ?", userID).Take(&assignment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return assignment.ProfileID, nil
}

// ListAssignments returns every explicit user assignment.
func (d *AccessProfileDAO) ListAssignments(ctx context.Context, db *gorm.DB) ([]entity.UserAccessProfile, error) {
	var assignments []entity.UserAccessProfile
	err := db.WithContext(ctx).Order("user_id ASC").Find(&assignments).Error
	return assignments, err
}

// SetAssignment assigns a profile to a user, replacing any earlier assignment.
func (d *AccessProfileDAO) SetAssignment(ctx context.Context, db *gorm.DB, userID, profileID string) error {
	assignment := entity.UserAccessProfile{UserID: userID, ProfileID: profileID}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"profile_id", "update_time", "update_date"}),
	}).Create(&assignment).Error
}

// DeleteAssignment removes a user's assignment so the user falls back to the
// default profile.
func (d *AccessProfileDAO) DeleteAssignment(ctx context.Context, db *gorm.DB, userID string) error {
	return db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.UserAccessProfile{}).Error
}
