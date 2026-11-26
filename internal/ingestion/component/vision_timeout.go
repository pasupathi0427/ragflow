package component

import (
	"time"

	"ragflow/internal/common"
)

// envVisionEnhancementTimeoutSeconds sets how long one vision-model call may
// take when describing an image or table. Self-hosted models that generate a
// long, detailed description can need more than the default.
const envVisionEnhancementTimeoutSeconds = "VISION_ENHANCEMENT_TIMEOUT_SECONDS"

// resolveVisionChatTimeout returns the per-call vision timeout: the env value
// in seconds when it is a positive whole number, otherwise the default.
func resolveVisionChatTimeout() time.Duration {
	seconds := common.GetEnvInt(envVisionEnhancementTimeoutSeconds, 0)
	if seconds <= 0 {
		return visionChatTimeoutDefault
	}
	return time.Duration(seconds) * time.Second
}
