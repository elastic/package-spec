// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
)

const (
	apiAgentPolicyPath   = "/api/fleet/agent_policies"
	apiPackagePolicyPath = "/api/fleet/package_policies"

	apiGetSloTemplatePath             = "/s/%s/api/observability/slo_templates"
	apiExportSavedObjectPath          = "/api/saved_objects/_export"
	apiGetDetecionRulePath            = "/api/detection_engine/rules"
	apiLoadPrebuiltDetectionRulesPath = "/api/detection_engine/rules/prepackaged"
	apiSavedObjects                   = "/api/saved_objects"
	apiStatusPath                     = "/api/status"
	apiInstalledPackagesPath          = "/api/fleet/epm/packages"
	apiFullAgentPolicyPath            = "/api/fleet/agent_policies/%s/full"

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
	Attributes struct {
		KibanaSavedObjectMeta struct {
			SearchSourceJSON string `json:"searchSourceJSON"`
		} `json:"kibanaSavedObjectMeta"`
		Title string `json:"title"`
	} `json:"attributes"`
	ID   string `json:"id"`
	Type string `json:"type"`
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

// createPolicyForPackageInputAndDataset creates an agent policy and a package policy for the
// given package. When templateName, inputName, and inputType are all non-empty, the package
// policy targets a specific input; otherwise a default policy with no custom inputs is created.
// inputEffectiveName is the input's effective name as defined by Fleet (name field when present,
// otherwise the input type), used to build the simplified policy inputs key.
func (k *Kibana) createPolicyForPackageInputAndDataset(name, version, templateName, inputName, inputType, inputEffectiveName, dataset string) (string, error) {
	err := k.deletePackagePolicyForPackage(name)
	if err != nil {
		return "", fmt.Errorf("failed to delete agent policy: %w", err)
	}

	agentPolicy, err := k.createAgentPolicyForPackage(name)
	if err != nil {
		return "", fmt.Errorf("failed to create agent policy: %w", err)
	}

	err = k.createPackagePolicy(agentPolicy.Item.ID, name, version, templateName, inputName, inputType, inputEffectiveName, dataset)
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

type packagePolicyItem struct {
	ID        string   `json:"id"`
	PolicyID  string   `json:"policy_id"`
	PolicyIDs []string `json:"policy_ids"`
}

func (p packagePolicyItem) belongsTo(agentPolicyID string) bool {
	if p.PolicyID == agentPolicyID {
		return true
	}
	return slices.Contains(p.PolicyIDs, agentPolicyID)
}

// deletePackagePoliciesForAgentPolicy removes package policies attached to the agent policy.
// Kibana 8.9 and 8.14 reject a "force" field on agent policy deletion, so the package
// policies have to be removed first.
func (k *Kibana) deletePackagePoliciesForAgentPolicy(agentPolicyID string) error {
	req, err := k.newRequest(http.MethodGet, apiPackagePolicyPath, nil)
	if err != nil {
		return err
	}
	req = addRequestParams(req, map[string]string{"perPage": "1000"})

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
		return fmt.Errorf("listing package policies failed with status %d, body: %q", resp.StatusCode, string(respBody))
	}

	var listed struct {
		Items []packagePolicyItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		return fmt.Errorf("failed to decode package policies response: %w", err)
	}

	var packagePolicyIDs []string
	for _, item := range listed.Items {
		if item.belongsTo(agentPolicyID) {
			packagePolicyIDs = append(packagePolicyIDs, item.ID)
		}
	}
	if len(packagePolicyIDs) == 0 {
		return nil
	}

	deleteBody, err := json.Marshal(map[string]any{"packagePolicyIds": packagePolicyIDs})
	if err != nil {
		return err
	}
	req, err = k.newRequest(http.MethodPost, apiPackagePolicyPath+"/delete", bytes.NewReader(deleteBody))
	if err != nil {
		return err
	}

	resp, err = k.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return fmt.Errorf("deleting package policies for agent policy %q failed with status %d, body: %q", agentPolicyID, resp.StatusCode, string(respBody))
	}

	return nil
}

func (k *Kibana) deletePackagePolicy(policyID string) error {
	if err := k.deletePackagePoliciesForAgentPolicy(policyID); err != nil {
		return err
	}

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

	if resp.StatusCode != http.StatusOK {
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

func (k *Kibana) createPackagePolicy(agentPolicyID, name, version, templateName, inputName, inputType, inputEffectiveName, dataset string) error {
	var packagePolicyRequest createPackagePolicyRequest
	packagePolicyRequest.Name = name + "-test-1"
	packagePolicyRequest.PolicyID = agentPolicyID
	packagePolicyRequest.Package.Name = name
	packagePolicyRequest.Package.Version = version

	if templateName != "" && inputName != "" && inputType != "" {
		policyInputName := templateName + "-" + inputEffectiveName
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

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// mustExistSLOTemplate checks if an SLO template with the given ID exists.
func (k *Kibana) mustExistSLOTemplate(templateID string) error {
	apiPath := fmt.Sprintf(apiGetSloTemplatePath, defaultSpace)
	apiPath, err := url.JoinPath(apiPath, templateID)
	if err != nil {
		return err
	}
	req, err := k.newInternalRequest(http.MethodGet, apiPath, nil)
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

// mustExistDashboard checks if a dashboard with the given ID exists.
func (k *Kibana) mustExistDashboard(dashboardID string) error {
	_, err := k.getDashboard(dashboardID)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kibana) getDashboard(dashboardID string) (*dashboardResponse, error) {
	// Use the export API to retrieve the given dashboard given its "id".
	body := fmt.Sprintf(`{"objects": [{"id": "%s", "type": "dashboard"}],"excludeExportDetails":true,"includeReferencesDeep":false}`, dashboardID)
	req, err := k.newRequest(http.MethodPost, apiExportSavedObjectPath, strings.NewReader(body))
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

// mustExistDetectionRule checks if a detection rule with the given ID exists.
func (k *Kibana) mustExistDetectionRule(detectionRuleID string) error {
	_, err := k.getDetectionRuleID(detectionRuleID)
	if err != nil {
		return err
	}
	return nil
}

// loadPrebuiltDetectionRules retrieves rule statuses and loads Elastic prebuilt detection rules.
func (k *Kibana) loadPrebuiltDetectionRules() error {
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

// mustExistSavedObject checks if a saved object with the given type and id exists.
func (k *Kibana) mustExistSavedObject(soType, id string) error {
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

var errPackageNotInstalled = errors.New("package is not installed")

type packageInstallResponse struct {
	Item struct {
		Status string `json:"status"`
	} `json:"item"`
}

// isPackageInstalled checks if a package with the given name is installed.
func (k *Kibana) isPackageInstalled(packageName string) error {
	apiPath, err := url.JoinPath(apiInstalledPackagesPath, packageName)
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

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("package %q is not installed: %w", packageName, errPackageNotInstalled)
	}
	if resp.StatusCode >= 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return fmt.Errorf("request failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var packageInfo packageInstallResponse
	if err := json.NewDecoder(resp.Body).Decode(&packageInfo); err != nil {
		return fmt.Errorf("failed to decode package %q response: %w", packageName, err)
	}
	if packageInfo.Item.Status != "installed" {
		return fmt.Errorf("package %q is not installed (status: %q): %w", packageName, packageInfo.Item.Status, errPackageNotInstalled)
	}

	return nil
}

// deleteDetectionRule deletes the detection rule with the given rule ID.
// Nothing is done if the rule does not exist.
func (k *Kibana) deleteDetectionRule(detectionRuleID string) error {
	req, err := k.newRequest(http.MethodDelete, apiGetDetecionRulePath, nil)
	if err != nil {
		return err
	}
	req = addRequestParams(req, map[string]string{"rule_id": detectionRuleID})

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body (status: %d)", resp.StatusCode)
		}
		return fmt.Errorf("deleting detection rule %q failed with status %d, body: %s", detectionRuleID, resp.StatusCode, string(respBody))
	}

	return nil
}

type installedPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type installedPackagesResponse struct {
	Items []installedPackage `json:"items"`
}

// listInstalledPackages returns the packages currently installed in Kibana.
func (k *Kibana) listInstalledPackages() ([]installedPackage, error) {
	apiPath, err := url.JoinPath(apiInstalledPackagesPath, "installed")
	if err != nil {
		return nil, err
	}
	req, err := k.newRequest(http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	req = addRequestParams(req, map[string]string{"perPage": "1000"})

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
		return nil, fmt.Errorf("listing installed packages failed with status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var packages installedPackagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return nil, fmt.Errorf("failed to decode installed packages response: %w", err)
	}

	return packages.Items, nil
}

// uninstallPackage uninstalls the given package. The installed version is looked up
// first in the list of installed packages, nothing is done if it is not installed.
func (k *Kibana) uninstallPackage(packageName string) error {
	packages, err := k.listInstalledPackages()
	if err != nil {
		return fmt.Errorf("failed to look for package %q to uninstall: %w", packageName, err)
	}

	idx := slices.IndexFunc(packages, func(p installedPackage) bool { return p.Name == packageName })
	if idx < 0 {
		// Nothing to do.
		return nil
	}

	apiPath, err := url.JoinPath(apiInstalledPackagesPath, packageName, packages[idx].Version)
	if err != nil {
		return err
	}
	req, err := k.newRequest(http.MethodDelete, apiPath, nil)
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
		return fmt.Errorf("uninstalling package %q failed with status %d, body: %s", packageName, resp.StatusCode, string(respBody))
	}

	return nil
}

type fullAgentPolicy struct {
	Inputs     []fullAgentPolicyInput            `json:"inputs"`
	Processors map[string]otelTransformProcessor `json:"processors"`
}

type fullAgentPolicyInput struct {
	DataStream dataStreamRef           `json:"data_stream"`
	Streams    []fullAgentPolicyStream `json:"streams"`
}

type fullAgentPolicyStream struct {
	DataStream dataStreamRef `json:"data_stream"`
}

type dataStreamRef struct {
	Dataset   string `json:"dataset"`
	Namespace string `json:"namespace"`
}

type otelTransformProcessor struct {
	LogStatements    []otelStatementGroup `json:"log_statements"`
	MetricStatements []otelStatementGroup `json:"metric_statements"`
	TraceStatements  []otelStatementGroup `json:"trace_statements"`
}

type otelStatementGroup struct {
	Context    string   `json:"context"`
	Statements []string `json:"statements"`
}

// getFullAgentPolicy retrieves the compiled/full agent policy for a given agent policy ID.
func (k *Kibana) getFullAgentPolicy(agentPolicyID string) (*fullAgentPolicy, error) {
	apiPath := fmt.Sprintf(apiFullAgentPolicyPath, agentPolicyID)
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

	var response struct {
		Item fullAgentPolicy `json:"item"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode full agent policy response: %w", err)
	}

	return &response.Item, nil
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

func (k *Kibana) newInternalRequest(method string, path string, body io.Reader) (*http.Request, error) {
	req, err := k.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-elastic-internal-origin", "kibana")
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
