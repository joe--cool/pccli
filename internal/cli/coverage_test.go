package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/services"
)

// Independent versioned API evidence guards against silently dropping a field
// when the typed input, command flags, or remote assignable contract changes.
func TestSongLibraryAssignableFieldCoverage(t *testing.T) {
	var snapshot struct {
		Attributes map[string]map[string]struct {
			Type string `json:"type"`
			Flag string `json:"flag"`
		} `json:"attributes"`
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "contracts", "services-song-library.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &snapshot); err != nil {
		t.Fatal(err)
	}
	inputs := map[string]any{"song:create": services.SongCreate{}, "song:update": services.SongPatch{}, "arrangement:create": services.ArrangementCreate{}, "arrangement:update": services.ArrangementPatch{}, "key:create": services.KeyCreate{}, "key:update": services.KeyPatch{}}
	for key, attrs := range snapshot.Attributes {
		t.Run(key, func(t *testing.T) {
			parts := strings.Split(key, ":")
			action := parts[1]
			if key == "attachment:create" {
				action = "add"
			}
			root := NewRootCommand()
			cmd, _, err := root.Find([]string{"services", parts[0] + "s", action})
			if err != nil {
				t.Fatal(err)
			}
			for field, spec := range attrs {
				flag := cmd.Flags().Lookup(spec.Flag)
				if flag == nil {
					t.Errorf("documented %s field missing CLI flag --%s", field, spec.Flag)
					continue
				}
				expected := map[string]string{"string": "string", "integer": "int", "number": "float64", "boolean": "bool", "array": "stringSlice"}[spec.Type]
				if field == "length" {
					expected = "string"
				}
				if flag.Value.Type() != expected {
					t.Errorf("--%s type %s want %s", spec.Flag, flag.Value.Type(), expected)
				}
			}
			if input, ok := inputs[key]; ok {
				typ := reflect.TypeOf(input)
				seen := map[string]bool{}
				for i := 0; i < typ.NumField(); i++ {
					field := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
					seen[field] = true
					if _, exists := attrs[field]; !exists {
						t.Errorf("typed field %s lacks documented assignable capability", field)
					}
				}
				for field := range attrs {
					if !seen[field] {
						t.Errorf("documented field %s missing typed input", field)
					}
				}
			}
		})
	}
}
