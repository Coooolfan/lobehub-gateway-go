package gateway

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConfigRequiresServiceTokenAndRejectsEmptyBearer(t *testing.T) {
	if err := (Config{}).Validate(); err == nil {
		t.Fatal("expected empty SERVICE_TOKEN config to fail validation")
	}
	if err := (Config{ServiceToken: "service-token"}).Validate(); err != nil {
		t.Fatalf("expected populated SERVICE_TOKEN config to pass validation: %v", err)
	}

	srv := NewServer(Config{})
	httpSrv := httptest.NewServer(srv.Routes())
	defer httpSrv.Close()

	res := postJSON(t, httpSrv.URL+"/api/device/status", "", `{"userId":"u1"}`)
	assertStatus(t, res, http.StatusUnauthorized)
	assertBody(t, res, "Unauthorized")
}

func TestHTTPAuthAndOfflineResponses(t *testing.T) {
	srv := NewServer(Config{ServiceToken: "service-token"})
	httpSrv := httptest.NewServer(srv.Routes())
	defer httpSrv.Close()

	res := postJSON(t, httpSrv.URL+"/api/device/status", "wrong", `{"userId":"u1"}`)
	assertStatus(t, res, http.StatusUnauthorized)
	assertBody(t, res, "Unauthorized")

	res = postJSON(t, httpSrv.URL+"/api/device/status", "service-token", `{}`)
	assertStatus(t, res, http.StatusBadRequest)
	assertBody(t, res, "Missing userId")

	res = postJSON(t, httpSrv.URL+"/api/device/status", "service-token", `{"userId":"u1"}`)
	assertStatus(t, res, http.StatusOK)
	assertJSON(t, res, map[string]any{"deviceCount": float64(0), "online": false})

	res = postJSON(t, httpSrv.URL+"/api/device/tool-call", "service-token", `{"userId":"u1","toolCall":{"identifier":"x"}}`)
	assertStatus(t, res, http.StatusServiceUnavailable)
	assertJSON(t, res, map[string]any{"content": "Desktop device offline", "error": "DEVICE_OFFLINE", "success": false})

	res = postJSON(t, httpSrv.URL+"/api/device/system-info", "service-token", `{"userId":"u1"}`)
	assertStatus(t, res, http.StatusServiceUnavailable)
	assertJSON(t, res, map[string]any{"error": "DEVICE_OFFLINE", "success": false})

	res = postJSON(t, httpSrv.URL+"/api/device/agent/run", "service-token", `{"userId":"u1","operationId":"op"}`)
	assertStatus(t, res, http.StatusServiceUnavailable)
	assertJSON(t, res, map[string]any{"error": "DEVICE_OFFLINE", "success": false})
}

func TestWebSocketServiceTokenHeartbeatAndRPC(t *testing.T) {
	srv := NewServer(Config{ServiceToken: "service-token"})
	srv.authTimeout = time.Second
	srv.heartbeatTimeout = 2 * time.Second
	httpSrv := httptest.NewServer(srv.Routes())
	defer httpSrv.Close()

	ws := dialTestWS(t, httpSrv.URL, "/ws?userId=u1&deviceId=d1&hostname=host&platform=linux")
	defer ws.close()

	ws.sendJSON(t, map[string]any{"type": "auth", "token": "service-token"})
	assertWSJSON(t, ws, map[string]any{"type": "auth_success"})

	ws.sendJSON(t, map[string]any{"type": "heartbeat"})
	assertWSJSON(t, ws, map[string]any{"type": "heartbeat_ack"})

	res := postJSON(t, httpSrv.URL+"/api/device/devices", "service-token", `{"userId":"u1"}`)
	assertStatus(t, res, http.StatusOK)
	var devicesBody struct {
		Devices []DeviceAttachment `json:"devices"`
	}
	decodeJSON(t, res, &devicesBody)
	if len(devicesBody.Devices) != 1 || devicesBody.Devices[0].DeviceID != "d1" || !devicesBody.Devices[0].Authenticated {
		t.Fatalf("unexpected devices body: %+v", devicesBody)
	}

	resultCh := make(chan map[string]any, 1)
	go func() {
		res := postJSON(t, httpSrv.URL+"/api/device/tool-call", "service-token", `{"userId":"u1","toolCall":{"identifier":"builtin","apiName":"echo","arguments":"{}"},"timeout":1000}`)
		assertStatus(t, res, http.StatusOK)
		var body map[string]any
		decodeJSON(t, res, &body)
		resultCh <- body
	}()

	request := ws.readJSON(t)
	if request["type"] != "tool_call_request" || request["requestId"] == "" {
		t.Fatalf("unexpected tool call request: %#v", request)
	}
	ws.sendJSON(t, map[string]any{
		"type":      "tool_call_response",
		"requestId": request["requestId"],
		"result": map[string]any{
			"content": "ok",
			"success": true,
		},
	})

	select {
	case body := <-resultCh:
		if body["success"] != true || body["content"] != "ok" {
			t.Fatalf("unexpected tool-call response: %#v", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for HTTP RPC result")
	}
}

func TestWebSocketAuthTimeoutAndReplacement(t *testing.T) {
	srv := NewServer(Config{ServiceToken: "service-token"})
	srv.authTimeout = 50 * time.Millisecond
	httpSrv := httptest.NewServer(srv.Routes())
	defer httpSrv.Close()

	ws := dialTestWS(t, httpSrv.URL, "/ws?userId=u1&deviceId=d1")
	msg := ws.readJSON(t)
	if msg["type"] != "auth_failed" || msg["reason"] != "Authentication timeout" {
		t.Fatalf("unexpected auth timeout message: %#v", msg)
	}
	code, reason := ws.readClose(t)
	if code != wsClosePolicy || reason != "Authentication timeout" {
		t.Fatalf("unexpected close: %d %q", code, reason)
	}

	first := dialTestWS(t, httpSrv.URL, "/ws?userId=u1&deviceId=d1")
	defer first.close()
	first.sendJSON(t, map[string]any{"type": "auth", "token": "service-token"})
	assertWSJSON(t, first, map[string]any{"type": "auth_success"})

	second := dialTestWS(t, httpSrv.URL, "/ws?userId=u1&deviceId=d1")
	defer second.close()
	code, reason = first.readClose(t)
	if code != wsCloseNormal || reason != "Replaced by new connection" {
		t.Fatalf("unexpected replacement close: %d %q", code, reason)
	}
}

func postJSON(t *testing.T, url string, token string, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func assertStatus(t *testing.T, res *http.Response, status int) {
	t.Helper()
	if res.StatusCode != status {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected status %d, got %d: %s", status, res.StatusCode, body)
	}
}

func assertBody(t *testing.T, res *http.Response, expected string) {
	t.Helper()
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if string(body) != expected {
		t.Fatalf("expected body %q, got %q", expected, string(body))
	}
}

func assertJSON(t *testing.T, res *http.Response, expected map[string]any) {
	t.Helper()
	var actual map[string]any
	decodeJSON(t, res, &actual)
	if fmt.Sprint(actual) != fmt.Sprint(expected) {
		t.Fatalf("expected JSON %#v, got %#v", expected, actual)
	}
}

func decodeJSON(t *testing.T, res *http.Response, target any) {
	t.Helper()
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func assertWSJSON(t *testing.T, ws *testWS, expected map[string]any) {
	t.Helper()
	actual := ws.readJSON(t)
	if fmt.Sprint(actual) != fmt.Sprint(expected) {
		t.Fatalf("expected WS JSON %#v, got %#v", expected, actual)
	}
}

type testWS struct {
	br   *bufio.Reader
	conn net.Conn
}

func dialTestWS(t *testing.T, serverURL string, path string) *testWS {
	t.Helper()
	addr := strings.TrimPrefix(serverURL, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 16)
	_, _ = rand.Read(key)
	_, _ = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, addr, base64Std(key))
	br := bufio.NewReader(conn)
	res, err := http.ReadResponse(br, nil)
	if err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		t.Fatalf("expected websocket upgrade, got %d", res.StatusCode)
	}
	return &testWS{br: br, conn: conn}
}

func base64Std(value []byte) string {
	const table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var out bytes.Buffer
	for i := 0; i < len(value); i += 3 {
		remaining := len(value) - i
		b0 := value[i]
		var b1, b2 byte
		if remaining > 1 {
			b1 = value[i+1]
		}
		if remaining > 2 {
			b2 = value[i+2]
		}
		out.WriteByte(table[b0>>2])
		out.WriteByte(table[((b0&0x03)<<4)|(b1>>4)])
		if remaining > 1 {
			out.WriteByte(table[((b1&0x0f)<<2)|(b2>>6)])
		} else {
			out.WriteByte('=')
		}
		if remaining > 2 {
			out.WriteByte(table[b2&0x3f])
		} else {
			out.WriteByte('=')
		}
	}
	return out.String()
}

func (w *testWS) sendJSON(t *testing.T, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	w.writeFrame(t, wsTextMessage, payload)
}

func (w *testWS) readJSON(t *testing.T) map[string]any {
	t.Helper()
	opcode, payload := w.readFrame(t)
	if opcode != wsTextMessage {
		t.Fatalf("expected text frame, got opcode %d", opcode)
	}
	var value map[string]any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func (w *testWS) readClose(t *testing.T) (int, string) {
	t.Helper()
	opcode, payload := w.readFrame(t)
	if opcode != wsCloseMessage {
		t.Fatalf("expected close frame, got opcode %d", opcode)
	}
	if len(payload) < 2 {
		return 0, ""
	}
	return int(binary.BigEndian.Uint16(payload[:2])), string(payload[2:])
}

func (w *testWS) writeFrame(t *testing.T, opcode byte, payload []byte) {
	t.Helper()
	mask := [4]byte{1, 2, 3, 4}
	header := []byte{0x80 | opcode}
	length := len(payload)
	if length < 126 {
		header = append(header, 0x80|byte(length))
	} else {
		t.Fatalf("test payload too large: %d", length)
	}
	masked := append([]byte{}, payload...)
	for i := range masked {
		masked[i] ^= mask[i%4]
	}
	_, err := w.conn.Write(append(append(header, mask[:]...), masked...))
	if err != nil {
		t.Fatal(err)
	}
}

func (w *testWS) readFrame(t *testing.T) (byte, []byte) {
	t.Helper()
	_ = w.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	header := make([]byte, 2)
	if _, err := io.ReadFull(w.br, header); err != nil {
		t.Fatal(err)
	}
	opcode := header[0] & 0x0f
	length := int(header[1] & 0x7f)
	if length == 126 {
		buf := make([]byte, 2)
		_, _ = io.ReadFull(w.br, buf)
		length = int(binary.BigEndian.Uint16(buf))
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(w.br, payload); err != nil {
		t.Fatal(err)
	}
	return opcode, payload
}

func (w *testWS) close() {
	_ = w.conn.Close()
}
