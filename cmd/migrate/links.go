package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/rokob/ryuguma/internal/content"
)

// shortener hosts redirect to a full Amazon URL.
var (
	shortenerHosts = []string{"a.co", "amzn.to"}
	amazonHosts    = []string{"amazon.com", "smile.amazon.com", "amzn.com", "a.co", "amzn.to"}
	urlPattern     = regexp.MustCompile(`https?://[^\s)<>"\]]+`)
	asinPattern    = regexp.MustCompile(`/(?:dp|gp/product|ASIN)/([A-Z0-9]{10})(?:[/?]|$)`)
	bareASIN       = regexp.MustCompile(`^/([A-Z0-9]{10})/?$`)
)

type resolver interface {
	// Resolve follows a short link to the full URL it points at.
	Resolve(ctx context.Context, short string) (string, error)
}

func hostOf(u *url.URL) string {
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

// findStoreLink returns the first Amazon product link in a post body: a
// short link, or a URL naming a product ID.
func findStoreLink(body string) string {
	for _, raw := range urlPattern.FindAllString(body, -1) {
		u, err := url.Parse(raw)
		if err != nil || !slices.Contains(amazonHosts, hostOf(u)) {
			continue
		}
		if slices.Contains(shortenerHosts, hostOf(u)) || asinPattern.MatchString(u.Path) ||
			(hostOf(u) == "amzn.com" && bareASIN.MatchString(u.Path)) {
			return raw
		}
	}
	return ""
}

// normalizeStoreLink turns the many Amazon link styles into
// https://www.amazon.com/dp/<ID>, dropping tracking and affiliate tags.
func normalizeStoreLink(ctx context.Context, r resolver, raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if slices.Contains(shortenerHosts, hostOf(u)) {
		long, err := r.Resolve(ctx, raw)
		if err != nil {
			return "", err
		}
		if u, err = url.Parse(long); err != nil {
			return "", err
		}
	}
	if m := asinPattern.FindStringSubmatch(u.Path); m != nil {
		return "https://www.amazon.com/dp/" + m[1], nil
	}
	if m := bareASIN.FindStringSubmatch(u.Path); m != nil && hostOf(u) == "amzn.com" {
		return "https://www.amazon.com/dp/" + m[1], nil
	}
	return "https://www.amazon.com" + u.Path, nil
}

// backfillLinks sets link: on books that lack one, from the first Amazon
// link in their text. It returns changed files and notes for anything that
// needs a look.
func backfillLinks(ctx context.Context, r resolver, files map[string][]byte) (map[string][]byte, []string, error) {
	out := map[string][]byte{}
	var report []string
	var errs []error
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		e, err := content.Parse(p, files[p])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if e.Book == nil || e.Book.Link != "" {
			continue
		}
		raw := findStoreLink(e.Body)
		if raw == "" {
			report = append(report, fmt.Sprintf("%s: no store link in the text", e.Slug))
			continue
		}
		link, err := normalizeStoreLink(ctx, r, raw)
		if err != nil {
			link = strings.Replace(raw, "http://", "https://", 1)
			report = append(report, fmt.Sprintf("%s: kept %s (could not resolve: %v)", e.Slug, link, err))
		}
		e.Book.Link = link
		data, err := content.Encode(e)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out[p] = data
	}
	return out, report, errors.Join(errs...)
}

// httpResolver follows redirects one hop at a time until it reaches an
// Amazon product URL, without fetching Amazon pages themselves.
type httpResolver struct {
	client *http.Client
}

func (h httpResolver) Resolve(ctx context.Context, short string) (string, error) {
	next := short
	for range 5 {
		// a.co answers HEAD with 404, so use GET; redirects are not followed.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, next, nil)
		if err != nil {
			return "", err
		}
		resp, err := h.client.Do(req)
		if err != nil {
			return "", err
		}
		resp.Body.Close()
		loc, err := resp.Location()
		if err != nil {
			return "", fmt.Errorf("%s: %s with no redirect", next, resp.Status)
		}
		if host := hostOf(loc); host == "amazon.com" || host == "smile.amazon.com" {
			return loc.String(), nil
		}
		next = loc.String()
	}
	return "", fmt.Errorf("%s: too many redirects", short)
}
