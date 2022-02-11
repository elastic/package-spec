// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spectypes

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
)

const (
	Byte     = FileSize(1)
	KiloByte = 1024 * Byte
	MegaByte = 1024 * KiloByte

	byteString     = "B"
	kiloByteString = "KB"
	megaByteString = "MB"
)

type FileSize uint

func (s FileSize) MarshalJSON() ([]byte, error) {
	formatJSON := func(q FileSize, unit string) []byte {
		return []byte(fmt.Sprintf(`"%d%s"`, uint(q), unit))
	}

	bytes := s
	if bytes >= MegaByte && (bytes%MegaByte == 0) {
		mb := bytes / MegaByte
		return formatJSON(mb, megaByteString), nil
	}

	if bytes >= KiloByte && (bytes%KiloByte == 0) {
		kb := bytes / KiloByte
		return formatJSON(kb, kiloByteString), nil
	}

	return formatJSON(bytes, byteString), nil
}

var bytesPattern = regexp.MustCompile(fmt.Sprintf(`^(\d+)(%s|%s|%s|)$`, byteString, kiloByteString, megaByteString))

func (s *FileSize) UnmarshalJSON(d []byte) error {
	var bytes uint
	err := json.Unmarshal(d, &bytes)
	if err == nil {
		*s = FileSize(bytes)
		return nil
	}

	var text string
	err = json.Unmarshal(d, &text)
	if err != nil {
		return err
	}

	match := bytesPattern.FindStringSubmatch(text)
	if len(match) < 3 {
		return fmt.Errorf("invalid format for file size (%s)", string(d))
	}

	q, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid format for file size (%s): %w", string(d), err)
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
		return fmt.Errorf("invalid unit for filesize (%s): %s", string(d), unit)
	}

	return nil
}
