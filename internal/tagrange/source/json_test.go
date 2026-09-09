package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// TestJSONSource_Fetch_NodejsTransform proves the issue's own exact
// transform expression against a trimmed real-shaped
// nodejs.org/dist/index.json fixture. This is the test that justifies the
// JSONata dependency choice.
func TestJSONSource_Fetch_NodejsTransform(t *testing.T) {
	fixture, err := os.ReadFile("testdata/nodejs-index.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	cfg := &model.SourceConfig{
		Type: "json",
		URL:  server.URL,
		Transform: `$[lts != false].{
			"version": $substring(version, 1),
			"npm": npm,
			"openssl": openssl
		}`,
	}

	versions, err := JSONSource{}.Fetch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The fixture has one non-LTS entry (lts: false) that must be filtered
	// out, leaving 3 LTS releases.
	if len(versions) != 3 {
		t.Fatalf("expected 3 LTS versions, got %d: %+v", len(versions), versions)
	}

	byRaw := make(map[string]Version, len(versions))
	for _, v := range versions {
		byRaw[v.Raw] = v
	}
	want := byRaw["24.11.1"]
	if want.Raw == "" {
		t.Fatalf("expected version 24.11.1 in result, got %+v", versions)
	}
	if want.Extra["npm"] != "11.6.2" {
		t.Errorf("npm extra = %q, want 11.6.2", want.Extra["npm"])
	}
	if want.Extra["openssl"] != "3.5.4" {
		t.Errorf("openssl extra = %q, want 3.5.4", want.Extra["openssl"])
	}
	if _, ok := byRaw["23.5.0"]; ok {
		t.Error("expected the non-LTS version 23.5.0 to be filtered out")
	}
}

func TestJSONSource_Fetch_NoTransform_PlainStringArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`["1.0.0", "1.1.0", "2.0.0"]`))
	}))
	defer server.Close()

	cfg := &model.SourceConfig{Type: "json", URL: server.URL}
	versions, err := JSONSource{}.Fetch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}
}

func TestJSONSource_Fetch_HeaderExpansion(t *testing.T) {
	t.Setenv("MY_TOKEN", "secret-value")
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	cfg := &model.SourceConfig{
		Type:    "json",
		URL:     server.URL,
		Headers: map[string]string{"Authorization": "Bearer $MY_TOKEN"},
	}
	if _, err := (JSONSource{}).Fetch(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer secret-value" {
		t.Errorf("Authorization header = %q, want expanded token", gotAuth)
	}
}

func TestJSONSource_Fetch_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &model.SourceConfig{Type: "json", URL: server.URL}
	if _, err := (JSONSource{}).Fetch(context.Background(), cfg); err == nil {
		t.Error("expected an error for a 500 response")
	}
}

func TestJSONSource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *model.SourceConfig
		wantErr bool
	}{
		{"valid", &model.SourceConfig{Type: "json", URL: "https://example.com"}, false},
		{"missing url", &model.SourceConfig{Type: "json"}, true},
		{"stray image field", &model.SourceConfig{Type: "json", URL: "https://example.com", Image: "library/node"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := JSONSource{}.Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
