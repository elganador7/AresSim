package main

import "log/slog"

// BridgeResult is the standard return type for all bridge calls.
type BridgeResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func ok() BridgeResult { return BridgeResult{Success: true} }

func fail(err error) BridgeResult {
	slog.Error("bridge error", "err", err)
	return BridgeResult{Error: err.Error()}
}

func failMsg(msg string) BridgeResult {
	slog.Error("bridge error", "msg", msg)
	return BridgeResult{Error: msg}
}

// GetVersion returns the application version string.
func (a *App) GetVersion() string {
	return "0.1.0-dev"
}
