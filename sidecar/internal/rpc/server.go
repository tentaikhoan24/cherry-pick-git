// Package rpc implements a newline-delimited JSON-RPC 2.0 server over io.Reader/Writer.
// See docs/IPC.md for the wire protocol.
package rpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is the wire-level JSON-RPC error object.
type Error struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// Handler runs a single RPC method. Returning a non-nil *Error marks the call
// as failed; otherwise the result is encoded as the response's "result" field.
type Handler func(ctx context.Context, params json.RawMessage) (any, *Error)

// ProgressWriter emits an intermediate progress notification for the current
// request. The value is encoded as the "progress" field in a JSON-RPC line
// that shares the same id as the request but carries no "result"/"error".
type ProgressWriter func(v any) error

type progressKey struct{}

// WriteProgress emits a progress notification via the writer injected into ctx
// by the server's dispatch loop. Does nothing if no writer is present.
func WriteProgress(ctx context.Context, v any) error {
	if pw, ok := ctx.Value(progressKey{}).(ProgressWriter); ok {
		return pw(v)
	}
	return nil
}

type progressLine struct {
	JSONRPC  string          `json:"jsonrpc"`
	ID       json.RawMessage `json:"id,omitempty"`
	Progress any             `json:"progress"`
}

type Server struct {
	handlers map[string]Handler
}

func NewServer() *Server {
	return &Server{handlers: map[string]Handler{}}
}

// Register associates a method name with a handler. Last registration wins.
func (s *Server) Register(method string, h Handler) {
	s.handlers[method] = h
}

// Serve reads one NDJSON request per line from in, dispatches it, and writes
// one response per line to out. Parse and dispatch errors go on out as JSON-RPC
// error responses; stderr is reserved for non-protocol diagnostics on errOut.
func (s *Server) Serve(ctx context.Context, in io.Reader, out, errOut io.Writer) error {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(out)

	for sc.Scan() {
		line := bytes.TrimPrefix(sc.Bytes(), utf8BOM)
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: -32700, Message: "Parse error: " + err.Error()},
			})
			continue
		}

		s.dispatch(ctx, &req, enc)
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(errOut, "sidecar: stdin scan error:", err)
		return err
	}
	return nil
}

func (s *Server) dispatch(ctx context.Context, req *Request, enc *json.Encoder) {
	// Inject a progress writer so handlers can emit intermediate notifications.
	pw := ProgressWriter(func(v any) error {
		return enc.Encode(progressLine{JSONRPC: "2.0", ID: req.ID, Progress: v})
	})
	ctx = context.WithValue(ctx, progressKey{}, pw)

	h, ok := s.handlers[req.Method]
	if !ok {
		_ = enc.Encode(Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: -32601, Message: "Method not found: " + req.Method},
		})
		return
	}

	result, rpcErr := h(ctx, req.Params)
	if rpcErr != nil {
		_ = enc.Encode(Response{JSONRPC: "2.0", ID: req.ID, Error: rpcErr})
		return
	}
	_ = enc.Encode(Response{JSONRPC: "2.0", ID: req.ID, Result: result})
}
