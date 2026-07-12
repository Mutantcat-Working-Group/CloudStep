package util

// defaultCollectionResolver 把 collection id 映射为名称。它由 dao 在
// InitSystemConfig 时注入(dao.GetCollectionNameById),避免 util 反过来 import dao
// 形成 dao → collection → util → da 循环。注入前(或未注入)调用 ResolveWayCollection
// 会安全地视为 miss。
var defaultCollectionResolver func(int) string

// SetDefaultCollectionResolver 由 dao 启动时注入 id→name 查询。单测可传入 mock。
func SetDefaultCollectionResolver(fn func(int) string) {
	defaultCollectionResolver = fn
}

// ResolveWayCollection 在 /self /agent 的 way 缺失时调用一次,把缺失的 way 落到
// 配置的默认映射集名称上。
//
//   - way != "" → ("", false): 让现有路径自行处理,默认配置完全不介入
//   - defaultId <= 0 → ("", false): 未配置默认(0 表示"清除默认")
//   - defaultId > 0 但 collection miss → ("", false): 误配,回落到 404
//   - defaultId > 0 且存在 → (collectionName, true)
//
// 通过注入的 resolver 走 DB 读 collection 表(单行 pk 查),本函数本身不持锁、不 import dao。
func ResolveWayCollection(way string, defaultId int) (collName string, ok bool) {
	if way != "" || defaultId <= 0 {
		return "", false
	}
	if defaultCollectionResolver == nil {
		return "", false
	}
	name := defaultCollectionResolver(defaultId)
	if name == "" {
		return "", false
	}
	return name, true
}
