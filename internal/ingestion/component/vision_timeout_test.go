package component

import (
	"context"
	"testing"
	"time"
)

func TestResolveVisionChatTimeout(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want time.Duration
	}{
		{name: "unset uses default", env: "", want: visionChatTimeoutDefault},
		{name: "positive seconds", env: "120", want: 120 * time.Second},
		{name: "surrounding spaces", env: " 90 ", want: 90 * time.Second},
		{name: "zero uses default", env: "0", want: visionChatTimeoutDefault},
		{name: "negative uses default", env: "-5", want: visionChatTimeoutDefault},
		{name: "not a number uses default", env: "2m", want: visionChatTimeoutDefault},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envVisionEnhancementTimeoutSeconds, tc.env)
			if got := resolveVisionChatTimeout(); got != tc.want {
				t.Fatalf("resolveVisionChatTimeout() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDefaultVisionChatInvoker_UsesConfiguredTimeout(t *testing.T) {
	t.Setenv(envVisionEnhancementTimeoutSeconds, "120")
	drv := &deadlineCaptureDriver{}
	if _, err := defaultVisionChatInvoker(context.Background(), drv, "m", nil, nil); err != nil {
		t.Fatalf("defaultVisionChatInvoker: %v", err)
	}
	if !drv.hasDeadline {
		t.Fatal("vision chat context must have a deadline")
	}
	if drv.remaining <= visionChatTimeoutDefault || drv.remaining > 120*time.Second {
		t.Fatalf("deadline remaining = %v, want just under 120s", drv.remaining)
	}
}
