// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"net"
	"net/url"
	"os"
	"testing"
)

const allowDestructiveEnv = "COMPLIANCE_ALLOW_DESTRUCTIVE"

// isLocalHost returns true if the given address points to the local machine.
func isLocalHost(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// requireSafeTarget fails the test if the target stack is not local, unless
// COMPLIANCE_ALLOW_DESTRUCTIVE=true. The suite installs and uninstalls packages and
// deletes policies and rules, so it must not run by accident against production clusters.
func requireSafeTarget(t *testing.T) {
	t.Helper()

	if os.Getenv(allowDestructiveEnv) == "true" {
		return
	}

	for _, name := range []string{"KIBANA_HOST", "ELASTICSEARCH_HOST"} {
		address := elasticPackageGetEnv(name)
		if !isLocalHost(address) {
			t.Fatalf("%s=%q is not a local address. The compliance suite is destructive (it installs and uninstalls packages, and deletes policies and detection rules), "+
				"do not run it against production clusters. Set %s=true to run it against this target anyway.", name, address, allowDestructiveEnv)
		}
	}
}
