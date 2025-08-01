// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	config := elasticsearch.Config{
		Addresses: []string{
			elasticPackageGetEnv("ELASTICSEARCH_HOST"),
		},
		Username: elasticPackageGetEnv("ELASTICSEARCH_USERNAME"),
		Password: elasticPackageGetEnv("ELASTICSEARCH_PASSWORD"),
	}

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
		config.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		}
	}
	client, err := elasticsearch.NewClient(config)
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

func newJSONDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec
}
