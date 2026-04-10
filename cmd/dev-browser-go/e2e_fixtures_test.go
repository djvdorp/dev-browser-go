package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func startE2ETestServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>dev-browser-go e2e</title>
  <style>
    html, body {
      margin: 0;
      min-height: 100%;
      background: #123456;
      color: #f7fafc;
      font-family: sans-serif;
    }
    main {
      min-height: 100vh;
      display: grid;
      place-items: center;
      background:
        linear-gradient(135deg, rgba(18, 52, 86, 1) 0%, rgba(0, 160, 160, 1) 100%);
    }
    h1 {
      font-size: 48px;
      margin: 0;
    }
  </style>
</head>
<body>
  <main>
    <h1>daemon persistence check</h1>
  </main>
</body>
</html>`))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startDelayedBodyServer(t *testing.T, delay time.Duration) string {
	t.Helper()
	delayMS := int(delay / time.Millisecond)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>dev-browser-go delayed body</title>
  <script>
    document.addEventListener('DOMContentLoaded', () => {
      setTimeout(() => {
        const main = document.createElement('main');
        main.id = 'marker';
        main.textContent = 'delayed body marker';
        document.body.appendChild(main);
      }, %d);
    });
  </script>
</head>
<body></body>
</html>`, delayMS)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startTitledServer(t *testing.T, title, body string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!doctype html>
<html>
<head><meta charset="utf-8"><title>%s</title></head>
<body><main>%s</main></body>
</html>`, title, body)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startHTMLValidateFixtureServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>html validate fixture</title></head>
<body>
  <div id="dup"></div>
  <span id="dup"></span>
  <img src="missing-alt.png">
  <input type="text">
</body>
</html>`))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startNetworkFixtureServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>network fixture</title></head>
<body>
  <main>network fixture</main>
</body>
</html>`))
		case "/burst":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>network burst</title></head>
<body>
  <main>network burst</main>
  <script>
    document.addEventListener('DOMContentLoaded', async () => {
      await fetch('/api/ok?ts=' + Date.now());
      await fetch('/api/fail?ts=' + Date.now());
    });
  </script>
</body>
</html>`))
		case "/api/ok":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/fail":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"ok":false}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startAssetSnapshotFixtureServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>asset snapshot fixture</title></head>
<body>
  <main id="asset-snapshot-marker">asset-snapshot-marker</main>
  <img src="/img.png" alt="fixture">
  <script src="/app.js"></script>
</body>
</html>`))
		case "/img.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(minimalPNG())
		case "/app.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(`window.assetSnapshotLoaded = true;`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startInteractiveRefsFixtureServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>interactive refs fixture</title>
  <style>
    body { font-family: sans-serif; margin: 24px; }
    form { display: grid; gap: 12px; max-width: 360px; }
    input, button { font-size: 16px; padding: 10px 12px; }
    #result { margin-top: 16px; min-height: 24px; }
  </style>
</head>
<body>
  <h1>Selector and ref fixture</h1>
  <form id="search-form">
    <label for="search">Search query</label>
    <input id="search" type="text" aria-label="Search query" placeholder="Search query">
    <button id="run-search" type="button">Run search</button>
  </form>
  <p id="result">idle</p>
  <script>
    (() => {
      const input = document.getElementById('search');
      const button = document.getElementById('run-search');
      const result = document.getElementById('result');
      const update = (mode) => {
        const text = input.value.trim() || 'empty';
        setTimeout(() => {
          result.textContent = text + ' via ' + mode;
        }, 120);
      };
      button.addEventListener('click', () => update('click'));
      input.addEventListener('keydown', (event) => {
        if (event.key === 'Enter') {
          event.preventDefault();
          update('enter');
        }
      });
    })();
  </script>
</body>
</html>`))
	}))
	t.Cleanup(server.Close)
	return server.URL
}
