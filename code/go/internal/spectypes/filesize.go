// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spectypes

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Common units for file sizes.
const (
	Byte     = FileSize(1)
	KiloByte = 1024 * Byte
	MegaByte = 1024 * KiloByte
)

const (
	byteString     = "B"
	kiloByteString = "KB"
	megaByteString = "MB"
)

// FileSize represents the size of a file.
type FileSize uint64

// Ensure FileSize implements these interfaces.
var (
	_ json.Marshaler   = new(FileSize)
	_ json.Unmarshaler = new(FileSize)
	_ yaml.Marshaler   = new(FileSize)
	_ yaml.Unmarshaler = new(FileSize)
)

func parseFileSizeInt(s string) (uint64, error) {
	// os.FileInfo reports size as int64, don't support bigger numbers.
	maxBitSize := 63
	return strconv.ParseUint(s, 10, maxBitSize)
}

// MarshalJSON implements the json.Marshaler interface for FileSize, it returns
// the string representation in a format that can be unmarshaled back to an
// equivalent value.
func (s FileSize) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// MarshalYAML implements the yaml.Marshaler interface for FileSize, it returns
// the string representation in a format that can be unmarshaled back to an
// equivalent value.
func (s FileSize) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for FileSize.
func (s *FileSize) UnmarshalJSON(d []byte) error {
	// Support unquoted plain numbers.
	bytes, err := parseFileSizeInt(string(d))
	if err == nil {
		*s = FileSize(bytes)
		return nil
	}

	var text string
	err = json.Unmarshal(d, &text)
	if err != nil {
		return err
	}

	return s.unmarshalString(text)
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for FileSize.
func (s *FileSize) UnmarshalYAML(value *yaml.Node) error {
	// Support unquoted plain numbers.
	bytes, err := parseFileSizeInt(value.Value)
	if err == nil {
		*s = FileSize(bytes)
		return nil
	}

	return s.unmarshalString(value.Value)
}

var bytesPattern = regexp.MustCompile(fmt.Sprintf(`^(\d+)(%s|%s|%s|)$`, byteString, kiloByteString, megaByteString))

func (s *FileSize) unmarshalString(text string) error {
	match := bytesPattern.FindStringSubmatch(text)
	if len(match) < 3 {
		return fmt.Errorf("invalid format for file size (%s)", text)
	}

	q, err := parseFileSizeInt(match[1])
	if err != nil {
		return fmt.Errorf("invalid format for file size (%s): %w", text, err)
	}

	unit := match[2]
	switch unit {
	case megaByteString:
		*s = FileSize(q) * MegaByte
	case kiloByteString:
		*s = FileSize(q) * KiloByte
	case byteString, "":
		*s = FileSize(q) * Byte
	default:
		return fmt.Errorf("invalid unit for filesize (%s): %s", text, unit)
	}

	return nil
}

// String returns the string representation of the FileSize.
func (s FileSize) String() string {
	format := func(q FileSize, unit string) string {
		return fmt.Sprintf("%d%s", q, unit)
	}

	if s >= MegaByte && (s%MegaByte == 0) {
		return format(s/MegaByte, megaByteString)
	}

	if s >= KiloByte && (s%KiloByte == 0) {
		return format(s/KiloByte, kiloByteString)
	}

	return format(s, byteString)
}
