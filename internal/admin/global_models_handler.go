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

package admin

import (
	"errors"
	"fmt"
	"strings"

	"ragflow/internal/common"
	"ragflow/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// globalModelsHandler serves the admin console endpoints for global model
// providers: providers, instances, and models every user may call.
type globalModelsHandler struct {
	service *service.GlobalModelAdminService
}

// RegisterGlobalModelRoutes adds the global model endpoints to the admin
// console's authenticated superuser group.
func RegisterGlobalModelRoutes(protected *gin.RouterGroup) {
	h := &globalModelsHandler{service: service.NewGlobalModelAdminService()}
	group := protected.Group("/global-models")
	group.GET("", h.list)
	group.GET("/catalog", h.catalog)
	group.GET("/catalog/:provider_name/models", h.catalogModels)
	group.POST("/providers", h.addProvider)
	group.DELETE("/providers/:provider_id", h.deleteProvider)
	group.POST("/providers/:provider_id/instances", h.createInstance)
	group.POST("/providers/:provider_id/connection", h.testConnection)
	group.PUT("/instances/:instance_id", h.updateInstance)
	group.DELETE("/instances/:instance_id", h.deleteInstance)
	group.POST("/instances/:instance_id/models", h.addModel)
	group.PUT("/models/:model_id/status", h.setModelStatus)
	group.DELETE("/models/:model_id", h.deleteModel)
	group.GET("/defaults", h.getDefaults)
	group.PUT("/defaults", h.setDefaults)
}

func respondGlobalModelError(c *gin.Context, err error, usage *service.GlobalModelUsage) {
	switch {
	case errors.Is(err, service.ErrGlobalModelInvalidInput),
		errors.Is(err, common.ErrModelAPIKeyEncryptionUnavailable),
		errors.Is(err, common.ErrModelAPIKeyTooLong):
		common.RespondError(c, common.CodeBadRequest, err.Error())
	case errors.Is(err, service.ErrGlobalModelNotFound):
		common.RespondError(c, common.CodeNotFound, err.Error())
	case errors.Is(err, service.ErrGlobalModelConflict):
		common.RespondError(c, common.CodeConflict, err.Error())
	case errors.Is(err, service.ErrGlobalModelInUse):
		common.Respond(c, 0, common.CodeConflict, usage, describeGlobalModelUsage(usage))
	default:
		common.Error("global model admin request failed", err)
		common.RespondError(c, common.CodeServerError, "")
	}
}

func auditGlobalModelChange(c *gin.Context, action string, fields ...zap.Field) {
	fields = append(fields, zap.String("admin_email", c.GetString("email")), zap.String("action", action))
	common.Info("global model configuration changed", fields...)
}

func (h *globalModelsHandler) list(c *gin.Context) {
	providers, err := h.service.List(c.Request.Context())
	if err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	common.RespondSuccessList(c, providers, "", int64(len(providers)))
}

func (h *globalModelsHandler) catalog(c *gin.Context) {
	providers, err := h.service.Catalog()
	if err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	common.RespondSuccessList(c, providers, "", int64(len(providers)))
}

func (h *globalModelsHandler) catalogModels(c *gin.Context) {
	models, err := h.service.CatalogModels(c.Param("provider_name"))
	if err != nil {
		respondGlobalModelError(c, service.ErrGlobalModelNotFound, nil)
		return
	}
	common.RespondSuccessList(c, models, "", int64(len(models)))
}

type addGlobalProviderRequest struct {
	ProviderName string `json:"provider_name"`
}

func (h *globalModelsHandler) addProvider(c *gin.Context) {
	var req addGlobalProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	provider, err := h.service.AddProvider(c.Request.Context(), req.ProviderName)
	if err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	auditGlobalModelChange(c, "add_provider", zap.String("provider", provider.ProviderName))
	common.RespondSuccess(c, map[string]string{"id": provider.ID, "provider_name": provider.ProviderName}, "")
}

func (h *globalModelsHandler) deleteProvider(c *gin.Context) {
	providerID := c.Param("provider_id")
	usage, err := h.service.DeleteProvider(c.Request.Context(), providerID)
	if err != nil {
		respondGlobalModelError(c, err, usage)
		return
	}
	auditGlobalModelChange(c, "delete_provider", zap.String("provider_id", providerID))
	common.RespondSuccess(c, nil, "")
}

func (h *globalModelsHandler) createInstance(c *gin.Context) {
	var input service.GlobalInstanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	instance, err := h.service.CreateInstance(c.Request.Context(), c.Param("provider_id"), input)
	if err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	auditGlobalModelChange(c, "create_instance", zap.String("instance_id", instance.ID))
	common.RespondSuccess(c, map[string]string{"id": instance.ID, "instance_name": instance.InstanceName}, "")
}

func (h *globalModelsHandler) testConnection(c *gin.Context) {
	var input service.GlobalConnectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	code, err := h.service.TestConnection(c.Request.Context(), c.Param("provider_id"), input)
	if err != nil {
		if code == common.CodeSuccess {
			code = common.CodeServerError
		}
		common.RespondError(c, code, err.Error())
		return
	}
	common.RespondSuccess(c, nil, "Connection succeeded")
}

func (h *globalModelsHandler) updateInstance(c *gin.Context) {
	var input service.GlobalInstanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	instanceID := c.Param("instance_id")
	if err := h.service.UpdateInstance(c.Request.Context(), instanceID, input); err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	fields := []zap.Field{zap.String("instance_id", instanceID), zap.Bool("api_key_changed", len(input.APIKey) > 0 || input.ClearAPIKey)}
	if input.Status != nil {
		fields = append(fields, zap.String("status", *input.Status))
	}
	auditGlobalModelChange(c, "update_instance", fields...)
	common.RespondSuccess(c, nil, "")
}

func (h *globalModelsHandler) deleteInstance(c *gin.Context) {
	instanceID := c.Param("instance_id")
	usage, err := h.service.DeleteInstance(c.Request.Context(), instanceID)
	if err != nil {
		respondGlobalModelError(c, err, usage)
		return
	}
	auditGlobalModelChange(c, "delete_instance", zap.String("instance_id", instanceID))
	common.RespondSuccess(c, nil, "")
}

func (h *globalModelsHandler) addModel(c *gin.Context) {
	var model service.CreateInstanceModelInfo
	if err := c.ShouldBindJSON(&model); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	instanceID := c.Param("instance_id")
	if err := h.service.AddModel(c.Request.Context(), instanceID, model); err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	auditGlobalModelChange(c, "add_model", zap.String("instance_id", instanceID), zap.String("model", model.ModelName))
	common.RespondSuccess(c, nil, "")
}

type setGlobalModelStatusRequest struct {
	Status string `json:"status"`
}

func (h *globalModelsHandler) setModelStatus(c *gin.Context) {
	var req setGlobalModelStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	modelID := c.Param("model_id")
	if err := h.service.SetModelStatus(c.Request.Context(), modelID, req.Status); err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	auditGlobalModelChange(c, "set_model_status", zap.String("model_id", modelID), zap.String("status", req.Status))
	common.RespondSuccess(c, nil, "")
}

func (h *globalModelsHandler) deleteModel(c *gin.Context) {
	modelID := c.Param("model_id")
	usage, err := h.service.DeleteModel(c.Request.Context(), modelID)
	if err != nil {
		respondGlobalModelError(c, err, usage)
		return
	}
	auditGlobalModelChange(c, "delete_model", zap.String("model_id", modelID))
	common.RespondSuccess(c, nil, "")
}

func (h *globalModelsHandler) getDefaults(c *gin.Context) {
	defaults, err := h.service.GetDefaults(c.Request.Context())
	if err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	common.RespondSuccess(c, defaults, "")
}

func (h *globalModelsHandler) setDefaults(c *gin.Context) {
	var defaults map[string]string
	if err := c.ShouldBindJSON(&defaults); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	stored, err := h.service.SetDefaults(c.Request.Context(), defaults)
	if err != nil {
		respondGlobalModelError(c, err, nil)
		return
	}
	auditGlobalModelChange(c, "set_defaults", zap.Any("defaults", stored))
	common.RespondSuccess(c, stored, "")
}

// describeGlobalModelUsage says what blocks a delete, e.g. "Still in use by 2
// datasets and 1 chat. Disable it instead, or move them to another model first."
func describeGlobalModelUsage(usage *service.GlobalModelUsage) string {
	if usage == nil {
		return service.ErrGlobalModelInUse.Error()
	}
	parts := []string{}
	add := func(count int64, singular, plural string) {
		switch {
		case count == 1:
			parts = append(parts, "1 "+singular)
		case count > 1:
			parts = append(parts, fmt.Sprintf("%d %s", count, plural))
		}
	}
	add(usage.Datasets, "dataset", "datasets")
	add(usage.Chats, "chat", "chats")
	add(usage.Memories, "memory", "memories")
	add(usage.Agents, "agent", "agents")
	add(usage.Searches, "search app", "search apps")
	add(usage.SkillConfigs, "skill configuration", "skill configurations")
	add(usage.UserDefaults, "user default", "user defaults")
	if len(usage.GlobalDefault) > 0 {
		parts = append(parts, "the global default for "+strings.Join(usage.GlobalDefault, ", "))
	}
	used := strings.Join(parts, ", ")
	if i := strings.LastIndex(used, ", "); i >= 0 {
		used = used[:i] + " and " + used[i+2:]
	}
	return "Still in use by " + used + ". Disable it instead, or move them to another model first."
}
