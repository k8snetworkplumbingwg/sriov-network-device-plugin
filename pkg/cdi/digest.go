/*
 * SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cdi

import (
	"encoding/json"
	"fmt"
	"strings"

	cdiSpecs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/opencontainers/go-digest"
)

// Digest returns the digest of the given CDI spec.
func Digest(spec cdiSpecs.Spec) digest.Digest {
	digester := digest.Canonical.Digester()
	enc := json.NewEncoder(digester.Hash())
	if err := enc.Encode(spec); err != nil {
		return ""
	}
	return digester.Digest()
}

func extractEncodedFromDigest(digest string) (string, error) {
	var encoded string
	// expects a digest in the <algorithm>:<encoded> format
	if pair := strings.Split(digest, ":"); len(pair) != 2 {
		return "", fmt.Errorf("invalid digest %s", digest)
	} else {
		encoded = pair[1]
	}
	if len(encoded) < 12 {
		return "", fmt.Errorf("invalid digest %s", digest)
	}
	return encoded[0:12], nil
}
