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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LinkExtension is the file extension for linked files.
const LinkExtension = ".link"

// A Link represents a linked file.
// It contains the path to the link file, the checksum of the linked file,
// the path to the target file, and the checksum of the included file contents.
// It also contains a boolean indicating whether the link is up to date.
type Link struct {
	root *os.Root

	LinkFilePath string
	LinkChecksum string

	TargetFilePath string

	IncludedFilePath             string
	IncludedFileContentsChecksum string

	UpToDate bool
}

func newLinkedFile(root *os.Root, linkFilePath string) (Link, error) {
	var l Link
	l.root = root
	firstLine, err := readFirstLine(root, linkFilePath)
	if err != nil {
		return Link{}, err
	}
	l.LinkFilePath = linkFilePath
	l.TargetFilePath = strings.TrimSuffix(linkFilePath, LinkExtension)

	fields := strings.Fields(firstLine)
	l.IncludedFilePath = fields[0]
	if len(fields) == 2 {
		l.LinkChecksum = fields[1]
	}

	cs, err := getLinkedFileChecksum(root, l.IncludedFilePath)
	if err != nil {
		return Link{}, fmt.Errorf("could not collect file %v: %w", l.IncludedFilePath, err)
	}
	if l.LinkChecksum == cs {
		l.UpToDate = true
	}
	l.IncludedFileContentsChecksum = cs

	return l, nil
}

// IncludeLinkedFiles function includes linked files from the source
// directory to the target directory.
// It returns a slice of Link structs representing the included files.
// It also updates the checksum of the linked files.
// Both directories must be relative to the root.
func IncludeLinkedFiles(root *os.Root, fromDir, toDir string) ([]Link, error) {
	links, err := ListLinkedFilesInRoot(root, fromDir)
	if err != nil {
		return nil, fmt.Errorf("including linked files failed: %w", err)
	}

	for _, l := range links {
		l.ReplaceTargetFilePathDirectory(fromDir, toDir)

		if _, err := l.UpdateChecksum(); err != nil {
			return nil, fmt.Errorf("could not update checksum for file %v: %w", l.LinkFilePath, err)
		}

		if err := CopyFileFromRoot(root, l.IncludedFilePath, l.TargetFilePath); err != nil {
			return nil, fmt.Errorf("could not write file %v: %w", l.TargetFilePath, err)
		}
	}

	return links, nil
}

// UpdateChecksum function updates the checksum of the linked file.
// It returns true if the checksum was updated, false if it was already up-to-date.
func (l *Link) UpdateChecksum() (bool, error) {
	if l.UpToDate {
		return false, nil
	}
	if l.IncludedFilePath == "" {
		return false, fmt.Errorf("file path is empty for file %v", l.IncludedFilePath)
	}
	if l.IncludedFileContentsChecksum == "" {
		return false, fmt.Errorf("checksum is empty for file %v", l.IncludedFilePath)
	}
	newContent := fmt.Sprintf("%v %v", filepath.ToSlash(l.IncludedFilePath), l.IncludedFileContentsChecksum)
	if err := WriteFileToRoot(l.root, l.LinkFilePath, []byte(newContent)); err != nil {
		return false, fmt.Errorf("could not update checksum for file %v: %w", l.LinkFilePath, err)
	}
	l.LinkChecksum = l.IncludedFileContentsChecksum
	l.UpToDate = true
	return true, nil
}

// ReplaceTargetFilePathDirectory function replaces the target file path directory.
func (l *Link) ReplaceTargetFilePathDirectory(fromDir, toDir string) {
	// if a destination dir is set we replace the source dir with the destination dir
	if toDir == "" {
		return
	}
	l.TargetFilePath = strings.Replace(
		l.TargetFilePath,
		fromDir,
		toDir,
		1,
	)
}

// ListLinkedFilesInRoot function returns a slice of Link structs representing linked files.
func ListLinkedFilesInRoot(root *os.Root, fromDir string) ([]Link, error) {
	var linkFiles []string
	if err := filepath.Walk(
		filepath.Join(root.Name(), filepath.FromSlash(fromDir)),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), LinkExtension) {
				// make the path relative to the root
				path, err = filepath.Rel(root.Name(), path)
				if err != nil {
					return err
				}
				linkFiles = append(linkFiles, path)
			}
			return nil
		}); err != nil {
		return nil, err
	}

	links := make([]Link, len(linkFiles))

	for i, f := range linkFiles {
		l, err := newLinkedFile(root, f)
		if err != nil {
			return nil, fmt.Errorf("could not initialize linked file %v: %w", f, err)
		}
		links[i] = l
	}

	return links, nil
}

// CopyFileFromRoot function copies a file from to to inside the root.
func CopyFileFromRoot(root *os.Root, from, to string) error {
	from = filepath.FromSlash(from)
	source, err := root.Open(from)
	if err != nil {
		return err
	}
	defer source.Close()

	to = filepath.FromSlash(to)
	if _, err := root.Stat(filepath.Dir(to)); os.IsNotExist(err) {
		dir := filepath.Join(root.Name(), filepath.Dir(to))
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	destination, err := root.Create(to)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// WriteFileToRoot function writes a byte slice to a file inside the root.
func WriteFileToRoot(root *os.Root, to string, b []byte) error {
	to = filepath.Join(root.Name(), filepath.FromSlash(to))
	if _, err := root.Stat(filepath.Dir(to)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(to), 0700); err != nil {
			return err
		}
	}
	return os.WriteFile(to, b, 0644)
}

func getLinkedFileChecksum(root *os.Root, path string) (string, error) {
	b, err := root.FS().(fs.ReadFileFS).ReadFile(filepath.FromSlash(path))
	if err != nil {
		return "", err
	}
	cs, err := checksum(b)
	if err != nil {
		return "", err
	}
	return cs, nil
}

func readFirstLine(root *os.Root, filePath string) (string, error) {
	file, err := root.Open(filepath.FromSlash(filePath))
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
