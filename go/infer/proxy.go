// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"context"
	"log/slog"
	"time"

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// forwardInference forwards an inference request to the backend.
func (inf *Infer) forwardInference(ctx context.Context, query *InferQuery, c echo.Context, resChan, errChan chan<- StreamedMsg) error {
	// Check if infer is already running
	inf.mu.Lock()
	if inf.IsInferring {
		inf.mu.Unlock()
		errChan <- StreamedMsg{
			Num:     0,
			Error:   gie.New(gie.InferErr, "infer already running"),
			MsgType: ErrorMsgType,
		}
		return nil
	}
	inf.mu.Unlock()

	// Execute infer in goroutine with timeout
	// inferCtx enforces a hard timeout for the backend inference request.
	inferCtx, cancel := context.WithTimeout(ctx, time.Minute*5)

	resChanInternal := make(chan StreamedMsg)
	errChanInternal := make(chan StreamedMsg)

	go inf.Infer(inferCtx, query, c, resChanInternal, errChanInternal)

	// Process response and forward to caller channels
	select {
	case response, ok := <-resChanInternal:
		if ok {
			resChan <- response
		} else {
			errChan <- StreamedMsg{
				Num:     0,
				Error:   gie.New(gie.InferErr, "inference channel closed unexpectedly"),
				MsgType: ErrorMsgType,
			}
		}
	case message, ok := <-errChanInternal:
		if ok {
			errChan <- message
		} else {
			errChan <- StreamedMsg{
				Num:     0,
				Error:   gie.New(gie.InferErr, "error channel closed unexpectedly"),
				MsgType: ErrorMsgType,
			}
		}
	case <-inferCtx.Done():
		slog.DebugContext(ctx, "Infer timeout")
		errChan <- StreamedMsg{
			Num:     0,
			Error:   gie.New(gie.Timeout, "inference timeout"),
			MsgType: ErrorMsgType,
		}
	case <-ctx.Done():
		// Client canceled request
		inf.mu.Lock()
		inf.ContinueInferringController = false
		inf.mu.Unlock()
		errChan <- StreamedMsg{
			Num:     0,
			Error:   gie.New(gie.InferErr, "request canceled by client"),
			MsgType: ErrorMsgType,
		}
	}

	defer cancel()
	return nil
}

// abortInference aborts an ongoing inference.
func (inf *Infer) abortInference() error {
	inf.mu.Lock()
	defer inf.mu.Unlock()
	if !inf.IsInferring {
		return gie.New(gie.InferErr, "no inference running, nothing to abort")
	}

	slog.Debug("Aborting inference")

	inf.ContinueInferringController = false
	return nil
}
