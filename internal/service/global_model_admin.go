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
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"ragflow/internal/common"
	"ragflow/internal/dao"
	"ragflow/internal/entity"
	"ragflow/internal/globalmodel"

	"gorm.io/gorm"
)

// Errors returned by the global model admin service.
var (
	ErrGlobalModelInvalidInput = errors.New("invalid input")
	ErrGlobalModelNotFound     = errors.New("global model configuration not found")
	ErrGlobalModelConflict     = errors.New("global model configuration already exists")
	ErrGlobalModelInUse        = errors.New("global model configuration is in use")
)

// GlobalModelUsage lists what still references a set of global models. A
// non-empty usage blocks deleting them.
type GlobalModelUsage struct {
	Datasets      int64    `json:"datasets"`
	Chats         int64    `json:"chats"`
	Memories      int64    `json:"memories"`
	Agents        int64    `json:"agents"`
	UserDefaults  int64    `json:"user_defaults"`
	Searches      int64    `json:"searches"`
	SkillConfigs  int64    `json:"skill_configs"`
	GlobalDefault []string `json:"global_defaults"`
}

// IsEmpty reports whether nothing references the models.
func (u GlobalModelUsage) IsEmpty() bool {
	return u.Datasets == 0 && u.Chats == 0 && u.Memories == 0 && u.Agents == 0 && u.UserDefaults == 0 &&
		u.Searches == 0 && u.SkillConfigs == 0 && len(u.GlobalDefault) == 0
}

// GlobalModelView is one global model as the admin console shows it.
type GlobalModelView struct {
	ID         string   `json:"id"`
	ModelName  string   `json:"model_name"`
	ModelTypes []string `json:"model_types"`
	Status     string   `json:"status"`
}

// GlobalInstanceView is one global provider instance. The API key is never
// returned, only whether one is set and its last four characters.
type GlobalInstanceView struct {
	ID           string            `json:"id"`
	InstanceName string            `json:"instance_name"`
	Status       string            `json:"status"`
	BaseURL      string            `json:"base_url"`
	Region       string            `json:"region"`
	HasAPIKey    bool              `json:"has_api_key"`
	APIKeyMasked string            `json:"api_key_masked"`
	Models       []GlobalModelView `json:"models"`
}

// GlobalProviderView is one global provider with its instances.
type GlobalProviderView struct {
	ID           string               `json:"id"`
	ProviderName string               `json:"provider_name"`
	Instances    []GlobalInstanceView `json:"instances"`
}

// GlobalInstanceInput creates or updates a global provider instance. On update,
// nil fields keep their current value.
type GlobalInstanceInput struct {
	InstanceName string                    `json:"instance_name"`
	APIKey       json.RawMessage           `json:"api_key"`
	ClearAPIKey  bool                      `json:"clear_api_key"`
	BaseURL      *string                   `json:"base_url"`
	Region       *string                   `json:"region"`
	Status       *string                   `json:"status"`
	Models       []CreateInstanceModelInfo `json:"models"`
}

// GlobalConnectionInput tests credentials for a global provider. Without an
// API key, the saved key of InstanceID is used.
type GlobalConnectionInput struct {
	InstanceID string                     `json:"instance_id"`
	APIKey     json.RawMessage            `json:"api_key"`
	BaseURL    string                     `json:"base_url"`
	Region     string                     `json:"region"`
	ModelInfo  []CheckConnectionModelInfo `json:"model_info"`
}

const (
	globalStatusActive   = "active"
	globalStatusInactive = "inactive"
)

// GlobalModelAdminService manages global model providers for super admins.
type GlobalModelAdminService struct {
	providers *ModelProviderService
	settings  *dao.SystemSettingsDAO
}

// NewGlobalModelAdminService creates the service.
func NewGlobalModelAdminService() *GlobalModelAdminService {
	return &GlobalModelAdminService{providers: NewModelProviderService(), settings: dao.NewSystemSettingsDAO()}
}

// Catalog lists the providers that can be configured, from the model catalog.
func (s *GlobalModelAdminService) Catalog() ([]map[string]interface{}, error) {
	return dao.GetModelProviderManager().ListProviders()
}

// CatalogModels lists the catalog models of a provider.
func (s *GlobalModelAdminService) CatalogModels(providerName string) ([]map[string]interface{}, error) {
	return dao.GetModelProviderManager().ListModels(providerName)
}

// List returns every global provider with its instances and models.
func (s *GlobalModelAdminService) List(ctx context.Context) ([]GlobalProviderView, error) {
	var providers []entity.TenantModelProvider
	if err := dao.DB.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", entity.GlobalModelOwnerID, entity.GlobalModelOwnerID).
		Order("provider_name ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	result := make([]GlobalProviderView, 0, len(providers))
	for _, provider := range providers {
		view := GlobalProviderView{ID: provider.ID, ProviderName: provider.ProviderName, Instances: []GlobalInstanceView{}}
		var instances []entity.TenantModelInstance
		if err := dao.DB.WithContext(ctx).Where("provider_id = ?", provider.ID).Order("instance_name ASC").Find(&instances).Error; err != nil {
			return nil, err
		}
		for _, instance := range instances {
			extra := decodeGlobalInstanceExtra(instance.Extra)
			instanceView := GlobalInstanceView{
				ID:           instance.ID,
				InstanceName: instance.InstanceName,
				Status:       instance.Status,
				BaseURL:      extra["base_url"],
				Region:       extra["region"],
				HasAPIKey:    instance.APIKey != "",
				APIKeyMasked: common.MaskModelAPIKey(instance.APIKey),
				Models:       []GlobalModelView{},
			}
			var models []entity.TenantModel
			if err := dao.DB.WithContext(ctx).Where("instance_id = ?", instance.ID).Order("model_name ASC").Find(&models).Error; err != nil {
				return nil, err
			}
			for _, model := range models {
				instanceView.Models = append(instanceView.Models, GlobalModelView{
					ID:         model.ID,
					ModelName:  model.ModelName,
					ModelTypes: entity.ModelType(model.ModelType).HumanReadable(),
					Status:     model.Status,
				})
			}
			view.Instances = append(view.Instances, instanceView)
		}
		result = append(result, view)
	}
	return result, nil
}

// AddProvider adds a catalog provider as a global provider.
func (s *GlobalModelAdminService) AddProvider(ctx context.Context, providerName string) (*entity.TenantModelProvider, error) {
	providerName = strings.TrimSpace(providerName)
	if providerName == "" || dao.GetModelProviderManager().FindProvider(providerName) == nil {
		return nil, fmt.Errorf("%w: unknown provider %q", ErrGlobalModelInvalidInput, providerName)
	}
	existing, err := dao.NewTenantModelProviderDAO().GetByTenantIDAndUserIDAndProviderName(ctx, dao.DB, entity.GlobalModelOwnerID, entity.GlobalModelOwnerID, providerName)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("%w: provider %q", ErrGlobalModelConflict, providerName)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	provider := &entity.TenantModelProvider{
		ID:           common.GenerateToken(),
		ProviderName: providerName,
		TenantID:     entity.GlobalModelOwnerID,
		UserID:       entity.GlobalModelOwnerID,
	}
	if err := dao.DB.WithContext(ctx).Create(provider).Error; err != nil {
		return nil, err
	}
	return provider, nil
}

// DeleteProvider deletes a global provider with its instances and models,
// unless any of its models is still in use.
func (s *GlobalModelAdminService) DeleteProvider(ctx context.Context, providerID string) (*GlobalModelUsage, error) {
	provider, err := s.provider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return s.deleteUnlessInUse(ctx, provider, "provider_id = ?", provider.ID, func(tx *gorm.DB) error {
		if err := tx.Where("provider_id = ?", provider.ID).Delete(&entity.TenantModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("provider_id = ?", provider.ID).Delete(&entity.TenantModelInstance{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", provider.ID).Delete(&entity.TenantModelProvider{}).Error
	})
}

// deleteUnlessInUse runs remove in one transaction with the usage check of the
// provider's models that match modelFilter, and skips it while any is in use.
func (s *GlobalModelAdminService) deleteUnlessInUse(ctx context.Context, provider *entity.TenantModelProvider, modelFilter, filterValue string, remove func(tx *gorm.DB) error) (*GlobalModelUsage, error) {
	var usage *GlobalModelUsage
	err := dao.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var models []entity.TenantModel
		if err := tx.Where(modelFilter, filterValue).Find(&models).Error; err != nil {
			return err
		}
		found, err := s.usage(ctx, tx, provider, models)
		if err != nil {
			return err
		}
		if !found.IsEmpty() {
			usage = found
			return ErrGlobalModelInUse
		}
		return remove(tx)
	})
	return usage, err
}

// CreateInstance adds an instance to a global provider. Its API key is stored
// encrypted; without RAGFLOW_CONNECTOR_KEY a key cannot be saved.
func (s *GlobalModelAdminService) CreateInstance(ctx context.Context, providerID string, input GlobalInstanceInput) (*entity.TenantModelInstance, error) {
	provider, err := s.provider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.InstanceName)
	if err := ValidateProviderInstanceName(name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGlobalModelInvalidInput, err)
	}
	if _, err := dao.NewTenantModelInstanceDAO().GetByProviderIDAndInstanceName(ctx, dao.DB, provider.ID, name); err == nil {
		return nil, fmt.Errorf("%w: instance %q", ErrGlobalModelConflict, name)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	storedKey, err := common.EncryptModelAPIKey(strings.TrimSpace(NormalizeAPIKey(input.APIKey)))
	if err != nil {
		return nil, err
	}
	extra, err := json.Marshal(map[string]string{
		"region":   strings.TrimSpace(globalValueOrEmpty(input.Region)),
		"base_url": strings.TrimSpace(globalValueOrEmpty(input.BaseURL)),
	})
	if err != nil {
		return nil, err
	}
	instance := &entity.TenantModelInstance{
		ID:           common.GenerateToken(),
		InstanceName: name,
		ProviderID:   provider.ID,
		APIKey:       storedKey,
		Status:       globalStatusActive,
		Extra:        string(extra),
	}
	err = dao.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(instance).Error; err != nil {
			return err
		}
		for _, model := range input.Models {
			if err := s.addModel(ctx, tx, provider, instance, model); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// UpdateInstance changes an instance's key, endpoint, region, or status.
// Setting the status to inactive stops every use of its models at once.
func (s *GlobalModelAdminService) UpdateInstance(ctx context.Context, instanceID string, input GlobalInstanceInput) error {
	instance, _, err := s.instance(ctx, instanceID)
	if err != nil {
		return err
	}
	updates := map[string]interface{}{}
	if input.ClearAPIKey {
		updates["api_key"] = ""
	} else if key := strings.TrimSpace(NormalizeAPIKey(input.APIKey)); key != "" {
		stored, err := common.EncryptModelAPIKey(key)
		if err != nil {
			return err
		}
		updates["api_key"] = stored
	}
	if input.BaseURL != nil || input.Region != nil {
		extra := decodeGlobalInstanceExtra(instance.Extra)
		if input.BaseURL != nil {
			extra["base_url"] = strings.TrimSpace(*input.BaseURL)
		}
		if input.Region != nil {
			extra["region"] = strings.TrimSpace(*input.Region)
		}
		encoded, err := json.Marshal(extra)
		if err != nil {
			return err
		}
		updates["extra"] = string(encoded)
	}
	if input.Status != nil {
		status, err := normalizeGlobalStatus(*input.Status)
		if err != nil {
			return err
		}
		updates["status"] = status
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.DB.WithContext(ctx).Model(&entity.TenantModelInstance{}).Where("id = ?", instance.ID).Updates(updates).Error
}

// DeleteInstance deletes an instance and its models unless one is in use.
func (s *GlobalModelAdminService) DeleteInstance(ctx context.Context, instanceID string) (*GlobalModelUsage, error) {
	instance, provider, err := s.instance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	return s.deleteUnlessInUse(ctx, provider, "instance_id = ?", instance.ID, func(tx *gorm.DB) error {
		if err := tx.Where("instance_id = ?", instance.ID).Delete(&entity.TenantModel{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", instance.ID).Delete(&entity.TenantModelInstance{}).Error
	})
}

// AddModel enables a model on a global instance.
func (s *GlobalModelAdminService) AddModel(ctx context.Context, instanceID string, model CreateInstanceModelInfo) error {
	instance, provider, err := s.instance(ctx, instanceID)
	if err != nil {
		return err
	}
	return s.addModel(ctx, dao.DB, provider, instance, model)
}

func (s *GlobalModelAdminService) addModel(ctx context.Context, db *gorm.DB, provider *entity.TenantModelProvider, instance *entity.TenantModelInstance, model CreateInstanceModelInfo) error {
	model.ModelName = strings.TrimSpace(model.ModelName)
	if model.ModelName == "" || entity.ModelTypeFromStrings(model.ModelTypes) == 0 {
		return fmt.Errorf("%w: a model needs a name and at least one known model type", ErrGlobalModelInvalidInput)
	}
	if err := s.providers.addModelToInstance(ctx, db, provider, instance, model); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("%w: %v", ErrGlobalModelConflict, err)
		}
		return err
	}
	return nil
}

// SetModelStatus enables or disables a global model. Disabling stops every use
// of it at once.
func (s *GlobalModelAdminService) SetModelStatus(ctx context.Context, modelID, status string) error {
	model, err := s.model(ctx, modelID)
	if err != nil {
		return err
	}
	normalized, err := normalizeGlobalStatus(status)
	if err != nil {
		return err
	}
	return dao.DB.WithContext(ctx).Model(&entity.TenantModel{}).Where("id = ?", model.ID).Update("status", normalized).Error
}

// DeleteModel deletes a global model unless it is in use.
func (s *GlobalModelAdminService) DeleteModel(ctx context.Context, modelID string) (*GlobalModelUsage, error) {
	model, err := s.model(ctx, modelID)
	if err != nil {
		return nil, err
	}
	provider, err := s.provider(ctx, model.ProviderID)
	if err != nil {
		return nil, err
	}
	return s.deleteUnlessInUse(ctx, provider, "id = ?", model.ID, func(tx *gorm.DB) error {
		return tx.Where("id = ?", model.ID).Delete(&entity.TenantModel{}).Error
	})
}

// TestConnection checks credentials against a global provider. The API key is
// the submitted one, or the saved key of the instance when none is submitted.
func (s *GlobalModelAdminService) TestConnection(ctx context.Context, providerID string, input GlobalConnectionInput) (common.ErrorCode, error) {
	provider, err := s.provider(ctx, providerID)
	if err != nil {
		return common.CodeNotFound, err
	}
	apiKey := input.APIKey
	baseURL, region := strings.TrimSpace(input.BaseURL), strings.TrimSpace(input.Region)
	if strings.TrimSpace(NormalizeAPIKey(apiKey)) == "" && input.InstanceID != "" {
		instance, owner, err := s.instance(ctx, input.InstanceID)
		if err != nil || owner.ID != provider.ID {
			return common.CodeNotFound, ErrGlobalModelNotFound
		}
		// The instance was loaded through AfterFind, so its key is plaintext.
		if apiKey, err = json.Marshal(instance.APIKey); err != nil {
			return common.CodeServerError, err
		}
		extra := decodeGlobalInstanceExtra(instance.Extra)
		if baseURL == "" {
			baseURL = extra["base_url"]
		}
		if region == "" {
			region = extra["region"]
		}
	}
	return s.providers.CheckConnection(ctx, provider.ProviderName, CheckConnectionRequest{
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Region:    region,
		ModelInfo: input.ModelInfo,
	})
}

// usage counts what still references the given models of one global provider.
// A reference by row ID always counts. A reference by model@instance@provider
// name counts only in tenants without a provider of that name of their own,
// because there the name resolves to the tenant's own model.
func (s *GlobalModelAdminService) usage(ctx context.Context, db *gorm.DB, provider *entity.TenantModelProvider, models []entity.TenantModel) (*GlobalModelUsage, error) {
	usage := &GlobalModelUsage{GlobalDefault: []string{}}
	if len(models) == 0 {
		return usage, nil
	}
	var instances []entity.TenantModelInstance
	if err := db.WithContext(ctx).Select("id", "instance_name").Where("provider_id = ?", provider.ID).Find(&instances).Error; err != nil {
		return nil, err
	}
	instanceNames := make(map[string]string, len(instances))
	for _, instance := range instances {
		instanceNames[instance.ID] = instance.InstanceName
	}
	ids := make([]string, 0, len(models))
	names := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
		names = append(names, model.ModelName+"@"+instanceNames[model.InstanceID]+"@"+provider.ProviderName)
	}
	tenantsWithOwnProvider := db.Session(&gorm.Session{NewDB: true}).WithContext(ctx).
		Model(&entity.TenantModelProvider{}).Select("tenant_id").
		Where("provider_name = ? AND tenant_id <> ?", provider.ProviderName, entity.GlobalModelOwnerID)

	count := func(model interface{}, idColumns, nameColumns []string, target *int64) error {
		clauses := make([]string, 0, len(idColumns)+1)
		args := make([]interface{}, 0, len(idColumns)+len(nameColumns)+1)
		for _, column := range idColumns {
			clauses = append(clauses, column+" IN ?")
			args = append(args, ids)
		}
		if len(nameColumns) > 0 {
			nameClauses := make([]string, 0, len(nameColumns))
			for _, column := range nameColumns {
				nameClauses = append(nameClauses, column+" IN ?")
				args = append(args, names)
			}
			clauses = append(clauses, "(("+strings.Join(nameClauses, " OR ")+") AND tenant_id NOT IN (?))")
			args = append(args, tenantsWithOwnProvider)
		}
		return db.WithContext(ctx).Model(model).Where(strings.Join(clauses, " OR "), args...).Count(target).Error
	}
	contains := func(model interface{}, column string, target *int64) error {
		values := append(append([]string{}, ids...), names...)
		clauses := make([]string, 0, len(values))
		args := make([]interface{}, 0, len(values))
		for _, value := range values {
			clauses = append(clauses, column+" LIKE ?")
			args = append(args, "%"+value+"%")
		}
		return db.WithContext(ctx).Model(model).Where(strings.Join(clauses, " OR "), args...).Count(target).Error
	}

	var skillSpaces, skillSearches int64
	for _, step := range []func() error{
		func() error {
			return count(&entity.Knowledgebase{}, []string{"tenant_embd_id"}, []string{"embd_id"}, &usage.Datasets)
		},
		func() error {
			return count(&entity.Chat{}, []string{"tenant_llm_id", "tenant_rerank_id", "tenant_decision_id"}, []string{"llm_id", "rerank_id"}, &usage.Chats)
		},
		func() error {
			return count(&entity.Memory{}, []string{"tenant_embd_id", "tenant_llm_id"}, []string{"embd_id", "llm_id"}, &usage.Memories)
		},
		func() error {
			return count(&entity.UserTenant{},
				[]string{"tenant_llm_id", "tenant_embd_id", "tenant_asr_id", "tenant_img2txt_id", "tenant_rerank_id", "tenant_tts_id", "tenant_ocr_id", "tenant_decision_id"},
				[]string{"llm_id", "embd_id", "asr_id", "img2txt_id", "rerank_id", "tts_id", "ocr_id", "decision_id"},
				&usage.UserDefaults)
		},
		func() error { return count(&entity.SkillSpace{}, nil, []string{"embd_id", "rerank_id"}, &skillSpaces) },
		func() error {
			return count(&entity.SkillSearchConfig{}, nil, []string{"embd_id", "rerank_id"}, &skillSearches)
		},
		func() error { return contains(&entity.Search{}, "search_config", &usage.Searches) },
		func() error { return contains(&entity.UserCanvas{}, "dsl", &usage.Agents) },
	} {
		if err := step(); err != nil {
			return nil, err
		}
	}
	usage.SkillConfigs = skillSpaces + skillSearches

	defaults, err := globalmodel.LoadDefaults(ctx, db)
	if err != nil {
		return nil, err
	}
	for modelType, modelID := range defaults {
		for _, id := range ids {
			if modelID == id {
				usage.GlobalDefault = append(usage.GlobalDefault, modelType)
			}
		}
	}
	sort.Strings(usage.GlobalDefault)
	return usage, nil
}

// GetDefaults returns the global default model ID per model type name.
func (s *GlobalModelAdminService) GetDefaults(ctx context.Context) (map[string]string, error) {
	return globalmodel.LoadDefaults(ctx, dao.DB)
}

// SetDefaults replaces the global defaults. Each model must be an enabled
// global model of the matching type; an empty ID clears that default.
func (s *GlobalModelAdminService) SetDefaults(ctx context.Context, defaults map[string]string) (map[string]string, error) {
	stored := make(map[string]string, len(defaults))
	for typeName, modelID := range defaults {
		modelType := entity.ModelTypeFromString(typeName)
		if modelType == 0 {
			return nil, fmt.Errorf("%w: unknown model type %q", ErrGlobalModelInvalidInput, typeName)
		}
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		model, err := s.model(ctx, modelID)
		if err != nil {
			return nil, err
		}
		if !entity.ModelType(model.ModelType).Has(modelType) {
			return nil, fmt.Errorf("%w: model %q is not a %s model", ErrGlobalModelInvalidInput, model.ModelName, modelType.String())
		}
		instance, _, err := s.instance(ctx, model.InstanceID)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(model.Status, globalStatusActive) || !strings.EqualFold(instance.Status, globalStatusActive) {
			return nil, fmt.Errorf("%w: model %q is disabled; enable it before making it a default", ErrGlobalModelInvalidInput, model.ModelName)
		}
		stored[modelType.String()] = modelID
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	if err := s.settings.SaveOrCreate(ctx, dao.DB, globalmodel.DefaultsSetting, string(encoded), "admin", "json"); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s *GlobalModelAdminService) provider(ctx context.Context, providerID string) (*entity.TenantModelProvider, error) {
	var provider entity.TenantModelProvider
	err := dao.DB.WithContext(ctx).Where("id = ? AND tenant_id = ?", strings.TrimSpace(providerID), entity.GlobalModelOwnerID).Take(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGlobalModelNotFound
	}
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (s *GlobalModelAdminService) instance(ctx context.Context, instanceID string) (*entity.TenantModelInstance, *entity.TenantModelProvider, error) {
	var instance entity.TenantModelInstance
	err := dao.DB.WithContext(ctx).Where("id = ?", strings.TrimSpace(instanceID)).Take(&instance).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrGlobalModelNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	provider, err := s.provider(ctx, instance.ProviderID)
	if err != nil {
		return nil, nil, err
	}
	return &instance, provider, nil
}

func (s *GlobalModelAdminService) model(ctx context.Context, modelID string) (*entity.TenantModel, error) {
	var model entity.TenantModel
	err := dao.DB.WithContext(ctx).Where("id = ?", strings.TrimSpace(modelID)).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGlobalModelNotFound
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.provider(ctx, model.ProviderID); err != nil {
		return nil, err
	}
	return &model, nil
}

func decodeGlobalInstanceExtra(raw string) map[string]string {
	extra := map[string]string{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &extra)
	}
	return extra
}

func normalizeGlobalStatus(status string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case globalStatusActive:
		return globalStatusActive, nil
	case globalStatusInactive, "disabled":
		return globalStatusInactive, nil
	default:
		return "", fmt.Errorf("%w: status must be active or inactive", ErrGlobalModelInvalidInput)
	}
}

func globalValueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
