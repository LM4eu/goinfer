// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package lm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

// sendStartMsg sends the start_emitting message to the client.
func sendStartMsg(ctx context.Context, jsonEncoder *json.Encoder, c echo.Context, params types.InferParams, ntok int, thinkingElapsed time.Duration) error {
	err := ctx.Err()
	if err != nil {
		return err
	}
	if !params.Stream {
		return nil
	}

	smsg := &types.StreamedMsg{
		Content: "start_emitting",
		Num:     ntok,
		MsgType: types.SystemMsgType,
		Data: map[string]any{
			"thinking_time":        thinkingElapsed,
			"thinking_time_format": thinkingElapsed.String(),
		},
	}

	return write(ctx, c, jsonEncoder, smsg)
}

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

// streamDelta handles token processing during prediction.
func streamDelta(ctx context.Context, ntok int, token string, jsonEncoder *json.Encoder, c echo.Context, params types.InferParams,
	startThinking time.Time, thinkingElapsed *time.Duration, startEmitting *time.Time,
) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	if ntok == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)
		return sendStartMsg(ctx, jsonEncoder, c, params, ntok, *thinkingElapsed)
	}
	if !params.Stream {
		return nil
	}

	tmsg := &types.StreamedMsg{
		Content: token,
		Num:     ntok,
		MsgType: types.TokenMsgType,
	}

	return write(ctx, c, jsonEncoder, tmsg)
}

// createResult creates the final result message.
func createResult(ctx context.Context, res string, stats types.InferStat, jsonEncoder *json.Encoder, c echo.Context, params types.InferParams) (types.StreamedMsg, error) {
	endmsg := types.StreamedMsg{
		Content: "result",
		Num:     stats.TotalTokens + 1,
		MsgType: types.SystemMsgType,
		Data: map[string]any{
			"text":  res,
			"stats": stats,
		},
	}

	if params.Stream {
		err := write(ctx, c, jsonEncoder, &endmsg)
		if err != nil {
			return endmsg, err
		}
	}

	return endmsg, nil
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
