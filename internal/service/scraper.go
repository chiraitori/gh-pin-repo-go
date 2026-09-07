package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/singleflight"

	"gh-pin-repo-go/internal/cache"
	"gh-pin-repo-go/internal/model"
)

var (
	ErrUserNotFound  = errors.New("github user or organization not found")
	ErrRateLimited   = errors.New("github rate limit reached")
	ErrFetchFailed   = errors.New("failed to fetch github profile")
	hexColorRegex    = regexp.MustCompile(`#([0-9a-fA-F]{3,6})`)
	countCleanRegex  = regexp.MustCompile(`[^0-9.kmKM]`)
)

type PinService interface {
	GetPinnedRepos(ctx context.Context, username string, forceRefresh bool) ([]model.PinnedRepo, bool, error)
}

type pinService struct {
	client      *http.Client
	cache       *cache.ShardedCache
	sfGroup     singleflight.Group
	cacheTTL    time.Duration
}

func NewPinService(c *cache.ShardedCache, cacheTTL time.Duration) PinService {
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     500,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	return &pinService{
		client:   client,
		cache:    c,
		cacheTTL: cacheTTL,
	}
}

// GetPinnedRepos fetches pinned repos with cache and singleflight protection
func (s *pinService) GetPinnedRepos(ctx context.Context, username string, forceRefresh bool) ([]model.PinnedRepo, bool, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	if username == "" {
		return nil, false, errors.New("username cannot be empty")
	}

	// 1. Check cache if not forcing refresh
	if !forceRefresh && s.cache != nil {
		if repos, found := s.cache.Get(username); found {
			return repos, true, nil
		}
	}

	// 2. Use singleflight to prevent duplicate concurrent scrapes for the same user
	result, err, _ := s.sfGroup.Do(username, func() (interface{}, error) {
		// Double check cache inside singleflight
		if !forceRefresh && s.cache != nil {
			if repos, found := s.cache.Get(username); found {
				return repos, nil
			}
		}

		repos, scrapeErr := s.scrapeProfile(ctx, username)
		if scrapeErr != nil {
			return nil, scrapeErr
		}

		// Store in cache
		if s.cache != nil {
			s.cache.Set(username, repos, s.cacheTTL)
		}

		return repos, nil
	})

	if err != nil {
		return nil, false, err
	}

	return result.([]model.PinnedRepo), false, nil
}

func (s *pinService) scrapeProfile(ctx context.Context, username string) ([]model.PinnedRepo, error) {
	profileURL := fmt.Sprintf("https://github.com/%s", username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUserNotFound
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrFetchFailed, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var repos []model.PinnedRepo

	doc.Find(".pinned-item-list-item").Each(func(i int, s *goquery.Selection) {
		repo := model.PinnedRepo{}

		// Find repo link and name
		linkEl := s.Find("a.text-bold, a[data-hovercard-type=\"repository\"], a[data-hydro-click*=\"PINNED_REPO\"]").First()
		href, _ := linkEl.Attr("href")
		href = strings.TrimSpace(href)

		repoName := strings.TrimSpace(s.Find("span.repo").First().Text())
		if repoName == "" && linkEl.Length() > 0 {
			repoName = strings.TrimSpace(linkEl.Text())
		}

		parts := strings.Split(strings.Trim(href, "/"), "/")
		owner := username
		if len(parts) >= 2 {
			owner = parts[0]
			if repoName == "" {
				repoName = parts[1]
			}
		}

		repo.Owner = owner
		repo.Repo = repoName
		if href != "" {
			if strings.HasPrefix(href, "http") {
				repo.Link = href
			} else {
				repo.Link = "https://github.com" + href
			}
		}

		// Description
		repo.Description = strings.TrimSpace(s.Find(".pinned-item-desc").First().Text())

		// Language
		repo.Language = strings.TrimSpace(s.Find("[itemprop=\"programmingLanguage\"]").First().Text())

		// Language color
		colorEl := s.Find(".repo-language-color").First()
		if colorStyle, exists := colorEl.Attr("style"); exists {
			matches := hexColorRegex.FindStringSubmatch(colorStyle)
			if len(matches) > 0 {
				repo.LanguageColor = matches[0]
			}
		}

		// Stars
		starEl := s.Find("a[href$=\"/stargazers\"]").First()
		if starEl.Length() > 0 {
			starsText := strings.TrimSpace(starEl.Text())
			repo.StarsDisplay = cleanNumberText(starsText)
			repo.Stars = parseCount(repo.StarsDisplay)
		}

		// Forks
		forkEl := s.Find("a[href$=\"/forks\"], a[href$=\"/network/members\"]").First()
		if forkEl.Length() > 0 {
			forksText := strings.TrimSpace(forkEl.Text())
			repo.ForksDisplay = cleanNumberText(forksText)
			repo.Forks = parseCount(repo.ForksDisplay)
		}

		// Is Fork
		metaText := s.Find(".pinned-item-meta").Text()
		if strings.Contains(strings.ToLower(metaText), "forked from") {
			repo.IsFork = true
		}

		if repo.Repo != "" {
			repos = append(repos, repo)
		}
	})

	if repos == nil {
		repos = []model.PinnedRepo{}
	}

	return repos, nil
}

func cleanNumberText(s string) string {
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !strings.EqualFold(trimmed, "stars") && !strings.EqualFold(trimmed, "forks") {
			return trimmed
		}
	}
	return strings.TrimSpace(s)
}

func parseCount(s string) int {
	s = countCleanRegex.ReplaceAllString(s, "")
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0
	}

	multiplier := 1.0
	if strings.HasSuffix(s, "k") {
		multiplier = 1000.0
		s = strings.TrimSuffix(s, "k")
	} else if strings.HasSuffix(s, "m") {
		multiplier = 1000000.0
		s = strings.TrimSuffix(s, "m")
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(val * multiplier)
}
