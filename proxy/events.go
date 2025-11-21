// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package proxy

// package level registry of the different event types

const (
	ProcessStateChangeEventID  = 0x01
	ChatCompletionStatsEventID = 0x02
	ConfigFileChangedEventID   = 0x03
	LogDataEventID             = 0x04
	TokenMetricsEventID        = 0x05
	ModelPreloadedEventID      = 0x06
)

type ProcessStateChangeEvent struct {
	ProcessName string
	NewState    ProcessState
	OldState    ProcessState
}

func (e ProcessStateChangeEvent) Type() uint32 {
	return ProcessStateChangeEventID
}

type ChatCompletionStats struct {
	TokensGenerated int
}

func (e ChatCompletionStats) Type() uint32 {
	return ChatCompletionStatsEventID
}

type ReloadingState int

const (
	ReloadingStateStart ReloadingState = iota
	ReloadingStateEnd
)

type ConfigFileChangedEvent struct {
	ReloadingState ReloadingState
}

func (e ConfigFileChangedEvent) Type() uint32 {
	return ConfigFileChangedEventID
}

type LogDataEvent struct {
	Data []byte
}

func (e LogDataEvent) Type() uint32 {
	return LogDataEventID
}

type ModelPreloadedEvent struct {
	ModelName string
	Success   bool
}

func (e ModelPreloadedEvent) Type() uint32 {
	return ModelPreloadedEventID
}
