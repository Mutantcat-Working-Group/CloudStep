package collection

import (
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
)

// filterProxyAlive returns only urls whose Alive == true. Copy of the
// self_help_mode.go helper kept in this file so the two helpers can evolve
// independently (e.g. future retry-count probing for the proxy layer).
// An empty result means "no usable address layer"; the call site returns "".
func filterProxyAlive(urls []entity.Url) []entity.Url {
	out := make([]entity.Url, 0, len(urls))
	for _, u := range urls {
		if u.Alive {
			out = append(out, u)
		}
	}
	return out
}

func GetProxyPath(way string) string {
	MWorkCllection.Lock()
	MProxyMode.Lock()
	defer MProxyMode.Unlock()
	if _, ok := ProxyMode[way]; !ok {
		MWorkCllection.Unlock()
		return ""
	}
	var now = ProxyMode[way]
	// Take a snapshot of the collection slice under the collection lock, then
	// release it. Filtering and selection run on the snapshot so the
	// collection lock is not held while scanning potentially many dead urls.
	urls := WorkCllection[now.Point]
	MWorkCllection.Unlock()

	alive := filterProxyAlive(urls)
	if len(alive) == 0 {
		return ""
	}

	switch now.Mode {
	case "random":
		now.Index = util.RandInt(0, len(alive))
		return alive[now.Index].Path
	case "polling":
		if now.Index < 0 || now.Index >= len(alive) {
			now.Index = 0
		}
		path := alive[now.Index].Path
		now.Index++
		if now.Index >= len(alive) {
			now.Index = 0
		}
		ProxyMode[way] = now
		return path
	}
	return ""
}

func GetProxyMode(way string) string {
	MProxyMode.Lock()
	defer MProxyMode.Unlock()
	if _, ok := ProxyMode[way]; !ok {
		return ""
	}
	return ProxyMode[way].ProxyMode
}
