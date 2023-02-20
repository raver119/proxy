package proxy

import (
	"context"
	"fmt"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/go-resty/resty/v2"
	"log"
	"strings"
	"time"
)

type Proxy struct {
	e int32
	c *resty.Client
	m *memcache.Client
}

func pageKey(keys ...string) string {
	return fmt.Sprintf("ProxyPageCache_%v", keys)
}

func NewProxy(mc string, expiration int32, timeOut time.Duration) Proxy {
	// append default port, if it wasn't specified
	if !strings.Contains(mc, ":") {
		mc = fmt.Sprintf("%v:11211", mc)
	}

	m := memcache.New(mc)
	c := resty.New()
	c.SetTimeout(timeOut)
	return Proxy{
		e: expiration,
		c: c,
		m: m,
	}
}

// Resolve
//
//	This method fetches HTML from the remote URL. Or from shared cache.
func (p *Proxy) Resolve(ctx context.Context, url string, keys ...string) (html string, err error) {
	if html, err = p.ReadFromCache(keys...); err == nil {
		// serve from the local cache
		return
	} else {
		// TODO: add performance logging here
		get, err := p.c.R().SetContext(ctx).Get(url)
		if err != nil {
			// TODO: timeout errors are very bad, and must be tracked here
			if IsVerbose() {
				log.Printf("Resolve req <%v> failed with message <%v>", url, err.Error())
			}
			return "", err
		}

		if get.IsSuccess() {
			html = get.String()
			_ = p.Cache(html, keys...)
			return html, nil
		} else {
			if IsVerbose() {
				log.Printf("Resolve req <%v> failed with code <%v> message <%v>", url, get.StatusCode(), get.String())
			}

			return "", fmt.Errorf("node endpoint returned bad code: %v - %v", get.StatusCode(), get.String())
		}
	}
}

// Forget
//
//	This method removes given pageId from the cache
func (p *Proxy) Forget(keys ...string) {
	_ = p.m.Delete(pageKey(keys...))
}

// HasInCache
//
//	This method checks if a given pageId is available in cache
func (p *Proxy) HasInCache(keys ...string) bool {
	item, err := p.m.Get(pageKey(keys...))
	return err == nil && item != nil
}

// Cache
//
//	This method injects given html into cache
func (p *Proxy) Cache(html string, keys ...string) (err error) {
	err = p.m.Set(&memcache.Item{
		Key:        pageKey(keys...),
		Value:      []byte(html),
		Expiration: p.e, // two weeks by default
	})

	return
}

// ReadFromCache
//
//	This method returns previously cached page
func (p *Proxy) ReadFromCache(keys ...string) (html string, err error) {
	item, err := p.m.Get(pageKey(keys...))
	if err != nil {
		return "", err
	}

	html = string(item.Value)
	return
}
