package app

import (
	"net/http"
	"strings"
)

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "max-age=3600")

	// Generate the HTML list items for each report
	var listItemsBuilder strings.Builder
	for _, name := range a.reportNames {
		listItemsBuilder.WriteString("<li><a href=\"/r?name=" + name + "\">" + name + "</a></li>\n")
	}
	listItemsHTML := listItemsBuilder.String()

	// Insert the list items into the main HTML response
	responseHTML := `<html>
	<head><title>LSTN</title></head>
	<body>
		<h1>LSTN</h1>
		<code>commit ` + a.commit + `</code>
		<p>Send events to <code>/</code> with headers <code>TYPE</code>, <code>USR</code>, <code>SESS</code> and <code>CID</code>. Optional headers: <code>SCROLLED</code> and <code>PAGE_SECONDS</code></p>
		<p>Get reports from <code>/r</code> with query parameter <code>name</code>:</p>
		<ul>` + listItemsHTML + `</ul>
		<p>Get client-side script from <a href="/js">/js</a>.</p>
		<p>See <a href="https://github.com/swissinfo-ch/lstn">github.com/swissinfo-ch/lstn</a> for more information.</p>
	</body>
</html>`

	w.Write([]byte(responseHTML))
}
