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

package common

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// modelAPIKeyPrefix marks a model API key stored encrypted with
// RAGFLOW_CONNECTOR_KEY. Only global model provider keys are stored this way.
const modelAPIKeyPrefix = "enc:model:v1:"

// MaxStoredModelAPIKeyLength is the width of tenant_model_instance.api_key.
const MaxStoredModelAPIKeyLength = 512

// Errors returned by the model API key helpers. They never carry the key.
var (
	ErrModelAPIKeyEncryptionUnavailable = errors.New(EnvRAGFlowConnectorKey + " must be set to store global model API keys")
	ErrModelAPIKeyTooLong               = errors.New("API key is too long to store encrypted")
	ErrModelAPIKeyDecrypt               = errors.New("cannot decrypt model API key: wrong " + EnvRAGFlowConnectorKey + " or corrupted value")
)

// IsEncryptedModelAPIKey reports whether a stored key is in encrypted form.
func IsEncryptedModelAPIKey(stored string) bool {
	return strings.HasPrefix(stored, modelAPIKeyPrefix)
}

// EncryptModelAPIKey encrypts a plaintext API key for storage. An empty key
// stays empty, and an already encrypted key is returned unchanged.
func EncryptModelAPIKey(plaintext string) (string, error) {
	if plaintext == "" || IsEncryptedModelAPIKey(plaintext) {
		return plaintext, nil
	}
	key, err := ConnectorKey()
	if err != nil {
		return "", err
	}
	if key == nil {
		return "", ErrModelAPIKeyEncryptionUnavailable
	}
	gcm, err := connectorCredentialsCipher(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("cannot generate model API key nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	stored := modelAPIKeyPrefix + base64.StdEncoding.EncodeToString(sealed)
	if len(stored) > MaxStoredModelAPIKeyLength {
		return "", ErrModelAPIKeyTooLong
	}
	return stored, nil
}

// DecryptModelAPIKey returns the plaintext of a stored key. A key that is not
// in encrypted form is returned unchanged.
func DecryptModelAPIKey(stored string) (string, error) {
	if !IsEncryptedModelAPIKey(stored) {
		return stored, nil
	}
	key, err := ConnectorKey()
	if err != nil {
		return "", err
	}
	if key == nil {
		return "", ErrModelAPIKeyEncryptionUnavailable
	}
	sealed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, modelAPIKeyPrefix))
	if err != nil {
		return "", ErrModelAPIKeyDecrypt
	}
	gcm, err := connectorCredentialsCipher(key)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize()+gcm.Overhead() {
		return "", ErrModelAPIKeyDecrypt
	}
	plaintext, err := gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrModelAPIKeyDecrypt
	}
	return string(plaintext), nil
}

// MaskModelAPIKey shows only the last four characters of a plaintext key.
func MaskModelAPIKey(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	if len(plaintext) <= 4 {
		return "••••"
	}
	return "••••" + plaintext[len(plaintext)-4:]
}
