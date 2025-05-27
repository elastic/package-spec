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

	apiGetSloPath                     = "/s/%s/api/observability/slos"
	apiGetDashboardPath               = "/api/dashboards/dashboard"
	apiGetDetecionRulePath            = "/api/detection_engine/rules"
	apiLoadPrebuiltDetectionRulesPath = "/api/detection_engine/rules/prepackaged"
	apiSavedObjects                   = "/api/saved_objects"

	defaultSpace = "default"
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
	Inputs map[string]packagePolicyInput `json:"inputs,omitempty"`
}

type packagePolicyInput struct {
	Streams map[string]packagePolicyStream `json:"streams,omitempty"`
}

type packagePolicyStream struct {
	Vars map[string]any `json:"vars,omitempty"`
}

type dashboardResponse struct {
	Item json.RawMessage `json:"item"`
}

type sloResponse struct {
	Description string `json:"description"`
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
}

type detectionRuleResponse struct {
	Description string `json:"description"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
}

// Kibana is a kibana client.
type Kibana struct {
	Host     string
	Username string
	Password string

	client *http.Client
}

// NewKibanaClient creates a new Kibana client using environment variables for its initialization.
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

// CreatePolicyForPackage creates a new policy for a package.
func (k *Kibana) CreatePolicyForPackage(name string, version string) (string, error) {
	err := k.deletePackagePolicyForPackage(name)
	if err != nil {
		return "", fmt.Errorf("failed to delete agent policy: %w", err)
	}

	agentPolicy, err := k.createAgentPolicyForPackage(name)
	if err != nil {
		return "", fmt.Errorf("failed to create agent policy: %w", err)
	}

	err = k.createPackagePolicy(agentPolicy.Item.ID, name, version, "", "", "", "")
	if err != nil {
		return "", fmt.Errorf("failed to create package policy: %w", err)
	}

	return agentPolicy.Item.ID, nil
}

// CreatePolicyForPackageInputAndDataset creates a policy for a package with a custom dataset.
// XXX: Pass the path of the manifest and read input name and type from there.
func (k *Kibana) CreatePolicyForPackageInputAndDataset(name, version, templateName, inputName, inputType, dataset string) (string, error) {
	err := k.deletePackagePolicyForPackage(name)
	if err != nil {
		return "", fmt.Errorf("failed to delete agent policy: %w", err)
	}

	agentPolicy, err := k.createAgentPolicyForPackage(name)
	if err != nil {
		return "", fmt.Errorf("failed to create agent policy: %w", err)
	}

	err = k.createPackagePolicy(agentPolicy.Item.ID, name, version, templateName, inputName, inputType, dataset)
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
		return nil, fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var agentPolicy agentPolicyResponse
	err = json.NewDecoder(resp.Body).Decode(&agentPolicy)
	if err != nil {
		return nil, err
	}

	return &agentPolicy, nil
}

func (k *Kibana) createPackagePolicy(agentPolicyID, name, version, templateName, inputName, inputType, dataset string) error {
	var packagePolicyRequest createPackagePolicyRequest
	packagePolicyRequest.Name = name + "-test-1"
	packagePolicyRequest.PolicyID = agentPolicyID
	packagePolicyRequest.Package.Name = name
	packagePolicyRequest.Package.Version = version

	if templateName != "" && inputName != "" && inputType != "" {
		policyInputName := templateName + "-" + inputType
		policyStreamName := name + "." + inputName
		vars := make(map[string]any)
		if dataset != "" {
			vars["data_stream.dataset"] = dataset
		}
		packagePolicyRequest.Inputs = map[string]packagePolicyInput{
			policyInputName: packagePolicyInput{
				Streams: map[string]packagePolicyStream{
					policyStreamName: packagePolicyStream{
						Vars: vars,
					},
				},
			},
		}
	}

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
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// MustExistSLO checks if an SLO with the given ID exists.
func (k *Kibana) MustExistSLO(sloID string) error {
	_, err := k.getSLO(sloID, defaultSpace)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kibana) getSLO(sloID, space string) (*sloResponse, error) {
	apiPath := fmt.Sprintf(apiGetSloPath, space)
	apiPath, err := url.JoinPath(apiPath, sloID)
	if err != nil {
		return nil, err
	}
	req, err := k.newRequest(http.MethodGet, apiPath, nil)
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
		return nil, fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var slo sloResponse
	err = json.NewDecoder(resp.Body).Decode(&slo)
	if err != nil {
		return nil, err
	}

	return &slo, nil
}

// MustExistDashboard checks if a dashboard with the given ID exists.
func (k *Kibana) MustExistDashboard(dashboardID string) error {
	_, err := k.getDashboard(dashboardID)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kibana) getDashboard(dashboardID string) (*dashboardResponse, error) {
	apiPath, err := url.JoinPath(apiGetDashboardPath, dashboardID)
	if err != nil {
		return nil, err
	}
	req, err := k.newRequest(http.MethodGet, apiPath, nil)
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
		return nil, fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var dashboard dashboardResponse
	err = json.NewDecoder(resp.Body).Decode(&dashboard)
	if err != nil {
		return nil, err
	}

	return &dashboard, nil
}

// MustExistDetectionRule checks if a detection rule with the given ID exists.
func (k *Kibana) MustExistDetectionRule(detectionRuleID string) error {
	_, err := k.getDetectionRuleID(detectionRuleID)
	if err != nil {
		return err
	}
	return nil
}

// LoadPrebuiltDetectionRules retrieves rule statuses and loads Elastic prebuilt detection rules.
func (k *Kibana) LoadPrebuiltDetectionRules() error {
	req, err := k.newRequest(http.MethodPut, apiLoadPrebuiltDetectionRulesPath, nil)
	if err != nil {
		return err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil

}

func (k *Kibana) getDetectionRuleID(detectionRuleID string) (*detectionRuleResponse, error) {
	apiPath, err := url.Parse(apiGetDetecionRulePath)
	if err != nil {
		return nil, err
	}
	req, err := k.newRequest(http.MethodGet, apiPath.String(), nil)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"rule_id": detectionRuleID,
	}
	req = addRequestParams(req, params)

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
		return nil, fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var detectionRule detectionRuleResponse
	err = json.NewDecoder(resp.Body).Decode(&detectionRule)
	if err != nil {
		return nil, err
	}

	return &detectionRule, nil
}

// MustExistSavedObject checks if a saved object with the given type and id exists.
func (k *Kibana) MustExistSavedObject(soType, id string) error {
	apiPath, err := url.JoinPath(apiSavedObjects, soType, id)
	if err != nil {
		return err
	}
	req, err := k.newRequest(http.MethodGet, apiPath, nil)
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
		return fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (k *Kibana) newRequest(method string, path string, body io.Reader) (*http.Request, error) {
	urlPath, err := url.JoinPath(k.Host, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, urlPath, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(k.Username, k.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("kbn-xsrf", "package-spec")

	return req, nil

}

func addRequestParams(request *http.Request, params map[string]string) *http.Request {
	values := request.URL.Query()
	for key, value := range params {
		values.Add(key, value)
	}
	request.URL.RawQuery = values.Encode()
	return request
}
