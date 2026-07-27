package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

var endpoints = map[string]string{
	"order":     env("ORDER_URL", "http://localhost:8080"),
	"inventory": env("INVENTORY_URL", "http://localhost:8081"),
	"payment":   env("PAYMENT_URL", "http://localhost:8082"),
	"shipping":  env("SHIPPING_URL", "http://localhost:8083"),
	"saga":      env("SAGA_URL", "http://localhost:8084"),
}

func TestOrderSaga(t *testing.T) {
	if os.Getenv("E2E") != "1" {
		t.Skip("set E2E=1 to run")
	}
	waitForServices(t)
	tests := []struct {
		name      string
		service   string
		operation string
		status    int
		assert    func(*testing.T, string, orderState)
	}{
		{name: "success", status: 1, assert: assertSuccess},
		{name: "inventory failure", service: "inventory", operation: "inventory.reserve", status: 3, assert: assertInventoryFailure},
		{name: "payment failure", service: "payment", operation: "payment.create", status: 3, assert: assertPaymentFailure},
		{name: "shipping failure", service: "shipping", operation: "shipping.create", status: 3, assert: assertShippingFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearFailures(t)
			if test.service != "" {
				request(t, http.MethodPost, endpoints[test.service]+"/__test/failures", map[string]string{"operation": test.operation}, http.StatusNoContent, nil)
			}
			orderID := uuid.NewString()
			request(t, http.MethodPost, endpoints["order"]+"/orders", map[string]any{
				"uuid":  orderID,
				"items": []map[string]any{{"productUuid": "product-1", "name": "Coffee", "price": 10.0, "quantity": 2}},
			}, http.StatusAccepted, nil)
			order := waitForOrder(t, orderID, test.status)
			test.assert(t, orderID, order)
		})
	}
	clearFailures(t)
}

type orderState struct {
	UUID         string
	PaymentUUID  string
	ShippingUUID string
	Status       int
}

func assertSuccess(t *testing.T, orderID string, order orderState) {
	t.Helper()
	if order.PaymentUUID == "" || order.ShippingUUID == "" {
		t.Fatalf("missing completion references: %+v", order)
	}
	assertState(t, endpoints["inventory"]+"/reservations/"+orderID, 0)
	assertState(t, endpoints["payment"]+"/payments/"+order.PaymentUUID, 2)
	assertState(t, endpoints["shipping"]+"/shippings/"+order.ShippingUUID, 1)
}

func assertInventoryFailure(t *testing.T, orderID string, _ orderState) {
	t.Helper()
	request(t, http.MethodGet, endpoints["inventory"]+"/reservations/"+orderID, nil, http.StatusNotFound, nil)
}

func assertPaymentFailure(t *testing.T, orderID string, _ orderState) {
	t.Helper()
	assertState(t, endpoints["inventory"]+"/reservations/"+orderID, 1)
	paymentID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("payment:"+orderID)).String()
	request(t, http.MethodGet, endpoints["payment"]+"/payments/"+paymentID, nil, http.StatusNotFound, nil)
}

func assertShippingFailure(t *testing.T, orderID string, _ orderState) {
	t.Helper()
	assertState(t, endpoints["inventory"]+"/reservations/"+orderID, 1)
	paymentID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("payment:"+orderID)).String()
	assertState(t, endpoints["payment"]+"/payments/"+paymentID, 3)
	shippingID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("shipping:"+orderID)).String()
	request(t, http.MethodGet, endpoints["shipping"]+"/shippings/"+shippingID, nil, http.StatusNotFound, nil)
}

func waitForOrder(t *testing.T, orderID string, status int) orderState {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		var order orderState
		if get(endpoints["order"]+"/orders/"+orderID, &order) == http.StatusOK && order.Status == status {
			return order
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("order %s did not reach status %d", orderID, status)
	return orderState{}
}

func waitForServices(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		ready := true
		for _, endpoint := range endpoints {
			if get(endpoint+"/health", nil) != http.StatusNoContent {
				ready = false
				break
			}
		}
		if ready {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("services did not become healthy")
}

func assertState(t *testing.T, url string, want int) {
	t.Helper()
	var state struct{ Status int }
	request(t, http.MethodGet, url, nil, http.StatusOK, &state)
	if state.Status != want {
		t.Fatalf("%s status: got %d want %d", url, state.Status, want)
	}
}

func clearFailures(t *testing.T) {
	t.Helper()
	for _, service := range []string{"inventory", "payment", "shipping", "order"} {
		request(t, http.MethodDelete, endpoints[service]+"/__test/failures", nil, http.StatusNoContent, nil)
	}
}

func request(t *testing.T, method, url string, body any, want int, target any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != want {
		data, _ := io.ReadAll(response.Body)
		t.Fatalf("%s %s: got %d want %d: %s", method, url, response.StatusCode, want, data)
	}
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func get(url string, target any) int {
	response, err := http.Get(url)
	if err != nil {
		return 0
	}
	defer response.Body.Close()
	if target != nil && response.StatusCode == http.StatusOK {
		_ = json.NewDecoder(response.Body).Decode(target)
	}
	return response.StatusCode
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
