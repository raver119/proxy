package proxy

import (
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

func pageKey(pageId int64) string {
	return fmt.Sprintf("ProxyPageCache_%v", pageId)
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
//	This method fetches HTML from the remote URL. Or from shared cache.
func (p *Proxy) Resolve(pageId int64, url string) (html string, err error) {
	if html, err = p.ReadFromCache(pageId); err == nil {
		// serve from the local cache
		return
	} else {
		// TODO: add performance logging here
		get, err := p.c.R().Get(url)
		if err != nil {
			// TODO: timeout errors are very bad, and must be tracked here
			if IsVerbose() {
				log.Printf("Resolve req <%v> failed with message <%v>", url, err.Error())
			}
			return "", err
		}

		if get.IsSuccess() {
			html = get.String()
			_ = p.Cache(html, pageId)
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
//	This method removes given pageId from the cache
func (p *Proxy) Forget(pageId int64) {
	_ = p.m.Delete(pageKey(pageId))
}

// HasInCache
//	This method checks if a given pageId is available in cache
func (p *Proxy) HasInCache(pageId int64) bool {
	item, err := p.m.Get(pageKey(pageId))
	return err == nil && item != nil
}

// Cache
//	This method injects given html into cache
func (p *Proxy) Cache(html string, pageId int64) (err error) {
	err = p.m.Set(&memcache.Item{
		Key:        pageKey(pageId),
		Value:      []byte(html),
		Expiration: p.e, // two weeks by default
	})

	return
}

// ReadFromCache
//	This method returns previously cached page
func (p *Proxy) ReadFromCache(pageId int64) (html string, err error) {
	item, err := p.m.Get(pageKey(pageId))
	if err != nil {
		return "", err
	}

	html = string(item.Value)
	return
}

