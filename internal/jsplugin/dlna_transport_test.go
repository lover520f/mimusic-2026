package jsplugin

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// Exercise chi -> scheduler -> QuickJS, including extension methods and the
// unmodified TCP peer address (forwarded headers must not replace it).
func TestDLNAHTTPTransport(t *testing.T) {
	_, router := loadNetIntegrationTestPluginWithCode(t, "dlna-transport", `
globalThis.onInit = function() {};
globalThis.onDeinit = function() {};
globalThis.onHTTPRequest = function(req) {
  return {statusCode:200,body:JSON.stringify({method:req.method,peer:req.remoteAddr,path:req.path})};
};`)
	for _, method := range []string{"SUBSCRIBE", "UNSUBSCRIBE", "POST"} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/jsplugin/dlna-transport/dlna/AVTransport/event", nil)
			req.RemoteAddr = "192.168.1.22:45678"
			req.Header.Set("X-Forwarded-For", "1.2.3.4")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != 200 {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var result struct{ Method, Peer, Path string }
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Method != method || result.Peer != req.RemoteAddr || result.Path != "/dlna/AVTransport/event" {
				t.Fatalf("wrong forwarded request: %+v", result)
			}
		})
	}
}
