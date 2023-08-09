// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

// Elasticsearch is an elasticsearch client.
type Elasticsearch struct {
	client *elasticsearch.TypedClient
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
	client, err := elasticsearch.NewTypedClient(config)
	if err != nil {
		return nil, err
	}

	return &Elasticsearch{
		client: client,
	}, nil
}

// IndexTemplate looks for an index template.
func (es *Elasticsearch) IndexTemplate(name string) (*types.IndexTemplate, error) {
	resp, err := es.client.Indices.GetIndexTemplate().Name(name).Do(context.TODO())
	if err != nil {
		return nil, err
	}
	if n := len(resp.IndexTemplates); n != 1 {
		return nil, fmt.Errorf("one index template expected, found %d", n)
	}

	return &resp.IndexTemplates[0].IndexTemplate, nil
}

// SimulateIndexTemplate simulates the instantiation of an index template, resolving its
// component templates.
func (es *Elasticsearch) SimulateIndexTemplate(name string) (*SimulatedIndexTemplate, error) {
	resp, err := es.client.Indices.SimulateTemplate().Name(name).Do(context.TODO())
	if err != nil {
		return nil, err
	}

	return &SimulatedIndexTemplate{resp.Template}, nil
}
