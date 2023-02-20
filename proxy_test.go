package proxy

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const content = "<html><body>BODY</body>"

func TestProxy_Cache(t *testing.T) {
	p := NewProxy("localhost", 86400, 1000*time.Millisecond)

	rand.Seed(time.Now().Unix())
	pid := uuid.NewString()

	require.False(t, p.HasInCache(pid), "Has %v", pid)

	// now store some content
	require.NoError(t, p.Cache(content, pid))
	require.True(t, p.HasInCache(pid))

	// and check it immediately
	html, err := p.ReadFromCache(pid)
	require.NoError(t, err)
	require.Equal(t, content, html)

	// forget it, and check again
	p.Forget(pid)
	require.False(t, p.HasInCache(pid))

}

func TestProxy_Resolve(t *testing.T) {
	rand.Seed(time.Now().Unix())
	pid := uuid.NewString()
	pid2 := uuid.NewString()

	testRouter := mux.NewRouter()
	testRouter.PathPrefix("/page").Queries("pageId", "{pageId}").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pageId := mux.Vars(r)["pageId"]

			if pageId == pid {
				fmt.Printf("test server was invoked: %v\n", r.Method)
				_, _ = w.Write([]byte(content))
			} else {
				http.Error(w, "Page not found", http.StatusNotFound)
			}
		})

	srv := httptest.NewServer(testRouter)
	defer srv.Close()

	p := NewProxy("localhost", 86400, 1000*time.Millisecond)

	html, err := p.Resolve(context.Background(), fmt.Sprintf("%v/page?pageId=%v", srv.URL, pid), pid)
	require.NoError(t, err)
	require.Equal(t, content, html)

	// now check if page exists in cache
	require.True(t, p.HasInCache(pid))

	html2, err := p.ReadFromCache(pid)
	require.NoError(t, err)
	require.Equal(t, content, html2)

	html, err = p.Resolve(context.Background(), pid2, fmt.Sprintf("http://localhost:39012/page?pageId=%v", pid2))
	require.Error(t, err)
}
