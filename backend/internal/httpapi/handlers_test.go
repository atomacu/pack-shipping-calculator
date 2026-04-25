package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"pack-shipping-calculator/backend/internal/packing"
	"pack-shipping-calculator/backend/internal/packs"
)

func TestHealth(t *testing.T) {
	response := request(NewRouter(&fakePackService{}), http.MethodGet, "/healthz", "")

	assertStatus(t, response, http.StatusOK)
	assertJSONContentType(t, response)
	assertCORSHeaders(t, response)

	var got map[string]string
	decodeResponse(t, response, &got)

	want := map[string]string{"status": "ok"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestCORSPreflight(t *testing.T) {
	response := request(NewRouter(&fakePackService{}), http.MethodOptions, "/api/v1/packs", "")

	assertStatus(t, response, http.StatusNoContent)
	assertCORSHeaders(t, response)
	if got := response.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PUT, OPTIONS" {
		t.Fatalf("got Access-Control-Allow-Methods %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Fatalf("got Access-Control-Allow-Headers %q", got)
	}
}

func TestGetPacks(t *testing.T) {
	response := request(NewRouter(&fakePackService{sizes: []int{250, 500}}), http.MethodGet, "/api/v1/packs", "")

	assertStatus(t, response, http.StatusOK)
	assertJSONContentType(t, response)

	var got packsResponse
	decodeResponse(t, response, &got)

	want := packsResponse{PackSizes: []int{250, 500}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestGetPacksRepositoryError(t *testing.T) {
	response := request(NewRouter(&fakePackService{getErr: packs.ErrRepository}), http.MethodGet, "/api/v1/packs", "")

	assertStatus(t, response, http.StatusInternalServerError)
	assertJSONContentType(t, response)
	assertError(t, response, "internal_error", "could not load pack sizes")
}

func TestReplacePacksSuccess(t *testing.T) {
	service := &fakePackService{sizes: []int{250, 500}}
	response := request(NewRouter(service), http.MethodPut, "/api/v1/packs", `{"pack_sizes":[500,250,500]}`)

	assertStatus(t, response, http.StatusOK)
	assertJSONContentType(t, response)

	var got packsResponse
	decodeResponse(t, response, &got)

	want := packsResponse{PackSizes: []int{250, 500}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}

	if !reflect.DeepEqual(service.replacedSizes, []int{500, 250, 500}) {
		t.Fatalf("got replaced sizes %#v", service.replacedSizes)
	}
}

func TestReplacePacksMalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `{`},
		{name: "unknown field", body: `{"pack_sizes":[250],"unknown":true}`},
		{name: "extra JSON value", body: `{"pack_sizes":[250]} {"pack_sizes":[500]}`},
		{name: "invalid trailing content", body: `{"pack_sizes":[250]} trailing`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := request(NewRouter(&fakePackService{}), http.MethodPut, "/api/v1/packs", tt.body)

			assertStatus(t, response, http.StatusBadRequest)
			assertJSONContentType(t, response)
			assertError(t, response, "malformed_json", "request body must be valid JSON")
		})
	}
}

func TestReplacePacksInvalidPackSizes(t *testing.T) {
	response := request(NewRouter(&fakePackService{replaceErr: packs.ErrInvalidPackSizes}), http.MethodPut, "/api/v1/packs", `{"pack_sizes":[]}`)

	assertStatus(t, response, http.StatusBadRequest)
	assertJSONContentType(t, response)
	assertError(t, response, "invalid_pack_sizes", "pack sizes must contain at least one positive integer")
}

func TestReplacePacksRepositoryError(t *testing.T) {
	response := request(NewRouter(&fakePackService{replaceErr: packs.ErrRepository}), http.MethodPut, "/api/v1/packs", `{"pack_sizes":[250]}`)

	assertStatus(t, response, http.StatusInternalServerError)
	assertJSONContentType(t, response)
	assertError(t, response, "internal_error", "could not save pack sizes")
}

func TestCalculateOrderSuccess(t *testing.T) {
	plan := packing.Plan{
		ItemsOrdered: 12001,
		ItemsShipped: 12250,
		ItemsOver:    249,
		TotalPacks:   4,
		Packs: []packing.PackLine{
			{Size: 5000, Quantity: 2},
			{Size: 2000, Quantity: 1},
			{Size: 250, Quantity: 1},
		},
	}
	response := request(NewRouter(&fakePackService{plan: plan}), http.MethodPost, "/api/v1/orders/calculate", `{"items":12001}`)

	assertStatus(t, response, http.StatusOK)
	assertJSONContentType(t, response)

	var got calculateOrderResponse
	decodeResponse(t, response, &got)

	if !reflect.DeepEqual(got, calculateOrderResponse(plan)) {
		t.Fatalf("got %#v, want %#v", got, calculateOrderResponse(plan))
	}
}

func TestCalculateOrderMalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `{`},
		{name: "unknown field", body: `{"items":1,"unknown":true}`},
		{name: "extra JSON value", body: `{"items":1} {"items":2}`},
		{name: "invalid trailing content", body: `{"items":1} trailing`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := request(NewRouter(&fakePackService{}), http.MethodPost, "/api/v1/orders/calculate", tt.body)

			assertStatus(t, response, http.StatusBadRequest)
			assertJSONContentType(t, response)
			assertError(t, response, "malformed_json", "request body must be valid JSON")
		})
	}
}

func TestCalculateOrderInvalidOrder(t *testing.T) {
	response := request(NewRouter(&fakePackService{calculateErr: packing.ErrInvalidOrderSize}), http.MethodPost, "/api/v1/orders/calculate", `{"items":0}`)

	assertStatus(t, response, http.StatusBadRequest)
	assertJSONContentType(t, response)
	assertError(t, response, "invalid_order", invalidOrderMessage())
}

func TestCalculateOrderTooLarge(t *testing.T) {
	response := request(NewRouter(&fakePackService{calculateErr: packing.ErrOrderTooLarge}), http.MethodPost, "/api/v1/orders/calculate", `{"items":1000001}`)

	assertStatus(t, response, http.StatusBadRequest)
	assertJSONContentType(t, response)
	assertError(t, response, "invalid_order", invalidOrderMessage())
}

func TestCalculateOrderInvalidPackSizes(t *testing.T) {
	response := request(NewRouter(&fakePackService{calculateErr: packing.ErrNoPackSizes}), http.MethodPost, "/api/v1/orders/calculate", `{"items":1}`)

	assertStatus(t, response, http.StatusBadRequest)
	assertJSONContentType(t, response)
	assertError(t, response, "invalid_pack_sizes", "pack sizes must contain at least one positive integer")
}

func TestCalculateOrderRepositoryError(t *testing.T) {
	response := request(NewRouter(&fakePackService{calculateErr: packs.ErrRepository}), http.MethodPost, "/api/v1/orders/calculate", `{"items":1}`)

	assertStatus(t, response, http.StatusInternalServerError)
	assertJSONContentType(t, response)
	assertError(t, response, "internal_error", "could not calculate order")
}

type fakePackService struct {
	sizes         []int
	replacedSizes []int
	plan          packing.Plan
	getErr        error
	replaceErr    error
	calculateErr  error
}

func (s *fakePackService) GetPackSizes(context.Context) ([]int, error) {
	return s.sizes, s.getErr
}

func (s *fakePackService) ReplacePackSizes(_ context.Context, sizes []int) ([]int, error) {
	s.replacedSizes = sizes
	return s.sizes, s.replaceErr
}

func (s *fakePackService) CalculateOrder(context.Context, int) (packing.Plan, error) {
	return s.plan, s.calculateErr
}

func request(handler http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("got status %d, want %d; body: %s", response.Code, want, response.Body.String())
	}
}

func assertJSONContentType(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json", got)
	}
}

func assertCORSHeaders(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("got Access-Control-Allow-Origin %q", got)
	}
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, code string, message string) {
	t.Helper()

	var got errorResponse
	decodeResponse(t, response, &got)

	want := errorResponse{Error: apiError{Code: code, Message: message}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, value any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(value); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
