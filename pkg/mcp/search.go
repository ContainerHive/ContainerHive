package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	baseURL            = "https://container-hive.timo-reymann.de"
	searchIndexURL     = baseURL + "/search/search_index.json"
	cacheTTL           = time.Hour
	defaultSearchLimit = 10
)

type searchIndex struct {
	Config SearchConfig `json:"config"`
	Docs   []SearchDoc  `json:"docs"`
}

type SearchConfig struct {
	Lang      []string         `json:"lang"`
	Separator string           `json:"separator"`
	Pipeline  []string         `json:"pipeline"`
	Fields    map[string]Field `json:"fields"`
}

type Field struct {
	Boost float64 `json:"boost"`
}

type SearchDoc struct {
	Location string   `json:"location"`
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	Tags     []string `json:"tags,omitempty"`
}

type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

type docCache struct {
	mu   sync.Mutex
	rmu  sync.RWMutex
	cmap map[string]*cacheEntry
}

func newDocCache() *docCache {
	return &docCache{cmap: make(map[string]*cacheEntry)}
}

func (c *docCache) Get(key string) (interface{}, bool) {
	c.rmu.RLock()
	defer c.rmu.RUnlock()
	entry, ok := c.cmap[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *docCache) Set(key string, data interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cmap[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

var (
	searchIndexCache = newDocCache()
	pageCache        = newDocCache()
	httpClient       = &http.Client{Timeout: 30 * time.Second}
)

func searchDocumentation(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	index, err := getSearchIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get search index: %w", err)
	}

	queryLower := strings.ToLower(query)
	var scoredResults []struct {
		doc   SearchDoc
		score float64
	}

	for _, doc := range index.Docs {
		titleLower := strings.ToLower(doc.Title)
		textLower := strings.ToLower(doc.Text)

		var score float64
		if strings.Contains(titleLower, queryLower) {
			score += 1000.0
		}
		if strings.Contains(textLower, queryLower) {
			score += 1.0
		}

		if score > 0 {
			scoredResults = append(scoredResults, struct {
				doc   SearchDoc
				score float64
			}{doc, score})
		}
	}

	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].score > scoredResults[j].score
	})

	if len(scoredResults) > limit {
		scoredResults = scoredResults[:limit]
	}

	results := make([]SearchResult, len(scoredResults))
	for i, sr := range scoredResults {
		results[i] = SearchResult{
			Title:   sr.doc.Title,
			Path:    sr.doc.Location,
			Excerpt: truncateText(stripHTML(sr.doc.Text), 200),
		}
	}

	return results, nil
}

const (
	gitHubRawURL = "https://raw.githubusercontent.com/ContainerHive/ContainerHive/main/docs"
	gitHubWebURL = "https://github.com/ContainerHive/ContainerHive/blob/main/docs"
)

func getDocumentation(ctx context.Context, path string) (GetDocumentationOutput, error) {
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}

	if isValidDocPath(path) == false {
		return GetDocumentationOutput{}, fmt.Errorf("invalid path: path must not contain '..' or be absolute")
	}

	cacheKey := path
	if cached, ok := pageCache.Get(cacheKey); ok {
		return cached.(GetDocumentationOutput), nil
	}

	path = strings.TrimSuffix(path, ".html") + ".md"

	rawURL := gitHubRawURL + "/" + path
	webURL := gitHubWebURL + "/" + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return GetDocumentationOutput{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return GetDocumentationOutput{}, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GetDocumentationOutput{}, fmt.Errorf("page not found: %s", path)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetDocumentationOutput{}, fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)
	title := extractTitleFromMarkdown(content)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	result := GetDocumentationOutput{
		Title:   title,
		URL:     webURL,
		Content: content,
	}

	pageCache.Set(cacheKey, result, cacheTTL)

	return result, nil
}

func getSearchIndex(ctx context.Context) (*searchIndex, error) {
	if cached, ok := searchIndexCache.Get("search_index"); ok {
		return cached.(*searchIndex), nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchIndexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search index not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var index searchIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("failed to parse search index: %w", err)
	}

	searchIndexCache.Set("search_index", &index, cacheTTL)

	return &index, nil
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func stripHTML(html string) string {
	var result strings.Builder
	var inTag bool
	var prevChar rune

	for _, r := range html {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if inTag {
			continue
		}
		if r == '\n' && prevChar == '\n' {
			continue
		}
		result.WriteRune(r)
		prevChar = r
	}

	text := result.String()
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "  ", " ")
	return strings.TrimSpace(text)
}

func extractTitle(html string) string {
	start := strings.Index(html, "<title>")
	if start == -1 {
		start = strings.Index(html, "<Title>")
	}
	if start == -1 {
		return ""
	}
	start += 7

	end := strings.Index(html, "</title>")
	if end == -1 {
		end = strings.Index(html, "</Title>")
	}
	if end == -1 || end <= start {
		return ""
	}

	return strings.TrimSpace(html[start:end])
}

func extractBody(html string) string {
	bodyStart := strings.Index(html, "<body")
	if bodyStart == -1 {
		return html
	}

	tagEnd := strings.Index(html[bodyStart:], ">")
	if tagEnd == -1 {
		return html
	}
	contentStart := bodyStart + tagEnd + 1

	bodyEnd := strings.Index(html, "</body>")
	if bodyEnd == -1 {
		return html
	}

	return strings.TrimSpace(html[contentStart:bodyEnd])
}

func extractTitleFromMarkdown(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func isValidDocPath(path string) bool {
	if filepath.IsAbs(path) {
		return false
	}
	if strings.Contains(path, "..") {
		return false
	}
	return true
}
