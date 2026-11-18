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
	"errors"
	"net/http"

	"ragflow/internal/common"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MessageFeatureDisabled is returned when a user's profile turns a feature off.
const MessageFeatureDisabled = "This feature is disabled for your account. Contact your administrator."

// RequireFeature blocks the request with 403 unless the signed-in user's
// profile enables the feature. It must run after the API auth middleware,
// which sets user_id.
func RequireFeature(feature Feature) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			common.Respond(c, http.StatusUnauthorized, common.CodeUnauthorized, nil, "")
			c.Abort()
			return
		}
		enabled, err := DefaultService().IsEnabled(c.Request.Context(), userID, feature)
		if err != nil {
			common.Error("access profile lookup failed", err, zap.String("user_id", userID))
			common.Respond(c, http.StatusInternalServerError, common.CodeServerError, nil, "")
			c.Abort()
			return
		}
		if !enabled {
			common.Respond(c, http.StatusForbidden, common.CodeForbidden, nil, MessageFeatureDisabled)
			c.Abort()
			return
		}
		c.Next()
	}
}

// MyFeatures returns the signed-in user's feature switches.
func MyFeatures(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		common.Respond(c, http.StatusUnauthorized, common.CodeUnauthorized, nil, "")
		return
	}
	features, err := DefaultService().FeaturesForUser(c.Request.Context(), userID)
	if err != nil {
		common.Error("access profile lookup failed", err, zap.String("user_id", userID))
		common.RespondError(c, common.CodeServerError, "")
		return
	}
	common.RespondSuccess(c, features, "")
}

// AdminHandler serves the admin console endpoints for access profiles.
type AdminHandler struct {
	service *Service
}

// RegisterAdminRoutes adds the access profile endpoints to the admin console's
// authenticated superuser group.
func RegisterAdminRoutes(protected *gin.RouterGroup) {
	h := &AdminHandler{service: DefaultService()}
	protected.GET("/access-profiles", h.List)
	protected.POST("/access-profiles", h.Create)
	protected.GET("/access-profiles/:profile_id", h.Get)
	protected.PUT("/access-profiles/:profile_id", h.Update)
	protected.DELETE("/access-profiles/:profile_id", h.Delete)
	protected.GET("/access-profile-assignments", h.ListAssignments)
	protected.PUT("/users/:username/access-profile", h.AssignUser)
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		common.RespondError(c, common.CodeBadRequest, err.Error())
	case errors.Is(err, ErrProfileNotFound), errors.Is(err, ErrUserNotFound):
		common.RespondError(c, common.CodeNotFound, err.Error())
	case errors.Is(err, ErrProfileNameTaken):
		common.RespondError(c, common.CodeConflict, err.Error())
	case errors.Is(err, ErrDefaultProfileReadOnly):
		common.RespondError(c, common.CodeForbidden, err.Error())
	default:
		common.Error("access profile admin request failed", err)
		common.RespondError(c, common.CodeServerError, "")
	}
}

func auditAdminChange(c *gin.Context, action string, fields ...zap.Field) {
	fields = append(fields, zap.String("admin_email", c.GetString("email")), zap.String("action", action))
	common.Info("access profile changed", fields...)
}

// List returns every profile.
func (h *AdminHandler) List(c *gin.Context) {
	profiles, err := h.service.List(c.Request.Context())
	if err != nil {
		respondServiceError(c, err)
		return
	}
	common.RespondSuccessList(c, profiles, "", int64(len(profiles)))
}

// Get returns one profile.
func (h *AdminHandler) Get(c *gin.Context) {
	profile, err := h.service.Get(c.Request.Context(), c.Param("profile_id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	common.RespondSuccess(c, profile, "")
}

// Create adds a profile.
func (h *AdminHandler) Create(c *gin.Context) {
	common.StartOperationLog(c, "create_access_profile")
	defer common.FinishOperationLog(c)

	var input ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	profile, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	common.SetOperationResource(c, "access_profile_id", profile.ID)
	auditAdminChange(c, "create", zap.String("profile_id", profile.ID), zap.Any("features", profile.Features))
	common.RespondSuccess(c, profile, "")
}

// Update replaces a profile's name, description, and switches.
func (h *AdminHandler) Update(c *gin.Context) {
	common.StartOperationLog(c, "update_access_profile")
	defer common.FinishOperationLog(c)

	profileID := c.Param("profile_id")
	common.SetOperationResource(c, "access_profile_id", profileID)
	var input ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	profile, err := h.service.Update(c.Request.Context(), profileID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	auditAdminChange(c, "update", zap.String("profile_id", profile.ID), zap.Any("features", profile.Features))
	common.RespondSuccess(c, profile, "")
}

// Delete removes a profile; its users fall back to the default profile.
func (h *AdminHandler) Delete(c *gin.Context) {
	common.StartOperationLog(c, "delete_access_profile")
	defer common.FinishOperationLog(c)

	profileID := c.Param("profile_id")
	common.SetOperationResource(c, "access_profile_id", profileID)
	if err := h.service.Delete(c.Request.Context(), profileID); err != nil {
		respondServiceError(c, err)
		return
	}
	auditAdminChange(c, "delete", zap.String("profile_id", profileID))
	common.RespondSuccess(c, nil, "")
}

// ListAssignments returns every user with an explicit profile.
func (h *AdminHandler) ListAssignments(c *gin.Context) {
	assignments, err := h.service.ListAssignments(c.Request.Context())
	if err != nil {
		respondServiceError(c, err)
		return
	}
	common.RespondSuccessList(c, assignments, "", int64(len(assignments)))
}

type assignUserRequest struct {
	ProfileID string `json:"profile_id"`
}

// AssignUser sets a user's profile.
func (h *AdminHandler) AssignUser(c *gin.Context) {
	common.StartOperationLog(c, "assign_access_profile")
	defer common.FinishOperationLog(c)

	username := c.Param("username")
	common.SetOperationResource(c, "target_username", username)
	var req assignUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespondError(c, common.CodeBadRequest, "Invalid request body")
		return
	}
	assignment, err := h.service.AssignUser(c.Request.Context(), username, req.ProfileID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	auditAdminChange(c, "assign", zap.String("user_id", assignment.UserID), zap.String("profile_id", assignment.ProfileID))
	common.RespondSuccess(c, assignment, "")
}
