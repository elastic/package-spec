// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	apiAgentPolicyPath   = "/api/fleet/agent_policies"
	apiPackagePolicyPath = "/api/fleet/package_policies"
)

type agentPolicyRequest struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

type agentPolicyResponse struct {
	Item  *agentPolicyRequest  `json:"item,omitempty"`
	Items []agentPolicyRequest `json:"items,omitempty"`
}

type createPackagePolicyRequest struct {
	Name     string `json:"name"`
	PolicyID string `json:"policy_id"`
	Package  struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"package"`
	Inputs map[string]struct {
		Streams map[string]any `json:"streams,omitempty"`
	} `json:"inputs,omitempty"`
}

type packagePolicyResponse struct {
	Items []json.RawMessage `json:"items"`
}

type Kibana struct {
	Host     string
	Username string
	Password string

	client *http.Client
}

func NewKibanaClient() (*Kibana, error) {
	var client http.Client
	if caCert := elasticPackageGetEnv("CA_CERT"); caCert != "" {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to get system certificate pool: %w", err)
		}
		pem, err := os.ReadFile(caCert)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate \"%s\": %w", caCert, err)
		}
		if ok := certPool.AppendCertsFromPEM(pem); !ok {
			return nil, fmt.Errorf("no certs were appended from \"%s\"", caCert)
		}
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		}
	}

	return &Kibana{
		Host:     elasticPackageGetEnv("KIBANA_HOST"),
		Password: elasticPackageGetEnv("ELASTICSEARCH_PASSWORD"),
		Username: elasticPackageGetEnv("ELASTICSEARCH_USERNAME"),
		client:   &client,
	}, nil
}

func (k *Kibana) CreatePolicyForPackage(name string, version string) (string, error) {
	err := k.deletePackagePolicyForPackage(name)
	if err != nil {
		return "", fmt.Errorf("failed to delete agent policy: %w", err)
	}

	agentPolicy, err := k.createAgentPolicyForPackage(name)
	if err != nil {
		return "", fmt.Errorf("failed to create agent policy: %w", err)
	}

	err = k.createPackagePolicy(agentPolicy.Item.ID, name, version)
	if err != nil {
		return "", fmt.Errorf("failed to create package policy: %w", err)
	}

	return agentPolicy.Item.ID, nil
}

func (k *Kibana) buildPolicyName(packageName string) string {
	return "test-policy-" + packageName
}

func (k *Kibana) deletePackagePolicyForPackage(name string) error {
	policy, err := k.getPolicyForName(name)
	if err != nil {
		return fmt.Errorf("failure while looking for policy to delete: %w", err)
	}
	if policy == nil {
		// Nothing to do.
		return nil
	}

	return k.deletePackagePolicy(policy.ID)
}

func (k *Kibana) deletePackagePolicy(policyID string) error {
	deleteBody := fmt.Sprintf(`{"agentPolicyId": "%s"}`, policyID)
	req, err := k.newRequest(http.MethodPost, apiAgentPolicyPath+"/delete", strings.NewReader(deleteBody))
	if err != nil {
		return err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return fmt.Errorf("deleting policy %q failed with status %d, body: %q", policyID, resp.StatusCode, string(respBody))
	}

	return nil
}

func (k *Kibana) getPolicyForName(name string) (*agentPolicyRequest, error) {
	req, err := k.newRequest(http.MethodGet, apiAgentPolicyPath, nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()
	query.Add("kuery", fmt.Sprintf(`name:"%s"`, k.buildPolicyName(name)))
	req.URL.RawQuery = query.Encode()

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("looking for policy failed with status %d, body: %q", resp.StatusCode, string(respBody))
	}

	var policiesResponse agentPolicyResponse
	err = json.NewDecoder(resp.Body).Decode(&policiesResponse)
	if err != nil {
		return nil, err
	}
	if len(policiesResponse.Items) == 0 {
		return nil, nil
	}
	return &policiesResponse.Items[0], nil
}

func (k *Kibana) createAgentPolicyForPackage(name string) (*agentPolicyResponse, error) {
	agentPolicyRequest := agentPolicyRequest{
		Name:      k.buildPolicyName(name),
		Namespace: "default",
	}

	body, err := json.Marshal(agentPolicyRequest)
	if err != nil {
		return nil, err
	}

	req, err := k.newRequest(http.MethodPost, apiAgentPolicyPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("request failed with status %d, body: %q", resp.StatusCode, string(respBody))
	}

	var agentPolicy agentPolicyResponse
	err = json.NewDecoder(resp.Body).Decode(&agentPolicy)
	if err != nil {
		return nil, err
	}

	return &agentPolicy, nil
}

func (k *Kibana) createPackagePolicy(agentPolicyID string, name string, version string) error {
	var packagePolicyRequest createPackagePolicyRequest
	packagePolicyRequest.PolicyID = agentPolicyID
	packagePolicyRequest.Package.Name = name
	packagePolicyRequest.Package.Version = version

	body, err := json.Marshal(packagePolicyRequest)
	if err != nil {
		return err
	}

	req, err := k.newRequest(http.MethodPost, apiPackagePolicyPath, bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (k *Kibana) newRequest(method string, path string, body io.Reader) (*http.Request, error) {
	url, err := url.JoinPath(k.Host, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(k.Username, k.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("kbn-xsrf", "package-spec")

	return req, nil
}
