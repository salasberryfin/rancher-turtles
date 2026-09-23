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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// resolveGKEKubernetesVersion reads required inputs from environment variables
// and delegates to resolveGKEVersion. Returns "" on any failure so the caller
// can fall back to the static value in operator.yaml.
func resolveGKEKubernetesVersion(ctx context.Context) string {
	minor := os.Getenv("GCP_GKE_KUBERNETES_MINOR")
	credsB64 := os.Getenv("CAPG_ENCODED_CREDS")
	project := os.Getenv("GCP_PROJECT")
	region := os.Getenv("GCP_REGION")

	if minor == "" || credsB64 == "" || project == "" || region == "" {
		ginkgo.GinkgoWriter.Println("resolveGKEKubernetesVersion: required env vars missing, using static GCP_GKE_KUBERNETES_VERSION")
		return ""
	}

	credsJSON, err := base64.StdEncoding.DecodeString(credsB64)
	if err != nil {
		ginkgo.GinkgoWriter.Printf("resolveGKEKubernetesVersion: failed to decode CAPG_ENCODED_CREDS: %v\n", err)
		return ""
	}

	version, err := resolveGKEVersion(ctx, minor, credsJSON, project, region)
	if err != nil {
		ginkgo.GinkgoWriter.Printf("resolveGKEKubernetesVersion: %v, using static GCP_GKE_KUBERNETES_VERSION\n", err)
		return ""
	}

	ginkgo.GinkgoWriter.Printf("resolveGKEKubernetesVersion: resolved GCP_GKE_KUBERNETES_VERSION=%s\n", version)
	return version
}

// resolveGKEVersion queries the GCP Container API for the latest patch version
// of the given minor in the REGULAR release channel. Exported for direct use
// in tests without requiring a Ginkgo runner.
func resolveGKEVersion(ctx context.Context, minor string, credsJSON []byte, project, region string) (string, error) {
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("parsing credentials: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, creds.TokenSource)
	url := fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/%s/serverConfig", project, region)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("querying GCP Container API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GCP Container API returned HTTP %d", resp.StatusCode)
	}

	var config struct {
		Channels []struct {
			Channel       string   `json:"channel"`
			ValidVersions []string `json:"validVersions"`
		} `json:"channels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return "", fmt.Errorf("decoding GCP Container API response: %w", err)
	}

	var candidates []string
	for _, ch := range config.Channels {
		if ch.Channel != "REGULAR" {
			continue
		}
		for _, v := range ch.ValidVersions {
			bare := strings.SplitN(v, "-", 2)[0] // strip -gke.NNNN suffix
			if strings.HasPrefix(bare, minor+".") {
				candidates = append(candidates, bare)
			}
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no version found for minor %s in REGULAR channel", minor)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return gkePatchNumber(candidates[i]) < gkePatchNumber(candidates[j])
	})

	return "v" + candidates[len(candidates)-1], nil
}

func gkePatchNumber(version string) int {
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return 0
	}
	n, _ := strconv.Atoi(parts[2])
	return n
}
