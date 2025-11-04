package ogcrawl

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Meta struct {
	Title       string
	Description string
	Image       string
}

func Fetch(ctx context.Context, pageURL string) (*Meta, error) {
	client := &http.Client{Timeout: 4 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ShortlinkCrawler/1.1)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	base, _ := url.Parse(pageURL)
	n := &Meta{}
	firstImg := ""
	tok := html.NewTokenizer(resp.Body)
	for {
		t := tok.Next()
		if t == html.ErrorToken {
			break
		}
		switch t {
		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := tok.TagName()
			name := string(tn)
			if name == "meta" && hasAttr {
				attrs := map[string]string{}
				for {
					k, v, more := tok.TagAttr()
					attrs[string(k)] = string(v)
					if !more {
						break
					}
				}
				prop := attrs["property"]
				metaName := attrs["name"]
				content := attrs["content"]
				if prop == "og:title" {
					n.Title = content
				}
				if prop == "og:description" {
					n.Description = content
				}
				if prop == "og:image" || prop == "og:image:secure_url" || metaName == "twitter:image" {
					if content != "" {
						n.Image = resolveURL(base, content)
					}
				}
				if metaName == "description" && n.Description == "" {
					n.Description = content
				}
			}
			if name == "img" && hasAttr && firstImg == "" {
				src := ""
				for {
					k, v, more := tok.TagAttr()
					if string(k) == "src" {
						src = string(v)
					}
					if !more {
						break
					}
				}
				if src != "" {
					firstImg = resolveURL(base, src)
				}
			}
		}
	}
	if n.Image == "" && firstImg != "" {
		n.Image = firstImg
	}
	// basic sanitization
	n.Title = strings.TrimSpace(n.Title)
	n.Description = strings.TrimSpace(n.Description)
	return n, nil
}

func resolveURL(base *url.URL, ref string) string {
	if base == nil || ref == "" {
		return ref
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return base.ResolveReference(u).String()
}
