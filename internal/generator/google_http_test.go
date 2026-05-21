package generator

import (
	"reflect"
	"strings"
	"testing"
)

func TestGooglePathToEchoRejectsCollidingParamNames(t *testing.T) {
	_, _, err := googlePathToEcho("/v1/{parent.name}/{parent_name}")
	if err == nil {
		t.Fatal("googlePathToEcho() error = nil, want collision error")
	}
	if !strings.Contains(err.Error(), "colliding route parameter name") {
		t.Fatalf("googlePathToEcho() error = %v, want collision error", err)
	}
}

func TestGooglePathToEchoConvertsNestedWildcardTemplate(t *testing.T) {
	path, params, err := googlePathToEcho("/v1/{name=shelves/*/books/**}")
	if err != nil {
		t.Fatalf("googlePathToEcho() error = %v", err)
	}
	if path != "/v1/shelves/:name/books/*" {
		t.Fatalf("path = %q, want %q", path, "/v1/shelves/:name/books/*")
	}
	want := []pathParam{
		{
			Field:    "name",
			Template: "shelves/*/books/**",
			Names:    []string{"name", "*"},
		},
	}
	if !reflect.DeepEqual(params, want) {
		t.Fatalf("params = %#v, want %#v", params, want)
	}
}

func TestGooglePathToEchoRejectsDeepWildcardBeforeRouteEnd(t *testing.T) {
	_, _, err := googlePathToEcho("/v1/{name=shelves/**}:archive")
	if err == nil {
		t.Fatal("googlePathToEcho() error = nil, want deep wildcard placement error")
	}
	if !strings.Contains(err.Error(), "must end the Echo route") {
		t.Fatalf("googlePathToEcho() error = %v, want route-end error", err)
	}
}
