package collection

import "com.mutantcat.cloud_step/entity"

// listSource classifies which in-memory mode-cache a way was resolved from.
type listSource int

const (
	SrcNone  listSource = iota // way not found in any mode cache
	SrcSelf                    // found in SelfHelpMode
	SrcProxy                   // found in ProxyMode
)

// String returns the wire value for the `mode` response field.
func (s listSource) String() string {
	switch s {
	case SrcSelf:
		return "selfhelp"
	case SrcProxy:
		return "proxy"
	default:
		return ""
	}
}

// GetSelfHelpList resolves `way` to its mapping set's URL list.
//
// Resolution order: SelfHelpMode, then ProxyMode. On collision the
// self-help entry wins (the two caches are not expected to share ways;
// self-help is checked first so its users keep working even if a proxy
// way were accidentally duplicated).
//
// The way is treated as opaque. Each mode entry carries the target
// collection name in its `Point` field (== collection.Name == url.Parent).
// The returned `urls` slice is a direct reference into the package-level
// WorkCllection cache; callers MUST NOT mutate it.
//
// Returns (SrcNone, "", nil) when the way is unknown. Returns (src, point,
// nil) when the collection has zero urls.
func GetSelfHelpList(way string) (source listSource, point string, urls []entity.Url) {
	if way == "" {
		return SrcNone, "", nil
	}

	MSelfHelpMode.Lock()
	sh, ok := SelfHelpMode[way]
	MSelfHelpMode.Unlock()
	if ok {
		MWorkCllection.Lock()
		urls = WorkCllection[sh.Point]
		MWorkCllection.Unlock()
		return SrcSelf, sh.Point, urls
	}

	MProxyMode.Lock()
	px, ok := ProxyMode[way]
	MProxyMode.Unlock()
	if ok {
		MWorkCllection.Lock()
		urls = WorkCllection[px.Point]
		MWorkCllection.Unlock()
		return SrcProxy, px.Point, urls
	}

	return SrcNone, "", nil
}
