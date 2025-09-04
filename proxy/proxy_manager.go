// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/LM4eu/goinfer/errors"
	"github.com/LM4eu/goinfer/lm"
	"github.com/LM4eu/goinfer/state"
	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

// ProxyManager manages proxying requests to the backend LLM engine.
type ProxyManager struct{}

// NewProxyManager creates a new ProxyManager instance.
func NewProxyManager() *ProxyManager {
	return &ProxyManager{}
}

// ForwardInference forwards an inference request to the backend.
func (pm *ProxyManager) ForwardInference(ctx context.Context, query *types.InferQuery, c echo.Context, resultChan, errorChan chan<- types.StreamedMsg) error {
	// Check if infer is already running
	if state.IsInferring {
		errorChan <- types.StreamedMsg{
			Num:     0,
			Content: errors.Wrap(errors.ErrInferenceRunning, errors.TypeInference, "INFERENCE_RUNNING", "infer already running").Error(),
			MsgType: types.ErrorMsgType,
		}
		return nil
	}

	// Execute infer in goroutine with timeout
	inferCtx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	resultChanInternal := make(chan types.StreamedMsg)
	errorChanInternal := make(chan types.StreamedMsg)
	defer close(resultChanInternal)
	defer close(errorChanInternal)

	// Call the existing lm.Infer function through the proxy
	go lm.Infer(inferCtx, query, c, resultChanInternal, errorChanInternal)

	// Process response and forward to caller channels
	select {
	case response, ok := <-resultChanInternal:
		if ok {
			resultChan <- response
		} else {
			errorChan <- types.StreamedMsg{
				Num:     0,
				Content: errors.Wrap(errors.ErrChannelClosed, errors.TypeInference, "CHANNEL_CLOSED", "infer channel closed unexpectedly").Error(),
				MsgType: types.ErrorMsgType,
			}
		}
	case message, ok := <-errorChanInternal:
		if ok {
			errorChan <- message
		} else {
			errorChan <- types.StreamedMsg{
				Num:     0,
				Content: errors.Wrap(errors.ErrChannelClosed, errors.TypeInference, "CHANNEL_CLOSED", "error channel closed unexpectedly").Error(),
				MsgType: types.ErrorMsgType,
			}
		}
	case <-inferCtx.Done():
		if state.Debug {
			fmt.Printf("DBG: Infer timeout\n")
		}
		errorChan <- types.StreamedMsg{
			Num:     0,
			Content: errors.Wrap(errors.ErrRequestTimeout, errors.TypeTimeout, "INFERENCE_TIMEOUT", "infer timeout").Error(),
			MsgType: types.ErrorMsgType,
		}
	case <-ctx.Done():
		// Client canceled request
		state.ContinueInferringController = false
		errorChan <- types.StreamedMsg{
			Num:     0,
			Content: errors.Wrap(errors.ErrClientCanceled, errors.TypeInference, "CLIENT_CANCELED", "req canceled by client").Error(),
			MsgType: types.ErrorMsgType,
		}
	}

	return nil
}

// AbortInference aborts an ongoing inference.
func (pm *ProxyManager) AbortInference() error {
	if !state.IsInferring {
		return errors.Wrap(errors.ErrInferenceNotRunning, errors.TypeInference, "INFERENCE_NOT_RUNNING", "no inference running, nothing to abort")
	}

	if state.Verbose {
		fmt.Println("INF: Aborting inference")
	}

	state.ContinueInferringController = false
	return nil
}

// IsInferring returns true if an inference is currently running.
func (pm *ProxyManager) IsInferring() bool {
	return state.IsInferring
}

// ForwardToLlama forwards a request to the llama backend.
// This function is kept for backward compatibility but delegates to the ProxyManager.
func ForwardToLlama(c echo.Context) error {
	// This function is deprecated and kept for backward compatibility.
	// New code should use the ProxyManager interface directly.
	// For simple requests, we can't use the full ProxyManager here without
	// the proper context and query structure, so we return a meaningful error
	return c.String(501, "ForwardToLlama deprecated: Use ProxyManager interface instead")
}
