// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/elastic/go-elasticsearch/v9"
)

// Elasticsearch is an elasticsearch client.
type Elasticsearch struct {
	client *elasticsearch.Client
}

// NewElasticsearchClient creates a new Elasticsearch client.
func NewElasticsearchClient() (*Elasticsearch, error) {
	opts := []elasticsearch.Option{
		elasticsearch.WithAddresses(elasticPackageGetEnv("ELASTICSEARCH_HOST")),
		elasticsearch.WithBasicAuth(elasticPackageGetEnv("ELASTICSEARCH_USERNAME"), elasticPackageGetEnv("ELASTICSEARCH_PASSWORD")),
	}

	if caCert := elasticPackageGetEnv("CA_CERT"); caCert != "" {
		// caCert is the path to the CA certificate file
		pem, err := os.ReadFile(caCert)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate \"%s\": %w", caCert, err)
		}

		opts = append(opts, elasticsearch.WithCACert(pem))
	}

	client, err := elasticsearch.New(opts...)
	if err != nil {
		return nil, err
	}

	return &Elasticsearch{
		client: client,
	}, nil
}

// IndexTemplate looks for an index template.
func (es *Elasticsearch) IndexTemplate(name string) (*IndexTemplate, error) {
	resp, err := es.client.Indices.GetIndexTemplate(
		es.client.Indices.GetIndexTemplate.WithContext(context.TODO()),
		es.client.Indices.GetIndexTemplate.WithName(name),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status code %d", resp.StatusCode)
	}

	var templatesResponse struct {
		IndexTemplates []struct {
			IndexTemplate *IndexTemplate `json:"index_template"`
		} `json:"index_templates"`
	}
	err = newJSONDecoder(resp.Body).Decode(&templatesResponse)
	if err != nil {
		return nil, err
	}

	if n := len(templatesResponse.IndexTemplates); n != 1 {
		return nil, fmt.Errorf("one index template expected, found %d", n)
	}

	return templatesResponse.IndexTemplates[0].IndexTemplate, nil
}

// SimulateIndexTemplate simulates the instantiation of an index template, resolving its
// component templates.
func (es *Elasticsearch) SimulateIndexTemplate(name string) (*SimulatedIndexTemplate, error) {
	resp, err := es.client.Indices.SimulateTemplate(
		es.client.Indices.SimulateTemplate.WithName(name),
		es.client.Indices.SimulateTemplate.WithContext(context.TODO()),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status code %d", resp.StatusCode)
	}

	var simulateResponse struct {
		Template *SimulatedIndexTemplate `json:"template"`
	}
	err = newJSONDecoder(resp.Body).Decode(&simulateResponse)
	if err != nil {
		return nil, err
	}
	if simulateResponse.Template == nil {
		return nil, errors.New("empty template simulated, something is wrong")
	}

	return simulateResponse.Template, nil
}

// TransformHasAlias checks whether a transform has the given alias configured
// in its dest.aliases definition. This verifies the configuration stored in
// Elasticsearch, not whether the alias exists as an index alias (which only
// happens after the transform runs and creates its destination index).
func (es *Elasticsearch) TransformHasAlias(transformID, aliasName string) error {
	resp, err := es.client.TransformGetTransform(
		es.client.TransformGetTransform.WithContext(context.TODO()),
		es.client.TransformGetTransform.WithTransformID(transformID),
	)
	if err != nil {
		return fmt.Errorf("failed to get transform %q: %w", transformID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("transform %q not found", transformID)
	}

	var response struct {
		Transforms []struct {
			Dest struct {
				Aliases []struct {
					Alias string `json:"alias"`
				} `json:"aliases"`
			} `json:"dest"`
		} `json:"transforms"`
	}
	if err := newJSONDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode transform response: %w", err)
	}

	for _, transform := range response.Transforms {
		for _, alias := range transform.Dest.Aliases {
			if alias.Alias == aliasName {
				return nil
			}
		}
	}

	return fmt.Errorf("alias %q not found in transform %q configuration", aliasName, transformID)
}

func newJSONDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec
}
