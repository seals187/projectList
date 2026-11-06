package httpapi_test

import (
	"fmt"
	"testing"
)

func TestCustomerPhones(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	type phone struct {
		Label  string `json:"label"`
		Number string `json:"number"`
	}
	type cust struct {
		ID     int64   `json:"id"`
		Phones []phone `json:"phones"`
	}
	var c cust
	expect(t, "anlegen", admin.do("POST", "/api/customers", map[string]any{"name": "A"}, &c), 201)
	if c.Phones == nil || len(c.Phones) != 0 {
		t.Fatalf("neuer kunde: leere liste erwartet, %+v", c.Phones)
	}
	url := fmt.Sprintf("/api/customers/%d", c.ID)

	// empty rows are dropped, values trimmed
	body := map[string]any{"phones": []phone{{" Zentrale ", " +41 41 123 45 67 "}, {"", ""}, {"Mobil", "079/123.45.67"}}}
	expect(t, "telefon speichern", admin.do("PATCH", url, body, &c), 200)
	if len(c.Phones) != 2 || c.Phones[0] != (phone{"Zentrale", "+41 41 123 45 67"}) || c.Phones[1].Number != "079/123.45.67" {
		t.Fatalf("telefon: %+v", c.Phones)
	}
	var got struct {
		Customer cust `json:"customer"`
	}
	admin.do("GET", url, nil, &got)
	if len(got.Customer.Phones) != 2 {
		t.Fatalf("telefon nach laden: %+v", got.Customer.Phones)
	}
	// other fields do not touch the phones
	admin.do("PATCH", url, map[string]any{"location": "Zug"}, &c)
	if len(c.Phones) != 2 {
		t.Fatalf("telefon verloren: %+v", c.Phones)
	}

	expect(t, "ungültige nummer", admin.do("PATCH", url, map[string]any{"phones": []phone{{"X", "abc"}}}, nil), 400)
	many := make([]phone, 11)
	for i := range many {
		many[i] = phone{"", fmt.Sprintf("+41 %d", 100+i)}
	}
	expect(t, "zu viele", admin.do("PATCH", url, map[string]any{"phones": many}, nil), 400)
	expect(t, "leeren", admin.do("PATCH", url, map[string]any{"phones": []phone{}}, &c), 200)
	if len(c.Phones) != 0 {
		t.Fatalf("leeren: %+v", c.Phones)
	}
}

// Customers from before migration 00023 have the column default '[]' stored as
// TEXT: they must still load (regression: everything was empty in the app).
func TestCustomerPhonesExistingRows(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	if _, err := e.db.Exec(`INSERT INTO customers (name) VALUES ('Alt')`); err != nil {
		t.Fatal(err)
	}
	var list []struct {
		Name   string `json:"name"`
		Phones []any  `json:"phones"`
	}
	expect(t, "kundenliste", admin.do("GET", "/api/customers", nil, &list), 200)
	expect(t, "sidebar", admin.do("GET", "/api/sidebar", nil, nil), 200)
	if len(list) != 1 || list[0].Phones == nil || len(list[0].Phones) != 0 {
		t.Fatalf("alter kunde: %+v", list)
	}
}
