package proxy


import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const content = "<html><body>BODY</body>"

func TestProxy_Cache(t *testing.T) {
	p := NewProxy("localhost", 86400, 1000 * time.Millisecond)

	rand.Seed(time.Now().Unix())
	pid := int64(RandInRange(1, 1000000) * -1)

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
	pid := int64(RandInRange(1, 1000000) * -1)
	pid2 := int64(RandInRange(1000000, 2000000) * -1)

	testRouter := mux.NewRouter()
	testRouter.PathPrefix("/page").Queries("pageId", "{pageId}").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := mux.Vars(r)["pageId"]
			pageId, _ := strconv.Atoi(p)

			if int64(pageId) == pid {
				fmt.Printf("test server was invoked: %v\n", r.Method)
				_, _ = w.Write([]byte(content))
			} else {
				http.Error(w, "Page not found", http.StatusNotFound)
			}
		})

	srv := &http.Server{
		Addr:    ":39012",
		Handler: testRouter,
	}

	go func() {
		_ = srv.ListenAndServe()
	}()

	time.Sleep(7 * time.Second)
	p := NewProxy("localhost", 86400, 1000 * time.Millisecond)

	html, err := p.Resolve(pid, fmt.Sprintf("http://localhost:39012/page?pageId=%v", pid))
	require.NoError(t, err)
	require.Equal(t, content, html)

	// now check if page exists in cache
	require.True(t, p.HasInCache(pid))

	html2, err := p.ReadFromCache(pid)
	require.NoError(t, err)
	require.Equal(t, content, html2)

	html, err = p.Resolve(pid2, fmt.Sprintf("http://localhost:39012/page?pageId=%v", pid2))
	require.Error(t, err)

	_ = srv.Shutdown(context.TODO())
}

