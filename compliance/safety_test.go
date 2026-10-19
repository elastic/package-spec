// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import "testing"

func TestIsLocalHost(t *testing.T) {
	cases := map[string]bool{
		"https://127.0.0.1:5601":                     true,
		"http://localhost:9200":                      true,
		"https://[::1]:5601":                         true,
		"https://kibana.example.com":                 false,
		"https://abc.us-east-1.aws.cloud.es.io:9243": false,
		"https://10.0.0.5:5601":                      false,
		"":                                           false,
		"not a url":                                  false,
	}
	for address, expected := range cases {
		t.Run(address, func(t *testing.T) {
			if got := isLocalHost(address); got != expected {
				t.Errorf("isLocalHost(%q) = %v, expected %v", address, got, expected)
			}
		})
	}
}
