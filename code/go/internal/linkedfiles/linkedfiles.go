// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package linkedfiles

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// A Link represents a linked file.
// It contains the path to the link file, the checksum of the linked file,
// the path to the target file, and the checksum of the included file contents.
// It also contains a boolean indicating whether the link is up to date.
type Link struct {
	LinkFilePath string
	LinkChecksum string

	IncludedFilePath             string
	IncludedFileContentsChecksum string

	UpToDate bool
}

// NewLinkedFile creates a new Link from the given link file path.
func NewLinkedFile(linkFilePath string) (Link, error) {
	var l Link
	firstLine, err := readFirstLine(linkFilePath)
	if err != nil {
		return Link{}, err
	}
	l.LinkFilePath = linkFilePath

	fields := strings.Fields(firstLine)
	l.IncludedFilePath = fields[0]
	if len(fields) == 2 {
		l.LinkChecksum = fields[1]
	}

	pathName := filepath.Join(filepath.Dir(linkFilePath), filepath.FromSlash(l.IncludedFilePath))
	cs, err := getLinkedFileChecksum(pathName)
	if err != nil {
		return Link{}, fmt.Errorf("could not collect file %v: %w", l.IncludedFilePath, err)
	}
	if l.LinkChecksum == cs {
		l.UpToDate = true
	}
	l.IncludedFileContentsChecksum = cs

	return l, nil
}

func getLinkedFileChecksum(path string) (string, error) {
	b, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		return "", err
	}
	cs, err := checksum(b)
	if err != nil {
		return "", err
	}
	return cs, nil
}

func readFirstLine(filePath string) (string, error) {
	file, err := os.Open(filepath.FromSlash(filePath))
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("file is empty or first line is missing")
}

func checksum(b []byte) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, bytes.NewReader(b)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
