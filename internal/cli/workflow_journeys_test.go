package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/config"
	"github.com/joe--cool/pccli/v2/internal/planningcenter"
	"github.com/joe--cool/pccli/v2/internal/services"
)

type journeyResource struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Attributes    map[string]any `json:"attributes"`
	Relationships map[string]any `json:"relationships,omitempty"`
}
type journeyEffect struct {
	Method, Path string
	Attributes   map[string]any
}
type journeyServer struct {
	t       *testing.T
	records map[string]journeyResource
	next    map[string]int
	effects []journeyEffect
}

func newJourneyServer(t *testing.T) *journeyServer {
	return &journeyServer{t: t, records: map[string]journeyResource{}, next: map[string]int{"Song": 1000, "Arrangement": 2000, "Key": 3000, "Attachment": 4000}}
}
func (j *journeyServer) response(status int, data any) *http.Response {
	j.t.Helper()
	var body []byte
	if data != nil {
		var err error
		body, err = json.Marshal(map[string]any{"data": data})
		if err != nil {
			j.t.Fatal(err)
		}
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}
}
func (j *journeyServer) RoundTrip(r *http.Request) (*http.Response, error) {
	j.t.Helper()
	user, secret, ok := r.BasicAuth()
	if !ok || user != "journey-client" || secret != "journey-secret" {
		j.t.Fatal("synthetic credentials missing")
	}
	if r.URL.Host != "api.planningcenteronline.com" || r.Header.Get("X-PCO-API-Version") != planningcenter.ServicesAPIVersion {
		j.t.Fatalf("unscoped request %s", r.URL)
	}
	p := r.URL.Path
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) < 3 || parts[0] != "services" || parts[1] != "v2" || parts[2] != "songs" {
		j.t.Fatalf("unexpected endpoint %s", p)
	}
	collection := p
	isCollection := len(parts)%2 == 1
	if !isCollection {
		collection = path.Dir(p)
	}
	collectionParts := strings.Split(strings.Trim(collection, "/"), "/")
	kind := map[string]string{"songs": "Song", "arrangements": "Arrangement", "keys": "Key", "attachments": "Attachment"}[collectionParts[len(collectionParts)-1]]
	if kind == "" {
		j.t.Fatalf("unexpected collection %s", collection)
	}
	if collection != "/services/v2/songs" {
		if _, ok := j.records[path.Dir(collection)]; !ok {
			j.t.Fatalf("requested child of missing parent: %s", p)
		}
	}
	switch r.Method {
	case http.MethodGet:
		if !isCollection {
			resource, ok := j.records[p]
			if !ok {
				return j.response(404, nil), nil
			}
			return j.response(200, resource), nil
		}
		ids := []string{}
		for resourcePath := range j.records {
			if path.Dir(resourcePath) == collection {
				ids = append(ids, resourcePath)
			}
		}
		sort.Strings(ids)
		result := []journeyResource{}
		for _, id := range ids {
			record := j.records[id]
			if kind == "Song" {
				hidden, _ := record.Attributes["hidden"].(bool)
				if value := r.URL.Query().Get("where[hidden]"); value != "" && strconv.FormatBool(hidden) != value {
					continue
				}
			}
			result = append(result, record)
		}
		return j.response(200, result), nil
	case http.MethodPost, http.MethodPatch:
		var envelope struct {
			Data struct {
				Type       string         `json:"type"`
				Attributes map[string]any `json:"attributes"`
			} `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			j.t.Fatal(err)
		}
		if envelope.Data.Type != kind || len(envelope.Data.Attributes) == 0 {
			j.t.Fatalf("invalid mutation envelope %+v", envelope)
		}
		j.effects = append(j.effects, journeyEffect{r.Method, p, envelope.Data.Attributes})
		var record journeyResource
		if r.Method == http.MethodPost {
			if !isCollection {
				j.t.Fatalf("POST to resource %s", p)
			}
			j.next[kind]++
			record = journeyResource{ID: strconv.Itoa(j.next[kind]), Type: kind, Attributes: map[string]any{}, Relationships: map[string]any{}}
			switch kind {
			case "Song":
				record.Attributes["hidden"] = false
			case "Arrangement":
				record.Attributes["archived_at"] = nil
				record.Attributes["sequence"] = []string{}
			}
			parentKind, parentID, relation := "", "", ""
			switch kind {
			case "Arrangement":
				parentKind, parentID, relation = "Song", parts[3], "song"
			case "Key":
				parentKind, parentID, relation = "Arrangement", parts[5], "arrangement"
			case "Attachment":
				relation = "attachable"
				switch parts[len(parts)-3] {
				case "songs":
					parentKind, parentID = "Song", parts[len(parts)-2]
				case "arrangements":
					parentKind, parentID = "Arrangement", parts[len(parts)-2]
				case "keys":
					parentKind, parentID = "Key", parts[len(parts)-2]
				}
			}
			if relation != "" {
				record.Relationships[relation] = map[string]any{"data": map[string]string{"type": parentKind, "id": parentID}}
			}
			p = collection + "/" + record.ID
		} else {
			var ok bool
			record, ok = j.records[p]
			if !ok {
				j.t.Fatalf("PATCH missing resource %s", p)
			}
		}
		for name, value := range envelope.Data.Attributes {
			record.Attributes[name] = value
		}
		// The documented CCLI-title import returns fetched metadata, rather than
		// echoing the numeric title supplied as an import instruction.
		if kind == "Song" && r.Method == http.MethodPost {
			title, _ := record.Attributes["title"].(string)
			if code, err := strconv.Atoi(title); err == nil && code > 0 {
				record.Attributes["title"] = "Fetched Song Title"
				record.Attributes["author"] = "Fetched Writer"
			}
		}
		j.records[p] = record
		return j.response(map[bool]int{true: 201, false: 200}[r.Method == http.MethodPost], record), nil
	case http.MethodDelete:
		if _, ok := j.records[p]; !ok {
			j.t.Fatalf("DELETE missing resource %s", p)
		}
		j.effects = append(j.effects, journeyEffect{Method: r.Method, Path: p})
		delete(j.records, p)
		return j.response(204, nil), nil
	}
	j.t.Fatalf("unexpected method %s", r.Method)
	return nil, nil
}
func (j *journeyServer) run(machine bool, args ...string) string {
	j.t.Helper()
	cfg := config.Config{BaseURL: "https://api.planningcenteronline.com", ClientID: "journey-client", ClientSecret: "journey-secret", Color: "never"}
	command := injectedCommand(services.NewLibrary(planningcenter.NewClient(cfg, j)))
	argv := append([]string{"services"}, args...)
	if machine {
		argv = append(argv, "--json")
	}
	out, stderr, err := runCommand(command, argv...)
	if err != nil {
		j.t.Fatalf("%s: %v stderr %s", strings.Join(argv, " "), err, stderr)
	}
	if machine {
		decodeOutput(j.t, out)
	} else if out == "" || strings.Contains(out, "\x1b") {
		j.t.Fatalf("invalid plain output %q", out)
	}
	return out
}
func (j *journeyServer) assertLast(method, p string, want map[string]any) {
	j.t.Helper()
	if len(j.effects) == 0 {
		j.t.Fatal("expected mutation")
	}
	last := j.effects[len(j.effects)-1]
	if last.Method != method || last.Path != p || !reflect.DeepEqual(last.Attributes, want) {
		j.t.Fatalf("effect %+v want %s %s %#v", last, method, p, want)
	}
}

func (j *journeyServer) assertResult(machine bool, out string, expected map[string]string, nested ...string) {
	j.t.Helper()
	if !machine {
		for _, value := range expected {
			if value != "" && !strings.Contains(out, value) {
				j.t.Fatalf("plain result omitted %q: %s", value, out)
			}
		}
		return
	}
	document := decodeOutput(j.t, out)
	var result map[string]any
	if err := json.Unmarshal(document["data"], &result); err != nil {
		j.t.Fatal(err)
	}
	for _, name := range nested {
		var ok bool
		result, ok = result[name].(map[string]any)
		if !ok {
			j.t.Fatalf("missing %s in %s", name, out)
		}
	}
	for field, want := range expected {
		if result[field] != want {
			j.t.Fatalf("result %s=%v want %q: %s", field, result[field], want, out)
		}
	}
}

func TestCanonicalLibraryLifecycleJourneys(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(fmt.Sprintf("JSON=%t", machine), func(t *testing.T) {
			j := newJourneyServer(t)
			out := j.run(machine, "songs", "create", "--title", "Journey Song", "--author", "Original Writer")
			j.assertResult(machine, out, map[string]string{"id": "1001", "title": "Journey Song"})
			j.assertLast("POST", "/services/v2/songs", map[string]any{"title": "Journey Song", "author": "Original Writer"})
			songPath := "/services/v2/songs/1001"
			if j.records[songPath].Attributes["hidden"] != false {
				t.Fatal("created song missing visible state")
			}
			j.run(machine, "songs", "show", "Journey Song")
			j.run(machine, "songs", "list", "--search", "Journey")
			j.run(machine, "songs", "update", "Journey Song", "--author=", "--hidden=true")
			j.assertLast("PATCH", songPath, map[string]any{"author": "", "hidden": true})
			j.run(machine, "songs", "show", "1001", "--visibility", "hidden")
			j.run(machine, "songs", "update", "1001", "--visibility", "hidden", "--hidden=false")
			j.assertLast("PATCH", songPath, map[string]any{"hidden": false})

			out = j.run(machine, "arrangements", "create", "--song", "Journey Song", "--name", "Acoustic", "--notes", "Verse and chorus", "--chart-key", "G", "--sequence", "Verse 1,Chorus 1")
			j.assertResult(machine, out, map[string]string{"id": "2001", "song_id": "1001", "name": "Acoustic"})
			arrPath := songPath + "/arrangements/2001"
			j.assertLast("POST", songPath+"/arrangements", map[string]any{"name": "Acoustic", "notes": "Verse and chorus", "chord_chart_key": "G", "sequence": []any{"Verse 1", "Chorus 1"}})
			j.run(machine, "arrangements", "list", "--song", "1001")
			j.run(machine, "arrangements", "show", "Acoustic", "--song", "1001")
			j.run(machine, "arrangements", "update", "Acoustic", "--song", "Journey Song", "--notes=", "--length", "0", "--sequence=")
			j.assertLast("PATCH", arrPath, map[string]any{"notes": "", "length": float64(0), "sequence": []any{}})

			out = j.run(machine, "keys", "create", "--song", "1001", "--arrangement", "Acoustic", "--name", "Lead", "--start", "Bb")
			j.assertResult(machine, out, map[string]string{"id": "3001", "song_id": "1001", "arrangement_id": "2001", "starting_key": "Bb", "ending_key": "Bb"})
			keyPath := arrPath + "/keys/3001"
			j.assertLast("POST", arrPath+"/keys", map[string]any{"name": "Lead", "starting_key": "Bb", "ending_key": "Bb"})
			j.run(machine, "keys", "list", "--song", "1001", "--arrangement", "2001")
			j.run(machine, "keys", "show", "Lead", "--song", "Journey Song", "--arrangement", "Acoustic")
			j.run(machine, "keys", "update", "Lead", "--song", "1001", "--arrangement", "2001", "--end", "C")
			j.assertLast("PATCH", keyPath, map[string]any{"ending_key": "C"})

			// Identical filenames in all three scopes must remain independent targets.
			scopes := [][]string{{"--song", "1001"}, {"--song", "1001", "--arrangement", "2001"}, {"--song", "1001", "--arrangement", "2001", "--key", "3001"}}
			parents := []string{songPath, arrPath, keyPath}
			for index, scope := range scopes {
				out = j.run(machine, append([]string{"attachments", "add", "--upload-id", "known-uuid-" + strconv.Itoa(index), "--filename", "chart.pdf"}, scope...)...)
				expected := map[string]string{"id": strconv.Itoa(4001 + index), "song_id": "1001", "filename": "chart.pdf"}
				if index > 0 {
					expected["arrangement_id"] = "2001"
				}
				if index > 1 {
					expected["key_id"] = "3001"
				}
				j.assertResult(machine, out, expected, "attachment")
				collection := parents[index] + "/attachments"
				j.assertLast("POST", collection, map[string]any{"file_upload_identifier": "known-uuid-" + strconv.Itoa(index), "filename": "chart.pdf"})
			}
			for index, scope := range scopes {
				collection := parents[index] + "/attachments"
				resource := collection + "/" + strconv.Itoa(4001+index)
				j.run(machine, append([]string{"attachments", "list"}, scope...)...)
				j.run(machine, append([]string{"attachments", "show", "chart.pdf"}, scope...)...)
				j.run(machine, append([]string{"attachments", "update", "chart.pdf", "--filename", "updated.pdf"}, scope...)...)
				j.assertLast("PATCH", resource, map[string]any{"filename": "updated.pdf"})
				j.run(machine, append([]string{"attachments", "delete", "updated.pdf", "--yes"}, scope...)...)
				j.assertLast("DELETE", resource, nil)
				if _, ok := j.records[resource]; ok {
					t.Fatal("attachment not deleted")
				}
			}
			j.run(machine, "keys", "delete", "Lead", "--song", "Journey Song", "--arrangement", "Acoustic", "--yes")
			j.assertLast("DELETE", keyPath, nil)
			j.run(machine, "keys", "list", "--song", "1001", "--arrangement", "2001")
			j.run(machine, "arrangements", "delete", "Acoustic", "--song", "Journey Song", "--yes")
			j.assertLast("DELETE", arrPath, nil)
			j.run(machine, "arrangements", "list", "--song", "1001")
			j.run(machine, "songs", "delete", "Journey Song", "--yes")
			j.assertLast("DELETE", songPath, nil)
			j.run(machine, "songs", "list", "--all")
			if len(j.records) != 0 {
				t.Fatalf("lifecycle left resources: %+v", j.records)
			}
		})
	}
}

func TestCCLIOnlyCreateAcceptsFetchedMetadataJourney(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(fmt.Sprintf("JSON=%t", machine), func(t *testing.T) {
			j := newJourneyServer(t)
			out := j.run(machine, "songs", "create", "--ccli", "22025")
			j.assertLast("POST", "/services/v2/songs", map[string]any{"title": "22025", "ccli_number": float64(22025)})
			if !strings.Contains(out, "Fetched Song Title") {
				t.Fatalf("fetched metadata missing: %s", out)
			}
			if machine {
				doc := decodeOutput(t, out)
				var song services.Song
				if err := json.Unmarshal(doc["data"], &song); err != nil || song.Title != "Fetched Song Title" || song.CCLINumber != 22025 {
					t.Fatal(out)
				}
			}
			j.run(machine, "songs", "show", "Fetched Song Title")
			j.run(machine, "songs", "delete", "1001", "--yes")
			if len(j.records) != 0 {
				t.Fatal("CCLI journey left records")
			}
		})
	}
}
