package collection

import "com.mutantcat.cloud_step/util"

// 通过way获得路径
func GetPath(way string) string {
	mWorkCllection.Lock()
	mSelfHelpMode.Lock()
	defer mSelfHelpMode.Unlock()
	defer mWorkCllection.Unlock()
	if _, ok := SelfHelpMode[way]; !ok {
		return ""
	}
	var now = SelfHelpMode[way]
	switch now.Mode {
	case "random":
		now.Index = util.RandInt(0, len(WorkCllection[now.WorkCllection]))
		return WorkCllection[now.WorkCllection][now.Index].Path
	case "polling":
		path := WorkCllection[now.WorkCllection][now.Index].Path
		now.Index++
		if now.Index >= len(WorkCllection[now.WorkCllection]) {
			now.Index = 0
		}
		return path
	}
	return ""
}
