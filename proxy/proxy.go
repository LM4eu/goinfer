// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/LM4eu/goinfer/gie"
	"github.com/LM4eu/goinfer/lm"
	"github.com/LM4eu/goinfer/state"
	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

// ProxyManager manages proxying requests to the backend LLM engine.
type ProxyManager struct{ IsInferring bool }

// ForwardInference forwards an inference request to the backend.
func (pm *ProxyManager) ForwardInference(ctx context.Context, query *types.InferQuery, c echo.Context, resChan, errChan chan<- types.StreamedMsg) error {
	// Check if infer is already running
	if pm.IsInferring {
		errChan <- types.StreamedMsg{
			Num:     0,
			Content: gie.Wrap(gie.ErrInferRunning, gie.TypeInference, "INFERENCE_RUNNING", "infer already running").Error(),
			MsgType: types.ErrorMsgType,
		}
		return nil
	}

	// Execute infer in goroutine with timeout
	inferCtx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	resChanInternal := make(chan types.StreamedMsg)
	errChanInternal := make(chan types.StreamedMsg)
	defer close(resChanInternal)
	defer close(errChanInternal)

	// Call the existing lm.Infer function through the proxy
	go lm.Infer(inferCtx, query, c, resChanInternal, errChanInternal)

	// Process response and forward to caller channels
	select {
	case response, ok := <-resChanInternal:
		if ok {
			resChan <- response
		} else {
			errChan <- types.StreamedMsg{
				Num:     0,
				Content: gie.Wrap(gie.ErrChanClosed, gie.TypeInference, "CHANNEL_CLOSED", "infer channel closed unexpectedly").Error(),
				MsgType: types.ErrorMsgType,
			}
		}
	case message, ok := <-errChanInternal:
		if ok {
			errChan <- message
		} else {
			errChan <- types.StreamedMsg{
				Num:     0,
				Content: gie.Wrap(gie.ErrChanClosed, gie.TypeInference, "CHANNEL_CLOSED", "error channel closed unexpectedly").Error(),
				MsgType: types.ErrorMsgType,
			}
		}
	case <-inferCtx.Done():
		if state.Debug {
			fmt.Printf("DBG: Infer timeout\n")
		}
		errChan <- types.StreamedMsg{
			Num:     0,
			Content: gie.Wrap(gie.ErrReqTimeout, gie.TypeTimeout, "INFERENCE_TIMEOUT", "infer timeout").Error(),
			MsgType: types.ErrorMsgType,
		}
	case <-ctx.Done():
		// Client canceled request
		state.ContinueInferringController = false
		errChan <- types.StreamedMsg{
			Num:     0,
			Content: gie.Wrap(gie.ErrClientCanceled, gie.TypeInference, "CLIENT_CANCELED", "req canceled by client").Error(),
			MsgType: types.ErrorMsgType,
		}
	}

	return nil
}

// AbortInference aborts an ongoing inference.
func (pm *ProxyManager) AbortInference() error {
	if !pm.IsInferring {
		return gie.Wrap(gie.ErrInferNotRunning, gie.TypeInference, "INFERENCE_NOT_RUNNING", "no inference running, nothing to abort")
	}

	if state.Verbose {
		fmt.Println("INF: Aborting inference")
	}

	state.ContinueInferringController = false
	return nil
}
