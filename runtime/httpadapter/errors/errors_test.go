package errors

import (
	"context"
	stderrors "errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/ca-x/bufplugins/runtime/httpadapter"
)

func TestDefaultMapperMapsCanceledToStandardHTTPStatus(t *testing.T) {
	mapped := DefaultMapper{}.MapError(
		context.Background(),
		httpadapter.MethodSpec{},
		connect.NewError(connect.CodeCanceled, stderrors.New("client canceled")),
	)
	if mapped == nil {
		t.Fatal("MapError() = nil, want HTTP error")
	}
	if mapped.Status != http.StatusRequestTimeout {
		t.Fatalf("status = %d, want %d", mapped.Status, http.StatusRequestTimeout)
	}
}
