// Eko: A terminal based social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package webserver

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kyren223/eko/embeds"
	"github.com/kyren223/eko/pkg/assert"
)

func ServePrometheusMetrics() {
	slog.Info("starting metrics webserver...")

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         ":2112",
		Handler:      metricsMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	err := srv.ListenAndServe()
	if err != nil {
		slog.Error("metrics webserver error", "error", err)
	} else {
		slog.Info("metrics webserver terminated")
	}
}

func ServeEkoWebsite() {
	slog.Info("starting public webserver...")

	publicMux := http.NewServeMux()

	publicMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://github.com/kyren223/eko", http.StatusFound)
	})

	publicMux.HandleFunc("/install.sh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-sh")
		_, err := io.WriteString(w, embeds.Installer)
		assert.NoError(err, "installer should be valid")
	})

	publicMux.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, err := w.Write([]byte(css))
		assert.NoError(err, "css *should* be valid")
	})

	publicMux.HandleFunc("/terms-of-service", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		tos := embeds.TermsOfService.Load().(string)
		tos = strings.ReplaceAll(tos, "Privacy Policy", "[Privacy Policy](../privacy-policy)")
		html := mdToHTML(tos)
		writeLegalLayoutHtml(w, html)
	})

	publicMux.HandleFunc("/privacy-policy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		privacy := embeds.PrivacyPolicy.Load().(string)
		privacy = strings.ReplaceAll(privacy, "Terms of Service", "[Terms of Service](../terms-of-service)")
		html := mdToHTML(privacy)
		writeLegalLayoutHtml(w, html)
	})

	srv := &http.Server{
		Addr:         ":7443",
		Handler:      publicMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	err := srv.ListenAndServe()
	if err != nil {
		slog.Error("public webserver error", "error", err)
	} else {
		slog.Info("public webserver terminated")
	}
}

func writeLegalLayoutHtml(w io.Writer, html string) {
	_, err := fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
		  <meta charset="utf-8">
		  <title>Terms of Service</title>
		  <link rel="stylesheet" href="/style.css">
		</head>
		<body class="prose">
			<main>
		  <div class="sl-markdown-content">
			%s
		  </div>
			</main>
		</body>
		</html>
		`, html)
	assert.NoError(err, "html *should* be valid")
}

func mdToHTML(md string) string {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(md))

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return string(markdown.Render(doc, renderer))
}

// Yes this is horrible, but it works and I don't like web dev :)
// Feel free to improve it and open a PR
// Probably most of these are unncessary, I just copy pasted it from my website css
var css = `
.sl-markdown-content
  :not(.expressive-code *)
  + :not(a, strong, em, del, span, input, code, br)
  + :not(a, strong, em, del, span, input, code, br, :where(.not-content *)) {
  margin-top: 1rem;
}

/* Headings after non-headings have more spacing. */
.sl-markdown-content
  :not(h1, h2, h3, h4, h5, h6)
  + :is(h1, h2, h3, h4, h5, h6):not(:where(.not-content *)) {
  margin-top: 1.5em;
}

.sl-markdown-content li + li:not(:where(.not-content *)),
.sl-markdown-content dt + dt:not(:where(.not-content *)),
.sl-markdown-content dt + dd:not(:where(.not-content *)),
.sl-markdown-content dd + dd:not(:where(.not-content *)) {
  margin-top: 0.25rem;
}

.sl-markdown-content li:not(:where(.not-content *)) {
  overflow-wrap: anywhere;
}

.sl-markdown-content
  li
  > :last-child:not(li, ul, ol):not(
    a,
    strong,
    em,
    del,
    span,
    input,
    :where(.not-content *)
  ) {
  margin-bottom: 1.25rem;
}

.sl-markdown-content dt:not(:where(.not-content *)) {
  font-weight: 700;
}
.sl-markdown-content dd:not(:where(.not-content *)) {
  padding-inline-start: 1rem;
}

.sl-markdown-content :is(h1, h2, h3, h4, h5, h6):not(:where(.not-content *)) {
  color: var(--sl-color-white);
  line-height: var(--sl-line-height-headings);
  font-weight: 600;
  font-size: 10000%;
}

.sl-markdown-content
  :is(img, picture, video, canvas, svg, iframe):not(:where(.not-content *)) {
  display: block;
  max-width: 100%;
  height: auto;
}

.sl-markdown-content h1:not(:where(.not-content *)) {
  font-size: var(--sl-text-h1);
}
.sl-markdown-content h2:not(:where(.not-content *)) {
  font-size: var(--sl-text-h2);
}
.sl-markdown-content h3:not(:where(.not-content *)) {
  font-size: var(--sl-text-h3);
}
.sl-markdown-content h4:not(:where(.not-content *)) {
  font-size: var(--sl-text-h4);
}
.sl-markdown-content h5:not(:where(.not-content *)) {
  font-size: var(--sl-text-h5);
}
.sl-markdown-content h6:not(:where(.not-content *)) {
  font-size: var(--sl-text-h6);
}

.sl-markdown-content a:not(:where(.not-content *)) {
  color: var(--sl-color-text-accent);
}
.sl-markdown-content a:hover:not(:where(.not-content *)) {
  color: var(--sl-color-white);
}

.sl-markdown-content code:not(:where(.not-content *)) {
  background-color: var(--sl-color-bg-inline-code);
  margin-block: -0.125rem;
  padding: 0.125rem 0.375rem;
  font-size: var(--sl-text-code-sm);
}
.sl-markdown-content :is(h1, h2, h3, h4, h5, h6) code {
  font-size: inherit;
}

.sl-markdown-content pre:not(:where(.not-content *)) {
  border: 1px solid var(--sl-color-gray-5);
  padding: 0.75rem 1rem;
  font-size: var(--sl-text-code);
  tab-size: 2;
}

.sl-markdown-content pre code:not(:where(.not-content *)) {
  all: unset;
  font-family: var(--__sl-font-mono);
}

.sl-markdown-content blockquote:not(:where(.not-content *)) {
  border-inline-start: 1px solid var(--sl-color-gray-5);
  padding-inline-start: 1rem;
}

/* Table styling */
.sl-markdown-content table:not(:where(.not-content *)) {
  display: block;
  overflow: auto;
  border-spacing: 0;
}
.sl-markdown-content :is(th, td):not(:where(.not-content *)) {
  border-bottom: 1px solid var(--sl-color-gray-5);
  padding: 0.5rem 1rem;
  /* Align text to the top of the row in multiline tables. */
  vertical-align: baseline;
}
.sl-markdown-content
  :is(th:first-child, td:first-child):not(:where(.not-content *)) {
  padding-inline-start: 0;
}
.sl-markdown-content
  :is(th:last-child, td:last-child):not(:where(.not-content *)) {
  padding-inline-end: 0;
}
.sl-markdown-content th:not(:where(.not-content *)) {
  color: var(--sl-color-white);
  font-weight: 600;
}
/* Align headings to the start of the line unless set by the align attribute. */
.sl-markdown-content th:not([align]):not(:where(.not-content *)) {
  text-align: start;
}
/* <table>s, <hr>s, and <blockquote>s inside asides */
.sl-markdown-content
  .starlight-aside
  :is(th, td, hr, blockquote):not(:where(.not-content *)) {
  border-color: var(--sl-color-gray-4);
}
@supports (
  border-color:
    color-mix(in srgb, var(--sl-color-asides-text-accent) 30%, transparent)
) {
  .sl-markdown-content
    .starlight-aside
    :is(th, td, hr, blockquote):not(:where(.not-content *)) {
    border-color: color-mix(
      in srgb,
      var(--sl-color-asides-text-accent) 30%,
      transparent
    );
  }
}

/* <code> inside asides */
@supports (
  border-color:
    color-mix(in srgb, var(--sl-color-asides-text-accent) 12%, transparent)
) {
  .sl-markdown-content .starlight-aside code:not(:where(.not-content *)) {
    background-color: color-mix(
      in srgb,
      var(--sl-color-asides-text-accent) 12%,
      transparent
    );
  }
}

.sl-markdown-content hr:not(:where(.not-content *)) {
  border: 0;
  border-bottom: 1px solid var(--sl-color-hairline);
}

/* <details> and <summary> styles */
.sl-markdown-content details:not(:where(.not-content *)) {
  --sl-details-border-color: var(--sl-color-gray-5);
  --sl-details-border-color--hover: var(--sl-color-text-accent);

  border-inline-start: 2px solid var(--sl-details-border-color);
  padding-inline-start: 1rem;
}
.sl-markdown-content details:not([open]):hover:not(:where(.not-content *)),
.sl-markdown-content details:has(> summary:hover):not(:where(.not-content *)) {
  border-color: var(--sl-details-border-color--hover);
}
.sl-markdown-content summary:not(:where(.not-content *)) {
  color: var(--sl-color-white);
  cursor: pointer;
  display: block; /* Needed to hide the default marker in some browsers. */
  font-weight: 600;
  /* Expand the outline so that the marker cannot distort it. */
  margin-inline-start: -0.5rem;
  padding-inline-start: 0.5rem;
}
.sl-markdown-content details[open] > summary:not(:where(.not-content *)) {
  margin-bottom: 1rem;
}

/* <summary> marker styles */
.sl-markdown-content summary:not(:where(.not-content *))::marker,
.sl-markdown-content
  summary:not(:where(.not-content *))::-webkit-details-marker {
  display: none;
}
.sl-markdown-content summary:not(:where(.not-content *))::before {
  --sl-details-marker-size: 1.25rem;

  background-color: currentColor;
  content: "";
  display: inline-block;
  height: var(--sl-details-marker-size);
  width: var(--sl-details-marker-size);
  margin-inline: calc((var(--sl-details-marker-size) / 4) * -1) 0.25rem;
  vertical-align: middle;
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath d='M14.8 11.3 10.6 7a1 1 0 1 0-1.4 1.5l3.5 3.5-3.5 3.5a1 1 0 0 0 0 1.4 1 1 0 0 0 .7.3 1 1 0 0 0 .7-.3l4.2-4.2a1 1 0 0 0 0-1.4Z'/%3E%3C/svg%3E%0A");
  mask-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath d='M14.8 11.3 10.6 7a1 1 0 1 0-1.4 1.5l3.5 3.5-3.5 3.5a1 1 0 0 0 0 1.4 1 1 0 0 0 .7.3 1 1 0 0 0 .7-.3l4.2-4.2a1 1 0 0 0 0-1.4Z'/%3E%3C/svg%3E%0A");
  -webkit-mask-repeat: no-repeat;
  mask-repeat: no-repeat;
}
@media (prefers-reduced-motion: no-preference) {
  .sl-markdown-content summary:not(:where(.not-content *))::before {
    transition: transform 0.2s ease-in-out;
  }
}
.sl-markdown-content
  details[open]
  > summary:not(:where(.not-content *))::before {
  transform: rotateZ(90deg);
}
[dir="rtl"] .sl-markdown-content summary:not(:where(.not-content *))::before,
.sl-markdown-content [dir="rtl"] summary:not(:where(.not-content *))::before {
  transform: rotateZ(180deg);
}
/* <summary> with only a paragraph automatically added when using MDX */
.sl-markdown-content summary:not(:where(.not-content *)) p:only-child {
  display: inline;
}

/* <details> styles inside asides */
.sl-markdown-content .starlight-aside details:not(:where(.not-content *)) {
  --sl-details-border-color: var(--sl-color-asides-border);
  --sl-details-border-color--hover: var(--sl-color-asides-text-accent);
}


.starlight-aside {
  padding: 1rem;
  border-inline-start: 0.25rem solid var(--sl-color-asides-border);
  color: var(--sl-color-white);
}
.starlight-aside--note {
  --sl-color-asides-text-accent: var(--sl-color-blue-high);
  --sl-color-asides-border: var(--sl-color-blue);
  background-color: var(--sl-color-blue-low);
}
.starlight-aside--tip {
  --sl-color-asides-text-accent: var(--sl-color-purple-high);
  --sl-color-asides-border: var(--sl-color-purple);
  background-color: var(--sl-color-purple-low);
}
.starlight-aside--caution {
  --sl-color-asides-text-accent: var(--sl-color-orange-high);
  --sl-color-asides-border: var(--sl-color-orange);
  background-color: var(--sl-color-orange-low);
}
.starlight-aside--danger {
  --sl-color-asides-text-accent: var(--sl-color-red-high);
  --sl-color-asides-border: var(--sl-color-red);
  background-color: var(--sl-color-red-low);
}

.starlight-aside__title {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  font-size: var(--sl-text-h5);
  font-weight: 600;
  line-height: var(--sl-line-height-headings);
  color: var(--sl-color-asides-text-accent);
}

.starlight-aside__icon {
  font-size: 1.333em;
  width: 1em;
  height: 1em;
}

.starlight-aside__title + .starlight-aside__content {
  margin-top: 0.5rem;
}

.starlight-aside__content a {
  color: var(--sl-color-asides-text-accent);
}

:root,
::backdrop {
  /* Colors (dark mode) */
  --sl-color-white: hsl(0, 0%, 100%); /* “white” */
  --sl-color-gray-1: hsl(224, 20%, 94%);
  --sl-color-gray-2: hsl(224, 6%, 77%);
  --sl-color-gray-3: hsl(224, 6%, 56%);
  --sl-color-gray-4: hsl(224, 7%, 36%);
  --sl-color-gray-5: hsl(224, 10%, 23%);
  --sl-color-gray-6: hsl(224, 14%, 16%);
  --sl-color-black: hsl(224, 10%, 10%);

  --sl-hue-orange: 41;
  --sl-color-orange-low: hsl(var(--sl-hue-orange), 39%, 22%);
  --sl-color-orange: hsl(var(--sl-hue-orange), 82%, 63%);
  --sl-color-orange-high: hsl(var(--sl-hue-orange), 82%, 87%);
  --sl-hue-green: 101;
  --sl-color-green-low: hsl(var(--sl-hue-green), 39%, 22%);
  --sl-color-green: hsl(var(--sl-hue-green), 82%, 63%);
  --sl-color-green-high: hsl(var(--sl-hue-green), 82%, 80%);
  --sl-hue-blue: 234;
  --sl-color-blue-low: hsl(var(--sl-hue-blue), 54%, 20%);
  --sl-color-blue: hsl(var(--sl-hue-blue), 100%, 60%);
  --sl-color-blue-high: hsl(var(--sl-hue-blue), 100%, 87%);
  --sl-hue-purple: 281;
  --sl-color-purple-low: hsl(var(--sl-hue-purple), 39%, 22%);
  --sl-color-purple: hsl(var(--sl-hue-purple), 82%, 63%);
  --sl-color-purple-high: hsl(var(--sl-hue-purple), 82%, 89%);
  --sl-hue-red: 339;
  --sl-color-red-low: hsl(var(--sl-hue-red), 39%, 22%);
  --sl-color-red: hsl(var(--sl-hue-red), 82%, 63%);
  --sl-color-red-high: hsl(var(--sl-hue-red), 82%, 87%);

  --sl-color-accent-low: hsl(224, 54%, 20%);
  --sl-color-accent: hsl(224, 100%, 60%);
  --sl-color-accent-high: hsl(224, 100%, 85%);

  --sl-color-text: var(--sl-color-gray-2);
  --sl-color-text-accent: var(--sl-color-accent-high);
  --sl-color-text-invert: var(--sl-color-accent-low);
  --sl-color-bg: var(--sl-color-black);
  --sl-color-bg-nav: var(--sl-color-gray-6);
  --sl-color-bg-sidebar: var(--sl-color-gray-6);
  --sl-color-bg-inline-code: var(--sl-color-gray-5);
  --sl-color-bg-accent: var(--sl-color-accent-high);
  --sl-color-hairline-light: var(--sl-color-gray-5);
  --sl-color-hairline: var(--sl-color-gray-6);
  --sl-color-hairline-shade: var(--sl-color-black);

  --sl-color-backdrop-overlay: hsla(223, 13%, 10%, 0.66);

  /* Shadows (dark mode) */
  --sl-shadow-sm: 0px 1px 1px hsla(0, 0%, 0%, 0.12),
    0px 2px 1px hsla(0, 0%, 0%, 0.24);
  --sl-shadow-md: 0px 8px 4px hsla(0, 0%, 0%, 0.08),
    0px 5px 2px hsla(0, 0%, 0%, 0.08), 0px 3px 2px hsla(0, 0%, 0%, 0.12),
    0px 1px 1px hsla(0, 0%, 0%, 0.15);
  --sl-shadow-lg: 0px 25px 7px hsla(0, 0%, 0%, 0.03),
    0px 16px 6px hsla(0, 0%, 0%, 0.1), 0px 9px 5px hsla(223, 13%, 10%, 0.33),
    0px 4px 4px hsla(0, 0%, 0%, 0.75), 0px 4px 2px hsla(0, 0%, 0%, 0.25);

  /* Text size and line height */
  --sl-text-2xs: 0.75rem; /* 12px */
  --sl-text-xs: 0.8125rem; /* 13px */
  --sl-text-sm: 0.875rem; /* 14px */
  --sl-text-base: 1rem; /* 16px */
  --sl-text-lg: 1.125rem; /* 18px */
  --sl-text-xl: 1.25rem; /* 20px */
  --sl-text-2xl: 1.5rem; /* 24px */
  --sl-text-3xl: 1.8125rem; /* 29px */
  --sl-text-4xl: 2.1875rem; /* 35px */
  --sl-text-5xl: 2.625rem; /* 42px */
  --sl-text-6xl: 4rem; /* 64px */

  --sl-text-body: var(--sl-text-base);
  --sl-text-body-sm: var(--sl-text-xs);
  --sl-text-code: var(--sl-text-sm);
  --sl-text-code-sm: var(--sl-text-xs);
  --sl-text-h1: var(--sl-text-4xl);
  --sl-text-h2: var(--sl-text-3xl);
  --sl-text-h3: var(--sl-text-2xl);
  --sl-text-h4: var(--sl-text-xl);
  --sl-text-h5: var(--sl-text-lg);

  --sl-line-height: 1.75;
  --sl-line-height-headings: 1.2;

  --sl-font-system: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont,
    "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif,
    "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
  --sl-font-system-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
    "Liberation Mono", "Courier New", monospace;
  --__sl-font: var(--sl-font, var(--sl-font-system)), var(--sl-font-system);
  --__sl-font-mono: var(--sl-font-mono, var(--sl-font-system-mono)),
    var(--sl-font-system-mono);

  /** Key layout values */
  --sl-nav-height: 3.5rem;
  --sl-nav-pad-x: 1rem;
  --sl-nav-pad-y: 0.75rem;
  --sl-mobile-toc-height: 3rem;
  --sl-sidebar-width: 18.75rem;
  --sl-sidebar-pad-x: 1rem;
  --sl-content-width: 45rem;
  --sl-content-pad-x: 1rem;
  --sl-menu-button-size: 2rem;
  --sl-nav-gap: var(--sl-content-pad-x);
  /* Offset required to show outline inside an element instead of round the outside */
  --sl-outline-offset-inside: -0.1875rem;

  /* Global z-index values */
  --sl-z-index-toc: 4;
  --sl-z-index-menu: 5;
  --sl-z-index-navbar: 10;
  --sl-z-index-skiplink: 20;
}

:root[data-theme="light"],
[data-theme="light"] ::backdrop {
  /* Colours (light mode) */
  --sl-color-white: hsl(224, 10%, 10%);
  --sl-color-gray-1: hsl(224, 14%, 16%);
  --sl-color-gray-2: hsl(224, 10%, 23%);
  --sl-color-gray-3: hsl(224, 7%, 36%);
  --sl-color-gray-4: hsl(224, 6%, 56%);
  --sl-color-gray-5: hsl(224, 6%, 77%);
  --sl-color-gray-6: hsl(224, 20%, 94%);
  --sl-color-gray-7: hsl(224, 19%, 97%);
  --sl-color-black: hsl(0, 0%, 100%);

  --sl-color-orange-high: hsl(var(--sl-hue-orange), 80%, 25%);
  --sl-color-orange: hsl(var(--sl-hue-orange), 90%, 60%);
  --sl-color-orange-low: hsl(var(--sl-hue-orange), 90%, 88%);
  --sl-color-green-high: hsl(var(--sl-hue-green), 80%, 22%);
  --sl-color-green: hsl(var(--sl-hue-green), 90%, 46%);
  --sl-color-green-low: hsl(var(--sl-hue-green), 85%, 90%);
  --sl-color-blue-high: hsl(var(--sl-hue-blue), 80%, 30%);
  --sl-color-blue: hsl(var(--sl-hue-blue), 90%, 60%);
  --sl-color-blue-low: hsl(var(--sl-hue-blue), 88%, 90%);
  --sl-color-purple-high: hsl(var(--sl-hue-purple), 90%, 30%);
  --sl-color-purple: hsl(var(--sl-hue-purple), 90%, 60%);
  --sl-color-purple-low: hsl(var(--sl-hue-purple), 80%, 90%);
  --sl-color-red-high: hsl(var(--sl-hue-red), 80%, 30%);
  --sl-color-red: hsl(var(--sl-hue-red), 90%, 60%);
  --sl-color-red-low: hsl(var(--sl-hue-red), 80%, 90%);

  --sl-color-accent-high: hsl(234, 80%, 30%);
  --sl-color-accent: hsl(234, 90%, 60%);
  --sl-color-accent-low: hsl(234, 88%, 90%);

  --sl-color-text-accent: var(--sl-color-accent);
  --sl-color-text-invert: var(--sl-color-black);
  --sl-color-bg-nav: var(--sl-color-gray-7);
  --sl-color-bg-sidebar: var(--sl-color-bg);
  --sl-color-bg-inline-code: var(--sl-color-gray-6);
  --sl-color-bg-accent: var(--sl-color-accent);
  --sl-color-hairline-light: var(--sl-color-gray-6);
  --sl-color-hairline-shade: var(--sl-color-gray-6);

  --sl-color-backdrop-overlay: hsla(225, 9%, 36%, 0.66);

  /* Shadows (light mode) */
  --sl-shadow-sm: 0px 1px 1px hsla(0, 0%, 0%, 0.06),
    0px 2px 1px hsla(0, 0%, 0%, 0.06);
  --sl-shadow-md: 0px 8px 4px hsla(0, 0%, 0%, 0.03),
    0px 5px 2px hsla(0, 0%, 0%, 0.03), 0px 3px 2px hsla(0, 0%, 0%, 0.06),
    0px 1px 1px hsla(0, 0%, 0%, 0.06);
  --sl-shadow-lg: 0px 25px 7px rgba(0, 0, 0, 0.01),
    0px 16px 6px hsla(0, 0%, 0%, 0.03), 0px 9px 5px hsla(223, 13%, 10%, 0.08),
    0px 4px 4px hsla(0, 0%, 0%, 0.16), 0px 4px 2px hsla(0, 0%, 0%, 0.04);
}

@media (min-width: 50em) {
  :root {
    --sl-nav-height: 4rem;
    --sl-nav-pad-x: 1.5rem;
    --sl-text-h1: var(--sl-text-5xl);
    --sl-text-h2: var(--sl-text-4xl);
    --sl-text-h3: var(--sl-text-3xl);
    --sl-text-h4: var(--sl-text-2xl);
  }
}

@media (min-width: 72rem) {
  :root {
    --sl-content-pad-x: 1.5rem;
    --sl-mobile-toc-height: 0rem;
  }
}

*,
*::before,
*::after {
  box-sizing: border-box;
}

* {
  margin: 0;
}

html {
  color-scheme: dark;
  accent-color: var(--sl-color-accent);
}

html[data-theme="light"] {
  color-scheme: light;
}

body {
  font-family: var(--__sl-font);
  line-height: var(--sl-line-height);
  -webkit-font-smoothing: antialiased;
  color: var(--sl-color-text);
  background-color: var(--sl-color-bg);
}

input,
button,
textarea,
select {
  font: inherit;
}

p,
h1,
h2,
h3,
h4,
h5,
h6,
code {
  overflow-wrap: anywhere;
}

code {
  font-family: var(--__sl-font-mono);
}

:root {
  --astro-code-color-text: var(--sl-color-white);
  --astro-code-color-background: var(--sl-color-gray-6);
  --astro-code-token-constant: var(--sl-color-blue-high);
  --astro-code-token-string: var(--sl-color-green-high);
  --astro-code-token-comment: var(--sl-color-gray-2);
  --astro-code-token-keyword: var(--sl-color-purple-high);
  --astro-code-token-parameter: var(--sl-color-red-high);
  --astro-code-token-function: var(--sl-color-red-high);
  --astro-code-token-string-expression: var(--sl-color-green-high);
  --astro-code-token-punctuation: var(--sl-color-gray-2);
  --astro-code-token-link: var(--sl-color-blue-high);
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}

.sl-hidden {
  display: none;
}
.sl-flex {
  display: flex;
}
.sl-block {
  display: block;
}
@media (min-width: 50rem) {
  .md\:sl-hidden {
    display: none;
  }
  .md\:sl-flex {
    display: flex;
  }
  .md\:sl-block {
    display: block;
  }
}
@media (min-width: 72rem) {
  .lg\:sl-hidden {
    display: none;
  }
  .lg\:sl-flex {
    display: flex;
  }
  .lg\:sl-block {
    display: block;
  }
}
[data-theme="light"] .light\:sl-hidden {
  display: none;
}
[data-theme="dark"] .dark\:sl-hidden {
  display: none;
}

/*
Flip an element around the y-axis when in an RTL context.
Primarily useful for things where we can’t rely on writing direction like icons.

<Icon name="right-arrow" class="rtl:flip" />

In a LTR context: →					In a RTL context: ←
*/
[dir="rtl"] .rtl\:flip:not(:where([dir="rtl"] [dir="ltr"] *)) {
  transform: matrix(-1, 0, 0, 1, 0, 0);
}

@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-Thin.woff2") format("woff2");
    font-weight: 100;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-ThinItalic.woff2") format("woff2");
    font-weight: 100;
    font-style: italic;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-ExtraLight.woff2") format("woff2");
    font-weight: 200;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-ExtraLightItalic.woff2") format("woff2");
    font-weight: 200;
    font-style: italic;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-Light.woff2") format("woff2");
    font-weight: 300;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-LightItalic.woff2") format("woff2");
    font-weight: 300;
    font-style: italic;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-Regular.woff2") format("woff2");
    font-weight: 400;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-Italic.woff2") format("woff2");
    font-weight: 400;
    font-style: italic;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-Medium.woff2") format("woff2");
    font-weight: 500;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-MediumItalic.woff2") format("woff2");
    font-weight: 500;
    font-style: italic;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-SemiBold.woff2") format("woff2");
    font-weight: 600;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-SemiBoldItalic.woff2") format("woff2");
    font-weight: 600;
    font-style: italic;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-Bold.woff2") format("woff2");
    font-weight: 700;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-BoldItalic.woff2") format("woff2");
    font-weight: 700;
    font-style: italic;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-ExtraBold.woff2") format("woff2");
    font-weight: 800;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-ExtraBoldItalic.woff2") format("woff2");
    font-weight: 800;
    font-style: italic;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-ExtraBold.woff2") format("woff2");
    font-weight: 900;
    font-style: normal;
}
@font-face {
    font-family: "Jetbrains Mono";
    src: url("/fonts/JetBrainsMono-ExtraBoldItalic.woff2") format("woff2");
    font-weight: 900;
    font-style: italic;
}

@font-face {
  font-family: 'Monocraft';
  src: url('/fonts/Monocraft.ttf') format('truetype');
  font-weight: 400;
  font-style: normal;
  font-display: swap;
}

:root {
  --primary: #54d7a9;
  --secondary: #9b7eca;
  --accent: #7ed4fb;
  --extra: #8fcc75;
  --text: #ffffff;
  --alt: #999999;
  --background: #000000;
  --font: "Jetbrains Mono", monospace;
}

* {
  margin: 0;
  padding: 0;
}

html {
  background-color: var(--background);
  color: var(--text);
  font-family: var(--font);
}

body {
  zoom: 1;
}

h1,
h2,
h3 {
  font-size: 1rem;
  font-weight: bold;
}

a {
  font-weight: 300;
}
a:link,
a:visited,
a:active {
  background-color: transparent;
  text-decoration: none;
  color: var(--secondary);
}
a:hover {
  transition-duration: 0.2s;
  transition-property: color, text-decoration;
  text-decoration: underline;
  color: var(--accent);
}

main {
	display: block;
	position: absolute;
	padding-top: 10%;
	padding-bottom: 10%;
	left: 50%;
	transform: translate(-50%, 0%);
	background: var(--background);
	font-weight: 300;
}
`
