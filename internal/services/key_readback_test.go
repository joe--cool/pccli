package services

import (
	"encoding/json"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func TestKeyReadbackAcceptsAlternateKeyArray(t *testing.T) {
	attrs := json.RawMessage(`{"name":"D","starting_key":"D","ending_key":"D","alternate_keys":[{"name":"Female lead","key":"Eb"}]}`)
	r, err := decodeRecord(planningcenter.Resource[json.RawMessage]{Type: "Key", ID: "1", Attributes: attrs}, "Key")
	if err != nil {
		t.Fatal(err)
	}
	var key Key
	if err := r.model(&key); err != nil {
		t.Fatal(err)
	}
	if key.AlternateKeys != `[{"name":"Female lead","key":"Eb"}]` || key.StartingKey != "D" {
		t.Fatalf("unexpected key readback: %#v", key)
	}
}
