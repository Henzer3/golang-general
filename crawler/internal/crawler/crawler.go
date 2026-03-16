package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Crawler interface {
	ListenAndServe(ctx context.Context, address string) error
}

type CrawlRequest struct {
	URLs      []string `json:"urls"`
	Workers   int      `json:"workers"`
	TimeoutMS int      `json:"timeout_ms"`
}

type CrawlResponse struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

const ShutdownTime = 10 * time.Second

var _ Crawler = (*crawlerImpl)(nil)

type crawlerImpl struct {
	client *http.Client
	cache  *CacheTTL[CrawlResponse, string]
	sl     *singleflightImpl[CrawlResponse]
}

const (
	shutdownTimeout = 10 * time.Second
	cacheTTL        = time.Second
)

func New() *crawlerImpl {
	return &crawlerImpl{
		client: &http.Client{
			//nolint:mnd // http.Transport defaults tuning
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   runtime.GOMAXPROCS(-1),
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: time.Second * 3,
				ResponseHeaderTimeout: time.Minute,
			},
		},
		cache: NewCacheTTL[CrawlResponse, string](),
		sl:    &singleflightImpl[CrawlResponse]{},
	}
}

func (c *crawlerImpl) ListenAndServe(ctx context.Context, address string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/crawl", c.crawlHandler)

	srv := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	errorChan := make(chan error)
	defer close(errorChan)

	go func() {
		err := srv.ListenAndServe()
		errorChan <- err
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if e := srv.Shutdown(shutdownCtx); e != nil {
			return fmt.Errorf("Shutdown error: %w", e)
		}

		err := <-errorChan
		c.cache.CloseCacheTTL()
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil

	case err := <-errorChan:
		c.cache.CloseCacheTTL()
		return err
	}
}

func (c *crawlerImpl) crawlHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req CrawlRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	if ok, s := isCorrectData(req); !ok {
		http.Error(w, s, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.TimeoutMS)*time.Millisecond)
	defer cancel()

	input := Generate(ctx, req.URLs, normalizeURL, len(req.URLs))

	result := make([]CrawlResponse, len(req.URLs))

	Run(ctx, input, func(t task) {
		doTask(ctx, t, result, c)
	}, req.Workers)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func isCorrectData(req CrawlRequest) (bool, string) {
	if req.Workers <= 0 || req.Workers > 1000 {
		return false, fmt.Sprintf("Invalid number of workers:%d", req.Workers)
	}

	var errs []string
	for _, v := range req.URLs {
		if _, err := url.Parse(v); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return false, strings.Join(errs, "\n")
	}

	return true, ""
}

func doTask(ctx context.Context, t task, result []CrawlResponse, crawler *crawlerImpl) {
	v, err, _ := crawler.sl.Do(t.normolizedURL, func() (CrawlResponse, error) {
		if v, ok := crawler.cache.Get(t.normolizedURL); ok {
			return v, nil
		}

		resp, err := crawler.fetchURL(t.normolizedURL)
		if err == nil {
			crawler.cache.Set(t.normolizedURL, resp)
		}
		return resp, err
	})

	if err != nil {
		result[t.index] = CrawlResponse{
			URL:   t.url,
			Error: err.Error(),
		}
		return
	}

	if ctx.Err() == context.DeadlineExceeded {
		v.Error = "timeout exceeded"
		v.StatusCode = 0
	}
	v.URL = t.url
	result[t.index] = v
}

func (c *crawlerImpl) fetchURL(url string) (CrawlResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return CrawlResponse{
			URL:   url,
			Error: "create request error",
		}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return CrawlResponse{
			URL:   url,
			Error: "do request error",
		}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	return CrawlResponse{
		URL:        url,
		StatusCode: resp.StatusCode,
	}, nil
}

type task struct {
	index         int
	url           string
	normolizedURL string
}

func normalizeURL(i int, str string) task {
	u, _ := url.Parse(str)

	u.RawPath, _ = url.PathUnescape(u.RawPath)
	u.Path = path.Clean(u.Path)

	u.RawQuery = ""
	u.Fragment = ""

	return task{
		index:         i,
		url:           str,
		normolizedURL: u.String(),
	}
}

func withLock(mutex sync.Locker, action func()) {
	mutex.Lock()
	defer mutex.Unlock()

	action()
}
