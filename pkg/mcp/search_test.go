package mcp

import (
	"sync"
	"testing"
	"time"
)

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{
			name:   "text shorter than max",
			text:   "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "text equal to max",
			text:   "hello world",
			maxLen: 11,
			want:   "hello world",
		},
		{
			name:   "text longer than max",
			text:   "hello world this is a very long text",
			maxLen: 10,
			want:   "hello worl...",
		},
		{
			name:   "empty text",
			text:   "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "zero max len",
			text:   "hello",
			maxLen: 0,
			want:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateText(tt.text, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "simple tags",
			html: "<p>Hello World</p>",
			want: "Hello World",
		},
		{
			name: "nested tags",
			html: "<div><span>Test</span></div>",
			want: "Test",
		},
		{
			name: "multiple lines",
			html: "<html><body><p>Line 1</p><p>Line 2</p></body></html>",
			want: "Line 1Line 2",
		},
		{
			name: "no tags",
			html: "plain text",
			want: "plain text",
		},
		{
			name: "empty",
			html: "",
			want: "",
		},
		{
			name: "consecutive newlines collapsed",
			html: "<p>Line</p><p></p><p>Next</p>",
			want: "LineNext",
		},
		{
			name: "extra spaces trimmed",
			html: "<div>  spaced  </div>",
			want: "spaced",
		},
		{
			name: "unclosed tag",
			html: "<p>unclosed",
			want: "unclosed",
		},
		{
			name: "tag with attributes",
			html: `<a href="http://example.com">link</a>`,
			want: "link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTML(tt.html)
			if got != tt.want {
				t.Errorf("stripHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "basic title",
			html: "<html><head><title>Test Page</title></head></html>",
			want: "Test Page",
		},
		{
			name: "title with case variation",
			html: "<html><head><Title>Test Page</Title></head></html>",
			want: "Test Page",
		},
		{
			name: "no title tag",
			html: "<html><head></head></html>",
			want: "",
		},
		{
			name: "empty title",
			html: "<html><head><title></title></head></html>",
			want: "",
		},
		{
			name: "title with surrounding text",
			html: "prefix<title>My Title</title>suffix",
			want: "My Title",
		},
		{
			name: "title with extra whitespace",
			html: "<title>  Spaced Title  </title>",
			want: "Spaced Title",
		},
		{
			name: "unclosed title",
			html: "<title>unclosed",
			want: "",
		},
		{
			name: "malformed title end",
			html: "<title>Test</span>",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.html)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractBody(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "basic body",
			html: "<html><body><p>Content</p></body></html>",
			want: "<p>Content</p>",
		},
		{
			name: "body with attributes",
			html: `<html><body class="main"><p>Content</p></body></html>`,
			want: "<p>Content</p>",
		},
		{
			name: "no body tag",
			html: "<html><p>Content</p></html>",
			want: "<html><p>Content</p></html>",
		},
		{
			name: "empty body",
			html: "<html><body></body></html>",
			want: "",
		},
		{
			name: "body with newlines",
			html: "<body>\n<p>Line 1</p>\n<p>Line 2</p>\n</body>",
			want: "<p>Line 1</p>\n<p>Line 2</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBody(tt.html)
			if got != tt.want {
				t.Errorf("extractBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTitleFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "basic h1",
			markdown: "# Hello World",
			want:     "Hello World",
		},
		{
			name:     "h1 not at start",
			markdown: "some text\n# Title",
			want:     "Title",
		},
		{
			name:     "no h1",
			markdown: "some text\n## Subtitle",
			want:     "",
		},
		{
			name:     "empty markdown",
			markdown: "",
			want:     "",
		},
		{
			name:     "multiple h1 uses first",
			markdown: "# First\n# Second",
			want:     "First",
		},
		{
			name:     "h1 with leading spaces",
			markdown: "  # Indented Title",
			want:     "Indented Title",
		},
		{
			name:     "h1 with extra whitespace",
			markdown: "#   Spaced Title   ",
			want:     "  Spaced Title",
		},
		{
			name:     "h1 at end with content",
			markdown: "Some intro\n# Main Title",
			want:     "Main Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitleFromMarkdown(tt.markdown)
			if got != tt.want {
				t.Errorf("extractTitleFromMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidDocPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "simple path",
			path: "docs/guide.md",
			want: true,
		},
		{
			name: "absolute path rejected",
			path: "/etc/passwd",
			want: false,
		},
		{
			name: "absolute path with content",
			path: "/docs/guide.md",
			want: false,
		},
		{
			name: "path with parent traversal",
			path: "../etc/passwd",
			want: false,
		},
		{
			name: "path with .. in middle",
			path: "docs/../../etc/passwd",
			want: false,
		},
		{
			name: "empty path",
			path: "",
			want: true,
		},
		{
			name: "just dots",
			path: ".",
			want: true,
		},
		{
			name: "double dots alone",
			path: "..",
			want: false,
		},
		{
			name: "valid nested path",
			path: "a/b/c/d.md",
			want: true,
		},
		{
			name: "path ending in double dots",
			path: "docs/..",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDocPath(tt.path)
			if got != tt.want {
				t.Errorf("isValidDocPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDocCache(t *testing.T) {
	cache := newDocCache()

	t.Run("Get miss", func(t *testing.T) {
		_, ok := cache.Get("missing")
		if ok {
			t.Error("Get() expected false for missing key")
		}
	})

	t.Run("Set and Get", func(t *testing.T) {
		cache.Set("key1", "value1", time.Hour)
		val, ok := cache.Get("key1")
		if !ok {
			t.Error("Get() expected true for existing key")
		}
		if val != "value1" {
			t.Errorf("Get() = %v, want %v", val, "value1")
		}
	})

	t.Run("Expired entry", func(t *testing.T) {
		cache.Set("key2", "value2", -time.Hour)
		_, ok := cache.Get("key2")
		if ok {
			t.Error("Get() expected false for expired key")
		}
	})

	t.Run("Overwrite", func(t *testing.T) {
		cache.Set("key1", "newvalue", time.Hour)
		val, _ := cache.Get("key1")
		if val != "newvalue" {
			t.Errorf("Get() = %v, want %v", val, "newvalue")
		}
	})

	t.Run("Concurrent read", func(t *testing.T) {
		cache.Set("key3", "value3", time.Hour)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var results []interface{}
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				val, ok := cache.Get("key3")
				if ok {
					mu.Lock()
					results = append(results, val)
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if len(results) == 0 {
			t.Error("Concurrent Get() failed")
		}
	})
}
