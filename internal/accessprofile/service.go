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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"

	"gorm.io/gorm"
)

// DefaultProfileID is the ID of the built-in profile used by every user
// without an assignment.
const DefaultProfileID = "default"

const (
	defaultProfileName   = "Default"
	maxNameLength        = 128
	maxDescriptionLength = 512
	// maxCacheEntries bounds the per-process feature cache.
	maxCacheEntries = 10000
)

// cacheTTL bounds how long a feature lookup is reused. The admin console and
// the API server run in separate processes, so an admin change reaches the API
// server within this interval.
var cacheTTL = 30 * time.Second

// Errors returned by the service.
var (
	ErrInvalidInput           = errors.New("invalid input")
	ErrProfileNotFound        = errors.New("access profile not found")
	ErrProfileNameTaken       = errors.New("access profile name already exists")
	ErrDefaultProfileReadOnly = errors.New("the default access profile cannot be renamed or deleted")
	ErrUserNotFound           = errors.New("user not found")
)

// Profile is the admin view of an access profile.
type Profile struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Features     map[string]bool   `json:"features"`
	GlobalModels GlobalModelAccess `json:"global_models"`
	IsDefault    bool              `json:"is_default"`
	MemberCount  int64             `json:"member_count"`
}

// ProfileInput is the admin request body for creating or updating a profile.
type ProfileInput struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Features    map[string]bool `json:"features"`
	// GlobalModels is optional on update: when omitted the profile keeps its
	// current global model access.
	GlobalModels *GlobalModelAccess `json:"global_models,omitempty"`
}

// Assignment is one user's explicit profile assignment.
type Assignment struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	ProfileID string `json:"profile_id"`
}

// UserFeatures is the feature view returned to a signed-in user.
type UserFeatures struct {
	ProfileID    string            `json:"profile_id"`
	ProfileName  string            `json:"profile_name"`
	Features     map[string]bool   `json:"features"`
	GlobalModels GlobalModelAccess `json:"global_models"`
}

type cacheEntry struct {
	features UserFeatures
	expires  time.Time
}

// Service manages access profiles and answers feature checks.
type Service struct {
	profileDAO *dao.AccessProfileDAO
	userDAO    *dao.UserDAO

	mu    sync.Mutex
	cache map[string]cacheEntry
	// generation increases on every invalidation, so a lookup that started
	// before an admin change does not store its stale result.
	generation uint64
	// cacheDB is the database the cached lookups came from; a different
	// dao.DB starts an empty cache.
	cacheDB *gorm.DB
}

// NewService creates an access profile service backed by dao.DB.
func NewService() *Service {
	return &Service{
		profileDAO: dao.NewAccessProfileDAO(),
		userDAO:    dao.NewUserDAO(),
		cache:      make(map[string]cacheEntry),
	}
}

var defaultService = NewService()

// DefaultService returns the process-wide service used by middleware and
// handlers.
func DefaultService() *Service {
	return defaultService
}

// ensureDefaultProfile creates the built-in default profile with every feature
// enabled when it does not exist yet.
func (s *Service) ensureDefaultProfile(ctx context.Context) (*entity.AccessProfile, error) {
	profile, err := s.profileDAO.GetByID(ctx, dao.DB, DefaultProfileID)
	if err != nil || profile != nil {
		return profile, err
	}
	profile = &entity.AccessProfile{
		ID:          DefaultProfileID,
		Name:        defaultProfileName,
		Description: "Applies to every user without an assigned profile.",
		Features:    allEnabled().toStored(),
	}
	if err := s.profileDAO.Create(ctx, dao.DB, profile); err != nil {
		// Another request may have created it first.
		existing, getErr := s.profileDAO.GetByID(ctx, dao.DB, DefaultProfileID)
		if getErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return profile, nil
}

func toProfile(profile *entity.AccessProfile, memberCount int64) Profile {
	return Profile{
		ID:           profile.ID,
		Name:         profile.Name,
		Description:  profile.Description,
		Features:     featureSetFromStored(profile.Features).toResponse(),
		GlobalModels: globalModelAccessFromStored(profile.GlobalModels),
		IsDefault:    profile.ID == DefaultProfileID,
		MemberCount:  memberCount,
	}
}

// List returns every profile, the default profile first.
func (s *Service) List(ctx context.Context) ([]Profile, error) {
	if _, err := s.ensureDefaultProfile(ctx); err != nil {
		return nil, err
	}
	profiles, err := s.profileDAO.List(ctx, dao.DB)
	if err != nil {
		return nil, err
	}
	counts, err := s.profileDAO.CountMembers(ctx, dao.DB)
	if err != nil {
		return nil, err
	}
	result := make([]Profile, 0, len(profiles))
	for i := range profiles {
		view := toProfile(&profiles[i], counts[profiles[i].ID])
		if view.IsDefault {
			result = append([]Profile{view}, result...)
			continue
		}
		result = append(result, view)
	}
	return result, nil
}

// Get returns one profile.
func (s *Service) Get(ctx context.Context, id string) (*Profile, error) {
	if id == DefaultProfileID {
		if _, err := s.ensureDefaultProfile(ctx); err != nil {
			return nil, err
		}
	}
	profile, err := s.profileDAO.GetByID(ctx, dao.DB, id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	counts, err := s.profileDAO.CountMembers(ctx, dao.DB)
	if err != nil {
		return nil, err
	}
	view := toProfile(profile, counts[profile.ID])
	return &view, nil
}

func normalizeInput(input ProfileInput) (string, string, FeatureSet, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", "", nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return "", "", nil, fmt.Errorf("%w: name must be at most %d characters", ErrInvalidInput, maxNameLength)
	}
	description := strings.TrimSpace(input.Description)
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return "", "", nil, fmt.Errorf("%w: description must be at most %d characters", ErrInvalidInput, maxDescriptionLength)
	}
	features, err := featureSetFromInput(input.Features)
	if err != nil {
		return "", "", nil, err
	}
	return name, description, features, nil
}

func (s *Service) checkNameAvailable(ctx context.Context, name, exceptID string) error {
	existing, err := s.profileDAO.GetByName(ctx, dao.DB, name)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != exceptID {
		return ErrProfileNameTaken
	}
	return nil
}

// Create adds a profile.
func (s *Service) Create(ctx context.Context, input ProfileInput) (*Profile, error) {
	if _, err := s.ensureDefaultProfile(ctx); err != nil {
		return nil, err
	}
	name, description, features, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.checkNameAvailable(ctx, name, ""); err != nil {
		return nil, err
	}
	globalModels := allGlobalModels()
	if input.GlobalModels != nil {
		if globalModels, err = normalizeGlobalModelAccess(*input.GlobalModels); err != nil {
			return nil, err
		}
	}
	profile := &entity.AccessProfile{
		ID:           common.GenerateUUID(),
		Name:         name,
		Description:  description,
		Features:     features.toStored(),
		GlobalModels: globalModels.toStored(),
	}
	if err := s.profileDAO.Create(ctx, dao.DB, profile); err != nil {
		// A concurrent create can pass the pre-check and hit the unique index.
		if existing, getErr := s.profileDAO.GetByName(ctx, dao.DB, name); getErr == nil && existing != nil {
			return nil, ErrProfileNameTaken
		}
		return nil, err
	}
	view := toProfile(profile, 0)
	return &view, nil
}

// Update replaces a profile's name, description, and switches. The default
// profile keeps its name.
func (s *Service) Update(ctx context.Context, id string, input ProfileInput) (*Profile, error) {
	if id == DefaultProfileID {
		if _, err := s.ensureDefaultProfile(ctx); err != nil {
			return nil, err
		}
	}
	profile, err := s.profileDAO.GetByID(ctx, dao.DB, id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	name, description, features, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}
	if id == DefaultProfileID && name != profile.Name {
		return nil, ErrDefaultProfileReadOnly
	}
	if err := s.checkNameAvailable(ctx, name, id); err != nil {
		return nil, err
	}
	profile.Name = name
	profile.Description = description
	profile.Features = features.toStored()
	if input.GlobalModels != nil {
		globalModels, err := normalizeGlobalModelAccess(*input.GlobalModels)
		if err != nil {
			return nil, err
		}
		profile.GlobalModels = globalModels.toStored()
	}
	if err := s.profileDAO.Update(ctx, dao.DB, profile); err != nil {
		return nil, err
	}
	s.invalidateCache()
	return s.Get(ctx, id)
}

// Delete removes a profile. Its users fall back to the default profile.
func (s *Service) Delete(ctx context.Context, id string) error {
	if id == DefaultProfileID {
		return ErrDefaultProfileReadOnly
	}
	profile, err := s.profileDAO.GetByID(ctx, dao.DB, id)
	if err != nil {
		return err
	}
	if profile == nil {
		return ErrProfileNotFound
	}
	if err := s.profileDAO.Delete(ctx, dao.DB, id); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

// ListAssignments returns every user with an explicit profile assignment.
func (s *Service) ListAssignments(ctx context.Context) ([]Assignment, error) {
	assignments, err := s.profileDAO.ListAssignments(ctx, dao.DB)
	if err != nil {
		return nil, err
	}
	if len(assignments) == 0 {
		return []Assignment{}, nil
	}
	userIDs := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		userIDs = append(userIDs, assignment.UserID)
	}
	var users []entity.User
	if err := dao.DB.WithContext(ctx).Select("id", "email").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	emails := make(map[string]string, len(users))
	for _, user := range users {
		emails[user.ID] = user.Email
	}
	result := make([]Assignment, 0, len(assignments))
	for _, assignment := range assignments {
		email, ok := emails[assignment.UserID]
		if !ok {
			// The user was deleted; the stale row has no effect.
			continue
		}
		result = append(result, Assignment{UserID: assignment.UserID, Email: email, ProfileID: assignment.ProfileID})
	}
	return result, nil
}

// AssignUser assigns a profile to the user with the given email. Assigning the
// default profile removes the explicit assignment.
func (s *Service) AssignUser(ctx context.Context, email, profileID string) (*Assignment, error) {
	user, err := s.userDAO.GetByEmail(ctx, dao.DB, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if user.IsSuperuser != nil && *user.IsSuperuser {
		return nil, fmt.Errorf("%w: superusers use the admin console, where access profiles do not apply", ErrInvalidInput)
	}
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, fmt.Errorf("%w: profile_id is required", ErrInvalidInput)
	}
	if profileID == DefaultProfileID {
		err = s.profileDAO.DeleteAssignment(ctx, dao.DB, user.ID)
	} else {
		var profile *entity.AccessProfile
		profile, err = s.profileDAO.GetByID(ctx, dao.DB, profileID)
		if err != nil {
			return nil, err
		}
		if profile == nil {
			return nil, ErrProfileNotFound
		}
		err = s.profileDAO.SetAssignment(ctx, dao.DB, user.ID, profileID)
	}
	if err != nil {
		return nil, err
	}
	s.invalidateCache()
	return &Assignment{UserID: user.ID, Email: user.Email, ProfileID: profileID}, nil
}

// FeaturesForUser returns the switches that apply to a user.
func (s *Service) FeaturesForUser(ctx context.Context, userID string) (UserFeatures, error) {
	now := time.Now()
	s.mu.Lock()
	if s.cacheDB != dao.DB {
		s.cache = make(map[string]cacheEntry)
		s.cacheDB = dao.DB
		s.generation++
	}
	entry, ok := s.cache[userID]
	generation := s.generation
	s.mu.Unlock()
	if ok && now.Before(entry.expires) {
		return entry.features, nil
	}

	features, err := s.loadFeatures(ctx, userID)
	if err != nil {
		return UserFeatures{}, err
	}
	s.cacheIfCurrent(userID, features, generation, now)
	return features, nil
}

// cacheIfCurrent caches a lookup unless the cache was invalidated after the
// lookup started, which would otherwise restore switches an admin just changed.
func (s *Service) cacheIfCurrent(userID string, features UserFeatures, generation uint64, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.generation == generation {
		s.storeLocked(userID, cacheEntry{features: features, expires: now.Add(cacheTTL)}, now)
	}
}

// storeLocked caches an entry, first dropping expired entries when the cache is
// full. s.mu must be held.
func (s *Service) storeLocked(userID string, entry cacheEntry, now time.Time) {
	if len(s.cache) >= maxCacheEntries {
		for key, cached := range s.cache {
			if !now.Before(cached.expires) {
				delete(s.cache, key)
			}
		}
		if len(s.cache) >= maxCacheEntries {
			s.cache = make(map[string]cacheEntry)
		}
	}
	s.cache[userID] = entry
}

func (s *Service) loadFeatures(ctx context.Context, userID string) (UserFeatures, error) {
	profileID, err := s.profileDAO.GetAssignment(ctx, dao.DB, userID)
	if err != nil {
		return UserFeatures{}, err
	}
	var profile *entity.AccessProfile
	if profileID != "" {
		profile, err = s.profileDAO.GetByID(ctx, dao.DB, profileID)
		if err != nil {
			return UserFeatures{}, err
		}
	}
	if profile == nil {
		profile, err = s.profileDAO.GetByID(ctx, dao.DB, DefaultProfileID)
		if err != nil {
			return UserFeatures{}, err
		}
	}
	if profile == nil {
		// No admin has configured profiles yet: nothing is restricted.
		return UserFeatures{ProfileID: DefaultProfileID, ProfileName: defaultProfileName, Features: allEnabled().toResponse(), GlobalModels: allGlobalModels()}, nil
	}
	return UserFeatures{
		ProfileID:    profile.ID,
		ProfileName:  profile.Name,
		Features:     featureSetFromStored(profile.Features).toResponse(),
		GlobalModels: globalModelAccessFromStored(profile.GlobalModels),
	}, nil
}

// IsEnabled reports whether a feature is enabled for a user.
func (s *Service) IsEnabled(ctx context.Context, userID string, feature Feature) (bool, error) {
	features, err := s.FeaturesForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	enabled, ok := features.Features[string(feature)]
	return ok && enabled, nil
}

func (s *Service) invalidateCache() {
	s.mu.Lock()
	s.cache = make(map[string]cacheEntry)
	s.generation++
	s.mu.Unlock()
}
