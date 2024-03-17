package collection

import "com.mutantcat.cloud_step/util"

// 通过way获得路径
func GetPath(way string) string {
	MWorkCllection.Lock()
	MSelfHelpMode.Lock()
	defer MSelfHelpMode.Unlock()
	defer MWorkCllection.Unlock()
	if _, ok := SelfHelpMode[way]; !ok {
		return ""
	}
	var now = SelfHelpMode[way]
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
		SelfHelpMode[way] = now
		return path
	}
	return ""
}
