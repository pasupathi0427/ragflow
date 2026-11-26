package service

import (
	"reflect"
	"testing"
)

func followUpConversation() []map[string]interface{} {
	return []map[string]interface{}{
		{"role": "system", "content": "You are the HR assistant."},
		{"role": "user", "content": "when is the next holiday for bangalore ?"},
		{"role": "assistant", "content": "Christmas Day on 25 December 2026."},
		{"role": "user", "content": "for other locations ??"},
	}
}

func TestOpenAIChatConversation_DefaultKeepsLastMessage(t *testing.T) {
	t.Setenv(envOpenAIChatKeepHistory, "")
	got := openAIChatConversation(followUpConversation())
	want := []map[string]interface{}{{"role": "user", "content": "for other locations ??"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("openAIChatConversation = %#v, want only the last message", got)
	}
}

func TestOpenAIChatConversation_KeepHistoryKeepsAllMessages(t *testing.T) {
	t.Setenv(envOpenAIChatKeepHistory, "true")
	in := followUpConversation()
	got := openAIChatConversation(in)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("openAIChatConversation = %#v, want the whole conversation", got)
	}
}

func TestOpenAIChatConversation_InvalidValueKeepsDefault(t *testing.T) {
	t.Setenv(envOpenAIChatKeepHistory, "sometimes")
	if got := openAIChatConversation(followUpConversation()); len(got) != 1 {
		t.Fatalf("len = %d, want 1 for an invalid setting", len(got))
	}
}

func TestOpenAIChatConversation_Empty(t *testing.T) {
	t.Setenv(envOpenAIChatKeepHistory, "true")
	if got := openAIChatConversation(nil); len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
	t.Setenv(envOpenAIChatKeepHistory, "")
	if got := openAIChatConversation([]map[string]interface{}{}); len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

// With history kept, the existing filter still drops the client's system
// prompt, so the pipeline gets the user/assistant turns ending in the
// follow-up question.
func TestOpenAIChatConversation_KeepHistoryThenFilter(t *testing.T) {
	t.Setenv(envOpenAIChatKeepHistory, "true")
	svc := &OpenAIChatService{}
	got := svc.filterMessages(openAIChatConversation(followUpConversation()))
	want := []map[string]interface{}{
		{"role": "user", "content": "when is the next holiday for bangalore ?"},
		{"role": "assistant", "content": "Christmas Day on 25 December 2026."},
		{"role": "user", "content": "for other locations ??"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterMessages(conversation) = %#v, want %#v", got, want)
	}
}
