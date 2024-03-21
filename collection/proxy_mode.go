package collection

import "com.mutantcat.cloud_step/util"

func GetProxyPath(way string) string {
	MWorkCllection.Lock()
	MProxyMode.Lock()
	defer MProxyMode.Unlock()
	defer MWorkCllection.Unlock()
	if _, ok := ProxyMode[way]; !ok {
		return ""
	}
	var now = ProxyMode[way]
	switch now.Mode {
	case "random":
		now.Index = util.RandInt(0, len(WorkCllection[now.Point]))
		return WorkCllection[now.Point][now.Index].Path
	case "polling":
		path := WorkCllection[now.Point][now.Index].Path
		now.Index++
		if now.Index >= len(WorkCllection[now.Point]) {
			now.Index = 0
		}
		ProxyMode[way] = now
		return path
	}
	return ""
}
