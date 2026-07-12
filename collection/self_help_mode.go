package collection

import (
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
)

// filterAlive returns only urls whose Alive == true. Preserving the existing
// mode contract: an empty result means "no usable address layer" and the
// call site must return "".
func filterAlive(urls []entity.Url) []entity.Url {
	out := make([]entity.Url, 0, len(urls))
	for _, u := range urls {
		if u.Alive {
			out = append(out, u)
		}
	}
	return out
}

// 通过way获得路径
func GetPath(way string) string {
	// 默认配置兜底: way 缺失时, 使用 system_config 里配置的默认映射集名称替换 way。
	if way == "" {
		if resolved, ok := util.ResolveWayCollection(way, util.GetSysConfigMirror().SelfDefaultCollectionId); ok {
			way = resolved
		} else {
			return ""
		}
	}
	MWorkCllection.Lock()
	MSelfHelpMode.Lock()
	defer MSelfHelpMode.Unlock()
	if _, ok := SelfHelpMode[way]; !ok {
		MWorkCllection.Unlock()
		return ""
	}
	var now = SelfHelpMode[way]
	// Take a snapshot of the collection slice under the collection lock, then
	// release it. Filtering and selection run on the snapshot so the
	// collection lock is not held while scanning potentially many dead urls.
	urls := WorkCllection[now.Point]
	MWorkCllection.Unlock()

	alive := filterAlive(urls)
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
		SelfHelpMode[way] = now
		return path
	}
	return ""
}
