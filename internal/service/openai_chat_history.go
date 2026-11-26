package service

import "ragflow/internal/common"

// envOpenAIChatKeepHistory makes the OpenAI-compatible chat endpoint keep the
// whole conversation the client sends instead of only its last message, so a
// follow-up question is answered with the earlier turns as context. Off by
// default. The chat pipeline still trims the conversation to the model's
// context window.
const envOpenAIChatKeepHistory = "OPENAI_CHAT_KEEP_HISTORY"

// openAIChatConversation returns the messages the endpoint passes on: all of
// them when OPENAI_CHAT_KEEP_HISTORY is on, otherwise only the last one.
func openAIChatConversation(messages []map[string]interface{}) []map[string]interface{} {
	if len(messages) == 0 || common.GetEnvBool(envOpenAIChatKeepHistory, false) {
		return messages
	}
	return messages[len(messages)-1:]
}
