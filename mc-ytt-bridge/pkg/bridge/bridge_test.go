package bridge_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GrahamDumpleton/fat-controller/mc-ytt-bridge/pkg/bridge"
	"github.com/go-chi/chi"
)

func TestBridgeExecuteCommandEcho(t *testing.T) {
	var handlersDirectory = "../../test/handlers"

	br := bridge.YttBridge{
		HandlersDirectory: handlersDirectory,
	}

	bre := bridge.YttBridgeEndpoint{
		HandlerName:  "testing",
		ResourceName: "tests",
		FunctionName: "echo",
	}

	input := map[string]interface{}{
		"key-1": "value-1",
	}

	body, _ := json.Marshal(input)

	brr := bridge.YttBridgeRequest{
		RequestMethod: "POST",
		QueryParams:   map[string][]string{},
		RequestBody:   body,
	}

	res, err := br.ExecuteCommand(bre, brr)

	if err != nil {
		t.Fatalf("execution of ytt failed: %v", err)
	}

	var output map[string]interface{}

	if err := json.Unmarshal(res, &output); err != nil {
		t.Fatalf("unable to decode response: %v", err)
	}

	// Compare as JSON as comparing struct doesn't seem to work. The JSON
	// output should always sort keys so direct string comparison is fine.

	s1, _ := json.Marshal(output)
	s2, _ := json.Marshal(input)

	if !bytes.Equal(s1, s2) {
		t.Fatalf("expected output does not match, expected %s, got %s", s2, s1)
	}
}

func TestBridgeExecuteCommandValues(t *testing.T) {
	var handlersDirectory = "../../test/handlers"
	var configurationFile = "../../test/config.yaml"

	br := bridge.YttBridge{
		HandlersDirectory: handlersDirectory,
		ConfigurationFile: configurationFile,
	}

	bre := bridge.YttBridgeEndpoint{
		HandlerName:  "testing",
		ResourceName: "tests",
		FunctionName: "values",
	}

	input := map[string]interface{}{
		"key-1": "value-1",
	}

	body, _ := json.Marshal(input)

	brr := bridge.YttBridgeRequest{
		RequestMethod: "POST",
		QueryParams:   map[string][]string{},
		RequestBody:   body,
	}

	result, err := br.ExecuteCommand(bre, brr)

	if err != nil {
		t.Fatalf("execution of ytt failed: %v", err)
	}

	var output map[string]interface{}

	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unable to decode response: %v", err)
	}

	expect := map[string]interface{}{
		"key-1": "value-1",
	}

	// Compare as JSON as comparing struct doesn't seem to work. The JSON
	// output should always sort keys so direct string comparison is fine.

	s1, _ := json.Marshal(output)
	s2, _ := json.Marshal(expect)

	if !bytes.Equal(s1, s2) {
		t.Fatalf("expected output does not match, expected %s, got %s", s2, s1)
	}
}

func TestBridgeHandleRequestEcho(t *testing.T) {
	var handlersDirectory = "../../test/handlers"

	br := bridge.YttBridge{
		HandlersDirectory: handlersDirectory,
	}

	w := httptest.NewRecorder()

	input := map[string]interface{}{
		"key-1": "value-1",
	}

	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

	req.Header.Set("content-type", "application/json")

	rctx := chi.NewRouteContext()

	rctx.URLParams.Add("handler", "testing")
	rctx.URLParams.Add("resource", "tests")
	rctx.URLParams.Add("function", "echo")

	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	br.HandleRequest(w, req)

	res := w.Result()

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		t.Fatalf("unable to read response body: %v", err)
	}

	var output map[string]interface{}

	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("unable to decode response body: %v", err)
	}

	s1, _ := json.Marshal(output)

	if !bytes.Equal(s1, body) {
		t.Fatalf("expected output does not match, expected %s, got %s", body, s1)
	}
}
