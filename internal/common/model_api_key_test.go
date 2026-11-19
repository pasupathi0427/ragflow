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
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func setModelKeyTestKey(t *testing.T, raw string) {
	t.Helper()
	t.Setenv(EnvRAGFlowConnectorKey, base64.StdEncoding.EncodeToString([]byte(raw)))
}

func TestModelAPIKeyRoundTrip(t *testing.T) {
	setModelKeyTestKey(t, "0123456789abcdef0123456789abcdef")
	stored, err := EncryptModelAPIKey("sk-secret-value")
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncryptedModelAPIKey(stored) || strings.Contains(stored, "secret") {
		t.Fatalf("stored = %q, want an encrypted value", stored)
	}
	again, err := EncryptModelAPIKey("sk-secret-value")
	if err != nil || again == stored {
		t.Fatalf("two encryptions match (%v): nonce reuse", err)
	}
	if reencrypted, err := EncryptModelAPIKey(stored); err != nil || reencrypted != stored {
		t.Fatalf("encrypting a stored value changed it (%v)", err)
	}
	plaintext, err := DecryptModelAPIKey(stored)
	if err != nil || plaintext != "sk-secret-value" {
		t.Fatalf("decrypt = %q (%v)", plaintext, err)
	}
}

func TestModelAPIKeyPlainValuesPassThrough(t *testing.T) {
	t.Setenv(EnvRAGFlowConnectorKey, "")
	for _, value := range []string{"", "sk-user-key"} {
		if got, err := DecryptModelAPIKey(value); err != nil || got != value {
			t.Fatalf("decrypt(%q) = %q (%v)", value, got, err)
		}
	}
	if got, err := EncryptModelAPIKey(""); err != nil || got != "" {
		t.Fatalf("encrypt(empty) = %q (%v)", got, err)
	}
}

func TestModelAPIKeyNeedsConnectorKey(t *testing.T) {
	t.Setenv(EnvRAGFlowConnectorKey, "")
	if _, err := EncryptModelAPIKey("sk-secret"); !errors.Is(err, ErrModelAPIKeyEncryptionUnavailable) {
		t.Fatalf("encrypt without key: %v", err)
	}
	if _, err := DecryptModelAPIKey(modelAPIKeyPrefix + "AAAA"); !errors.Is(err, ErrModelAPIKeyEncryptionUnavailable) {
		t.Fatalf("decrypt without key: %v", err)
	}
}

func TestModelAPIKeyWrongKeyOrTamperFails(t *testing.T) {
	setModelKeyTestKey(t, "0123456789abcdef0123456789abcdef")
	stored, err := EncryptModelAPIKey("sk-secret")
	if err != nil {
		t.Fatal(err)
	}
	tampered := stored[:len(stored)-4] + "AAAA"
	if _, err := DecryptModelAPIKey(tampered); !errors.Is(err, ErrModelAPIKeyDecrypt) {
		t.Fatalf("tampered value: %v", err)
	}
	setModelKeyTestKey(t, "fedcba9876543210fedcba9876543210")
	if _, err := DecryptModelAPIKey(stored); !errors.Is(err, ErrModelAPIKeyDecrypt) {
		t.Fatalf("wrong key: %v", err)
	}
}

func TestModelAPIKeyTooLong(t *testing.T) {
	setModelKeyTestKey(t, "0123456789abcdef0123456789abcdef")
	if _, err := EncryptModelAPIKey(strings.Repeat("k", MaxStoredModelAPIKeyLength)); !errors.Is(err, ErrModelAPIKeyTooLong) {
		t.Fatalf("over-long key: %v", err)
	}
}

func TestMaskModelAPIKey(t *testing.T) {
	for in, want := range map[string]string{"": "", "abc": "••••", "sk-12345678": "••••5678"} {
		if got := MaskModelAPIKey(in); got != want {
			t.Fatalf("mask(%q) = %q, want %q", in, got, want)
		}
	}
}
