package herdr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

var errNoCore = errors.New("no core")

// ExecResult is what machine.exec ran and what it said.
type ExecResult struct {
	Code     *int   `json:"code"`
	Stdout   string `json:"-"`
	Stderr   string `json:"-"`
	TimedOut bool   `json:"timed_out"`
}

// Exec runs a program on a machine ("" is this computer's core machine
// space) and returns its output. argv is run as given, no shell.
func (s *Session) Exec(on, cwd string, argv []string, timeout time.Duration) (ExecResult, error) {
	var out ExecResult
	if s.Core == nil {
		return out, errNoCore
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout+10*time.Second)
	defer cancel()
	params := map[string]any{"argv": argv}
	if cwd != "" {
		params["cwd"] = cwd
	}
	if on != "" {
		params["on"] = on
	}
	raw, err := s.Core.Call(ctx, "machine.exec", params)
	if err != nil {
		return out, err
	}
	var res struct {
		Code     *int   `json:"code"`
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		TimedOut bool   `json:"timed_out"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return out, err
	}
	out.Code = res.Code
	out.TimedOut = res.TimedOut
	out.Stdout = decodeBase64(res.Stdout)
	out.Stderr = decodeBase64(res.Stderr)
	return out, nil
}

func decodeBase64(s string) string {
	if s == "" {
		return ""
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(b)
}
