// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteWithHeader(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "out.txt")
	header := "# Test Header\n\n"
	data := []byte("key: value\nanother: line\n")

	_, err := writeWithHeader(filePath, header, data)
	require.NoError(t, err)

	// Read back the file and verify contents.
	content, err := os.ReadFile(filepath.Clean(filePath))
	require.NoError(t, err)

	expected := header + string(data)
	require.Equal(t, expected, string(content))
}

func TestGen64HexDigits(t *testing.T) {
	t.Parallel()

	hexStr := gen64HexDigits()
	// Should be exactly 64 characters long.
	require.Len(t, hexStr, 64)
	// Should contain only hexadecimal characters.
	require.Regexp(t, "^[0-9a-f]{64}$", hexStr)
}
