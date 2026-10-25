package search

import (
	"strings"

	"github.com/yoanbernabeu/grepai/store"
)

// PreferOverlayResults keeps overlay hits when the same relative path also
// appears under the declared upstream project.
func PreferOverlayResults(results []store.SearchResult, workspace, overlay, upstream string) []store.SearchResult {
	if overlay == "" || upstream == "" {
		return results
	}
	overlayPrefix := workspace + "/" + overlay + "/"
	upstreamPrefix := workspace + "/" + upstream + "/"
	overlayRel := map[string]struct{}{}
	for _, r := range results {
		if strings.HasPrefix(r.Chunk.FilePath, overlayPrefix) {
			overlayRel[strings.TrimPrefix(r.Chunk.FilePath, overlayPrefix)] = struct{}{}
		}
	}
	if len(overlayRel) == 0 {
		return results
	}
	out := make([]store.SearchResult, 0, len(results))
	for _, r := range results {
		if strings.HasPrefix(r.Chunk.FilePath, upstreamPrefix) {
			rel := strings.TrimPrefix(r.Chunk.FilePath, upstreamPrefix)
			if _, ok := overlayRel[rel]; ok {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}
