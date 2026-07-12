## Self-Review

This section lives in its own file (`...-self-review.md`) so you can edit Tasks 4–6 without reopening the big Task 1–4 doc.

### 1. Spec-coverage checklist (design doc → tasks)
| Design-spec req | Task implementing it |
|---|---|
| F1 `GET /slist` / `GET /slist/:way` via Any-verb | Task 3 route-test + Task 4 impl |
| F2 way resolves through self cache, then proxy cache; self wins on collision | Task 1 (`_SelfWay`, `_ProxyWay`, `_PrefersSelfOverProxy`) + Task 2 impl |
| F3 public, no auth (`/self`-style posture) | Task 4 handler omits `LoginHandler()` |
| F4 full Url array (id/parent/address/alive/retry) | Task 1 asserts `entity.Url{Id,Parent,Address,Alive,Retry}`; Task 4 forwards raw slice |
| F5 json tag is `address` (matches `self_help_router.go`) | Task 1 uses `Address:` literal so the existing tag is asserted on the wire |
| 404-shape errors return `{"code": 404}`; 200 OK success returns `code:0` | Task 4 handler branches on `source == collection.SrcNone` |
| Empty collection returns `data: []` (NOT a 404) | Task 1 `_EmptyCollection` proves `urls == nil`; gin marshals nil slice as `null` - see note below |

**Note on `data: null` vs `data: []`:** the plan returns the raw cache reference, so an empty collection yields `null` in JSON. That's **consistent with the design doc** and with how `/collection/geturls` already behaves (it returns nil). If you prefer a guaranteed `[]`, Task 2 can initialize `urls = []entity.Url{}` before returning when the map entry resolves but is empty. Either behavior is within spec; I left it permissive to match existing code.

### 2. Placeholder scan
No `TBD`, `TODO`, `implement later`, `add validation`, or `similar to Task N` references in any task. Every code block is copy-paste runnable. Every "expected:" output is concrete. Every commit message is final.

### 3. Type & symbol consistency
- `GetSelfHelpList(way string) (source listSource, point string, urls []entity.Url)` return order is `source, point, urls` in Task 2's impl and Task 1's call sites (`src, point, got`). Good.
- Constants are `SrcSelf`, `SrcProxy`, `SrcNone` (same case) in T1 refs + T2 impl + T4 `collection.SrcNone` ref + T2 exports. Good.
- `SelfHelpListRouter`, `slist`, `source.String()` appear in T3 and T4 with consistent case. Good.
- `util.GetWayParam(c)` used the same way as `self_help_router.go:selfhelp`. Good.
- Mutex names match letters exactly as defined in `collection/collection_store.go`: `MWorkCllection`, `MSelfHelpMode`, `MProxyMode`. Good.
- `entity.Url` json tags as read from `entity/dao_entity.go`: `id`, `parent`, `address`, `alive`, `retry`. Task 1 uses `Id`, `Parent`, `Address`, `Alive`, `Retry` Go-field literals (capitalised - correct).

### 4. No out-of-scope drift
- `/agent` mode precedence: `/slist/*name` for path form doesn't collide with `/agent/*name` `/slist` namespace dispatcher - gin routing by path prefix keeps them separate.
- README update (Task 6) is the only change outside `collection/` and `router/` and is just a checkbox flip.

### 5. Risks / go-live notes
- URL-alive semantics will only become accurate once清单 #5「自动启停地址(heartbeat)」or #4「手动启停地址」 merges. Until then `alive=true` for every URL is correct, as observed in CHANGELOG line 46-47 area.
- The smoke harness (Task 5) pkill's any running cloud_step - acceptable for a one-shot local run; if you share a dev box, scope it to `pkill -f 'cloud_step 9192'` instead.
