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
	// 默认配置兜底: way 缺失时, 使用 system_config 里配置的默认映射集名称替换 way。
	if way == "" {
		if resolved, ok := util.ResolveWayCollection(way, util.GetSysConfigMirror().AgentDefaultCollectionId); ok {
			way = resolved
		} else {
			return ""
		}
	}

	// 读路径: 两把 RLock, random 模式全程无写锁。
	MWorkCllection.RLock()
	MProxyMode.RLock()
	if _, ok := ProxyMode[way]; !ok {
		MProxyMode.RUnlock()
		MWorkCllection.RUnlock()
		return ""
	}
	var now = ProxyMode[way]
	// Take a snapshot of the collection slice under the collection lock, then
	// release it. Filtering and selection run on the snapshot so the
	// collection lock is not held while scanning potentially many dead urls.
	urls := WorkCllection[now.Point]
	MWorkCllection.RUnlock()

	alive := filterProxyAlive(urls)
	if len(alive) == 0 {
		MProxyMode.RUnlock()
		return ""
	}

	switch now.Mode {
	case "random":
		// random 模式: 全程读锁, 不写回 index, 零阻塞。
		now.Index = util.RandInt(0, len(alive))
		MProxyMode.RUnlock()
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
		MProxyMode.RUnlock()

		// polling 写回 index: 仅此几微秒用写锁。
		MProxyMode.Lock()
		if e, ok := ProxyMode[way]; ok {
			e.Index = now.Index
			ProxyMode[way] = e
		}
		MProxyMode.Unlock()
		return path
	}

	MProxyMode.RUnlock()
	return ""
}

func GetProxyMode(way string) string {
	MProxyMode.RLock()
	defer MProxyMode.RUnlock()
	if _, ok := ProxyMode[way]; !ok {
		return ""
	}
	return ProxyMode[way].ProxyMode
}
