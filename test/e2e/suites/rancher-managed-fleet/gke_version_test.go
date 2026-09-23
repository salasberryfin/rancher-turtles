//go:build e2e
// +build e2e

/*
Copyright © 2023 - 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rancher_managed_fleet

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
)

// TestResolveGKEVersion validates the GKE version resolution logic against the
// real GCP Container API. It requires the same credentials used by the e2e suite.
//
// Run it locally without triggering the full e2e environment:
//
//	CAPG_ENCODED_CREDS=<base64-sa-json> \
//	GCP_PROJECT=<project> \
//	GCP_REGION=europe-west2 \
//	GCP_GKE_KUBERNETES_MINOR=1.35 \
//	go test -tags=e2e -run TestResolveGKEVersion -v \
//	  ./test/e2e/suites/rancher-managed-fleet/
func TestResolveGKEVersion(t *testing.T) {
	credsB64 := os.Getenv("CAPG_ENCODED_CREDS")
	if credsB64 == "" {
		t.Skip("CAPG_ENCODED_CREDS not set")
	}

	credsJSON, err := base64.StdEncoding.DecodeString(credsB64)
	if err != nil {
		t.Fatalf("failed to decode CAPG_ENCODED_CREDS: %v", err)
	}

	minor := os.Getenv("GCP_GKE_KUBERNETES_MINOR")
	if minor == "" {
		minor = "1.35"
	}

	version, err := resolveGKEVersion(
		context.Background(),
		minor,
		credsJSON,
		os.Getenv("GCP_PROJECT"),
		os.Getenv("GCP_REGION"),
	)
	if err != nil {
		t.Fatalf("resolveGKEVersion: %v", err)
	}

	t.Logf("resolved GKE version for minor %s: %s", minor, version)
}
