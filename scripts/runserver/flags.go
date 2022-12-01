package main

import "flag"

func parameters() {
	flag.StringVar(&address, "address", "localhost:8080", "Address of the package-registry service.")
	flag.StringVar(&httpProfAddress, "httpprof", "", "Enable HTTP profiler listening on the given address.")
}
