package lm

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/types"
)

// StreamTokenMessage creates a token message for streaming
func StreamTokenMessage(content string, num int, data map[string]any) *types.StreamedMessage {
	return &types.StreamedMessage{
		Content: content,
		Num:     num,
		MsgType: types.TokenMsgType,
		Data:    data,
	}
}

// StreamSystemMessage creates a system message for streaming
func StreamSystemMessage(content string, num int, data map[string]any) *types.StreamedMessage {
	return &types.StreamedMessage{
		Num:     num,
		Content: content,
		MsgType: types.SystemMsgType,
		Data:    data,
	}
}

// StreamErrorMessage creates an error message for streaming
func StreamErrorMessage(content string, num int) *types.StreamedMessage {
	return &types.StreamedMessage{
		Num:     num,
		Content: content,
		MsgType: types.ErrorMsgType,
	}
}

// SendStartEmittingMessage sends the start_emitting message to the client
func SendStartEmittingMessage(enc *json.Encoder, c echo.Context, params types.InferParams, ntokens int, thinkingElapsed time.Duration) error {
	if !params.Stream {
		return nil
	}

	smsg := types.StreamedMessage{
		Content: "start_emitting",
		Num:     ntokens,
		MsgType: types.SystemMsgType,
		Data: map[string]any{
			"thinking_time":        thinkingElapsed,
			"thinking_time_format": thinkingElapsed.String(),
		},
	}

	_, err := c.Response().Write([]byte("data: "))
	if err != nil {
		return fmt.Errorf("failed to write stream begin: %w", err)
	}

	err = enc.Encode(smsg)
	if err != nil {
		return fmt.Errorf("failed to encode stream message: %w", err)
	}

	_, err = c.Response().Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write stream message: %w", err)
	}

	c.Response().Flush()
	time.Sleep(2 * time.Millisecond) // Give some time to stream this message
	return err
}

// StreamDeltaMessage handles token processing during prediction
func StreamDeltaMessage(ntokens int, token string, enc *json.Encoder, c echo.Context, params types.InferParams,
	startThinking time.Time, thinkingElapsed *time.Duration, startEmitting *time.Time) error {
	
	if ntokens == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)

		err := SendStartEmittingMessage(enc, c, params, ntokens, *thinkingElapsed)
		if err != nil {
			fmt.Printf("Error emitting msg: %v\n", err)
			return err
		}
	}

	if !params.Stream {
		return nil
	}

	tmsg := types.StreamedMessage{
		Content: token,
		Num:     ntokens,
		MsgType: types.TokenMsgType,
	}

	_, err := c.Response().Write([]byte("data: "))
	if err != nil {
		return fmt.Errorf("failed to write stream begin: %w", err)
	}

	err = enc.Encode(tmsg)
	if err != nil {
		return fmt.Errorf("failed to encode stream message: %w", err)
	}

	_, err = c.Response().Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write stream message: %w", err)
	}

	c.Response().Flush()
	return nil
}

// CreateResultMessage creates the final result message
func CreateResultMessage(res string, stats InferStats, enc *json.Encoder, c echo.Context, params types.InferParams) (types.StreamedMessage, error) {
	result := InferResult{
		Text:  res,
		Stats: stats,
	}

	endmsg := types.StreamedMessage{}

	b, err := json.Marshal(&result)
	if err != nil {
		return endmsg, fmt.Errorf("error marshalling result: %w", err)
	}

	var _res map[string]any
	err = json.Unmarshal(b, &_res)
	if err != nil {
		return endmsg, fmt.Errorf("error unmarshalling result: %w", err)
	}

	endmsg = types.StreamedMessage{
		Content: "result",
		Num:     stats.TotalTokens + 1,
		MsgType: types.SystemMsgType,
		Data:    _res,
	}

	if params.Stream {
		_, err := c.Response().Write([]byte("data: "))
		if err != nil {
			return endmsg, fmt.Errorf("failed to write stream begin: %w", err)
		}

		err = enc.Encode(endmsg)
		if err != nil {
			return endmsg, fmt.Errorf("failed to encode stream message: %w", err)
		}

		_, err = c.Response().Write([]byte("\n"))
		if err != nil {
			return endmsg, fmt.Errorf("failed to write stream message: %w", err)
		}

		c.Response().Flush()
	}

	return endmsg, nil
}

// StreamMsg streams a message to the client
func StreamMsg(msg *types.StreamedMessage, c echo.Context, enc *json.Encoder) error {
	_, err := c.Response().Write([]byte("data: "))
	if err != nil {
		return fmt.Errorf("failed to write stream begin: %w", err)
	}

	err = enc.Encode(msg)
	if err != nil {
		return fmt.Errorf("failed to encode stream message: %w", err)
	}

	_, err = c.Response().Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write stream message: %w", err)
	}

	c.Response().Flush()
	return nil
}

// SendStreamTermination sends a stream termination message
func SendStreamTermination(c echo.Context) error {
	_, err := c.Response().Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		return fmt.Errorf("failed to write stream termination: %w", err)
	}
	c.Response().Flush()
	return nil
}
