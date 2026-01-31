package main

import (
    "bufio"
    "context"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "net"
    "runtime"
    "strings"
    "time"
)

type runnerExecRequest struct {
    V int `json:"v"`
    Req string `json:"req"`
    Action string `json:"action"`
    Params map[string]any `json:"params,omitempty"`
    InvokedBy *struct {
        DeviceID string `json:"device_id,omitempty"`
        Remote   string `json:"remote,omitempty"`
    } `json:"invoked_by,omitempty"`
}

type runnerExecResponse struct {
    V int `json:"v"`
    Req string `json:"req"`
    OK bool `json:"ok"`
    Error string `json:"error,omitempty"`
    ExitCode int `json:"exit_code,omitempty"`
    DurationMS int64 `json:"duration_ms,omitempty"`
    StdoutB64 string `json:"stdout_b64,omitempty"`
    StderrB64 string `json:"stderr_b64,omitempty"`
}

type RunnerClient struct {
    transport string // unix|tcp
    addr      string
    maxFrame  int
}

func NewRunnerClient() *RunnerClient {
    tr := strings.ToLower(strings.TrimSpace(cfg.RunnerTransport))
    if tr == "" || tr == "auto" {
        if runtime.GOOS == "windows" {
            tr = "tcp"
        } else {
            tr = "unix"
        }
    }
    return &RunnerClient{
        transport: tr,
        addr: cfg.RunnerAddr,
        maxFrame: cfg.RunnerMaxFrameBytes,
    }
}

func (c *RunnerClient) Exec(ctx context.Context, reqID, action string, params map[string]any, deviceID, remote string) (runnerExecResponse, error) {
    d := net.Dialer{Timeout: 2 * time.Second}
    conn, err := d.DialContext(ctx, c.transport, c.addr)
    if err != nil {
        return runnerExecResponse{}, err
    }
    defer conn.Close()

    _ = conn.SetDeadline(time.Now().Add(30 * time.Second))

    br := bufio.NewReader(conn)
    bw := bufio.NewWriter(conn)

    req := runnerExecRequest{
        V: 1,
        Req: reqID,
        Action: action,
        Params: params,
        InvokedBy: &struct {
            DeviceID string `json:"device_id,omitempty"`
            Remote   string `json:"remote,omitempty"`
        }{DeviceID: deviceID, Remote: remote},
    }

    b, _ := json.Marshal(req)
    if err := writeFrame(bw, c.maxFrame, b); err != nil {
        return runnerExecResponse{}, err
    }
    if err := bw.Flush(); err != nil {
        return runnerExecResponse{}, err
    }

    respBytes, err := readFrame(br, c.maxFrame)
    if err != nil {
        return runnerExecResponse{}, err
    }

    var resp runnerExecResponse
    if err := json.Unmarshal(respBytes, &resp); err != nil {
        return runnerExecResponse{}, err
    }
    if resp.V != 1 || resp.Req != reqID {
        return runnerExecResponse{}, fmt.Errorf("bad runner response")
    }
    return resp, nil
}

func readFrame(r *bufio.Reader, max int) ([]byte, error) {
    var n uint32
    if err := binary.Read(r, binary.BigEndian, &n); err != nil {
        return nil, err
    }
    if n == 0 || int(n) > max {
        return nil, fmt.Errorf("frame too large")
    }
    b := make([]byte, n)
    _, err := io.ReadFull(r, b)
    return b, err
}

func writeFrame(w *bufio.Writer, max int, payload []byte) error {
    if len(payload) == 0 || len(payload) > max {
        return fmt.Errorf("frame too large")
    }
    if err := binary.Write(w, binary.BigEndian, uint32(len(payload))); err != nil {
        return err
    }
    _, err := w.Write(payload)
    return err
}

