package service

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseCount(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"0", 0},
		{"10", 10},
		{"999", 999},
		{"1.2k", 1200},
		{"2.5K", 2500},
		{"45.7k", 45700},
		{"1M", 1000000},
		{"1.5m", 1500000},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := parseCount(tt.input)
		if got != tt.expected {
			t.Errorf("parseCount(%q) = %d, expected %d", tt.input, got, tt.expected)
		}
	}
}

func TestCleanNumberText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\n  stars\n  4.5k\n", "4.5k"},
		{"\n  forks\n  322\n", "322"},
		{"100", "100"},
	}

	for _, tt := range tests {
		got := cleanNumberText(tt.input)
		if got != tt.expected {
			t.Errorf("cleanNumberText(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestScrapeHtmlParsing(t *testing.T) {
	htmlSnippet := `
	<html>
	<body>
		<div class="pinned-item-list-item">
			<a class="text-bold" href="/octocat/Hello-World"><span class="repo">Hello-World</span></a>
			<p class="pinned-item-desc">My first repository on GitHub!</p>
			<span itemprop="programmingLanguage">Ruby</span>
			<span class="repo-language-color" style="background-color: #701516;"></span>
			<a href="/octocat/Hello-World/stargazers">1.5k</a>
			<a href="/octocat/Hello-World/forks">300</a>
		</div>
	</body>
	</html>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlSnippet))
	if err != nil {
		t.Fatalf("failed to parse test HTML: %v", err)
	}

	items := doc.Find(".pinned-item-list-item")
	if items.Length() != 1 {
		t.Fatalf("expected 1 item, got %d", items.Length())
	}

	linkEl := items.Find("a.text-bold").First()
	href, _ := linkEl.Attr("href")
	if href != "/octocat/Hello-World" {
		t.Errorf("expected href '/octocat/Hello-World', got %q", href)
	}

	desc := strings.TrimSpace(items.Find(".pinned-item-desc").Text())
	if desc != "My first repository on GitHub!" {
		t.Errorf("expected desc 'My first repository on GitHub!', got %q", desc)
	}

	lang := strings.TrimSpace(items.Find("[itemprop=\"programmingLanguage\"]").Text())
	if lang != "Ruby" {
		t.Errorf("expected lang 'Ruby', got %q", lang)
	}

	starsText := strings.TrimSpace(items.Find("a[href$=\"/stargazers\"]").Text())
	stars := parseCount(starsText)
	if stars != 1500 {
		t.Errorf("expected 1500 stars, got %d", stars)
	}
}
