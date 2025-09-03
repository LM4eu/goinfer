// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package lm

import (
	"context"
	"encoding/json"

	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

// write writes a stream message to the client.
func write(ctx context.Context, c echo.Context, jsonEncoder *json.Encoder, msg *types.StreamedMsg) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	_, err = c.Response().Write([]byte("data: "))
	if err != nil {
		return err
	}
	err = jsonEncoder.Encode(msg)
	if err != nil {
		return err
	}
	_, err = c.Response().Write([]byte("\n"))
	if err != nil {
		return err
	}
	c.Response().Flush()
	return nil
}

// sendTerm sends a stream termination message.
func sendTerm(ctx context.Context, c echo.Context) error {
	err := ctx.Err()
	if err != nil {
		return err
	}
	_, err = c.Response().Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		return err
	}
	c.Response().Flush()
	return nil
}
