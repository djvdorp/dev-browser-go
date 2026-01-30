package devbrowser

import (
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type PerfMetricsOptions struct {
	SampleMs int
	TopN     int
}

func GetPerfMetrics(page playwright.Page, opts PerfMetricsOptions) (map[string]interface{}, error) {
	if opts.SampleMs <= 0 {
		opts.SampleMs = 1200
	}
	if opts.TopN <= 0 {
		opts.TopN = 20
	}

	js := `async (opts) => {
  const sampleMs = Math.max(200, Number(opts.sampleMs || 1200));
  const topN = Math.max(0, Number(opts.topN || 20));

  function navTiming() {
    const nav = performance.getEntriesByType('navigation');
    return nav && nav.length ? nav[0].toJSON ? nav[0].toJSON() : nav[0] : null;
  }

  function paint() {
    const paints = performance.getEntriesByType('paint') || [];
    const out = {};
    for (const p of paints) out[p.name] = p.startTime;
    return out;
  }

  // Best-effort CWV using PerformanceObserver.
  let cls = 0;
  let lcp = null;

  const observers = [];
  function tryObs(type, handler, buffered=true) {
    try {
      const po = new PerformanceObserver((list) => handler(list.getEntries()));
      po.observe({ type, buffered });
      observers.push(po);
    } catch { }
  }

  tryObs('layout-shift', (entries) => {
    for (const e of entries) {
      if (e && !e.hadRecentInput) cls += e.value || 0;
    }
  });

  tryObs('largest-contentful-paint', (entries) => {
    for (const e of entries) {
      if (!e) continue;
      const v = e.startTime || 0;
      if (lcp === null || v > lcp) lcp = v;
    }
  });

  // FPS sample
  let frames = 0;
  let rafStart = performance.now();
  await new Promise((resolve) => {
    function tick(now) {
      frames++;
      if (now - rafStart >= sampleMs) return resolve();
      requestAnimationFrame(tick);
    }
    requestAnimationFrame(tick);
  });
  const rafEnd = performance.now();
  const fps = frames / ((rafEnd - rafStart) / 1000);

  for (const o of observers) { try { o.disconnect(); } catch {} }

  // Resource timing summary
  const res = performance.getEntriesByType('resource') || [];
  const byType = {};
  const top = [];
  for (const r of res) {
    const t = String(r.initiatorType || 'other');
    const dur = Number(r.duration || 0);
    byType[t] = byType[t] || { count: 0, totalDuration: 0 };
    byType[t].count++;
    byType[t].totalDuration += dur;

    if (topN > 0) top.push({ name: r.name, initiatorType: t, duration: dur, transferSize: r.transferSize || 0 });
  }
  if (topN > 0) top.sort((a,b) => b.duration - a.duration);

  return {
    url: location.href,
    timing: {
      navigation: navTiming(),
      paint: paint(),
    },
    cwv: { cls, lcp },
    fps: { sampleMs, frames, fps },
    resources: {
      total: res.length,
      byType,
      top: topN > 0 ? top.slice(0, topN) : []
    }
  };
}`

	res, err := page.Evaluate(js, map[string]interface{}{"sampleMs": opts.SampleMs, "topN": opts.TopN})
	if err != nil {
		return nil, err
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected perf metrics result")
	}
	return m, nil
}

func normalizePerfFields(fields []string) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		v := strings.TrimSpace(f)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}
