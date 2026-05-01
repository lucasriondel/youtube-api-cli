# `yt` — Product Requirements Document

Scope: an agent-friendly CLI over the YouTube Data API v3 for inspecting and reorganizing the authenticated user's YouTube account.

This PRD maps the full API surface (per https://developers.google.com/youtube/v3/docs) to CLI commands, marks current state, and prioritizes the work. Constraints from `CLAUDE.md` (quota budget, Watch Later/History inaccessible, OAuth2 Desktop flow, no config file, structured output) apply throughout.

---

## Status legend

- ✅ implemented
- 🚧 partial
- 🎯 priority — next batch
- 📋 planned
- ❌ out of scope (explained inline)

---

## API surface → CLI mapping

### Authentication & session

| API | CLI | Status |
|---|---|---|
| OAuth2 Desktop loopback flow | `yt auth credentials <path>` | ✅ |
| OAuth2 token exchange | `yt auth login` | ✅ |
| Token introspection | `yt auth status` | ✅ |
| Token/credential revocation | `yt auth logout [--all]` | ✅ |

No additional work planned. Scope changes for new commands must force re-login (per CLAUDE.md).

---

### Playlists (`playlists` resource)

API methods: `list`, `insert`, `update`, `delete`. Cost: 1 / 50 / 50 / 50.

| Command | API call | Status | Notes |
|---|---|---|---|
| `yt playlists list` | `playlists.list (mine=true)` | ✅ | |
| `yt playlists show <id>` | `playlists.list (id=...)` | ✅ | |
| `yt playlists create --title --description --privacy` | `playlists.insert` | ✅ | 50 units. Supports `--dry-run`. |
| `yt playlists update <id> [--title --description --privacy]` | `playlists.update` | ✅ | 50 units (+1 read). Patch semantics: fetches current snippet/status, overlays provided flags. Supports `--dry-run`. |
| `yt playlists delete <id>` | `playlists.delete` | 🎯 | 50 units. Confirm prompt unless `--yes`. |

---

### Playlist items (`playlistItems` resource)

API methods: `list`, `insert`, `update`, `delete`. Cost: 1 / 50 / 50 / 50.

| Command | API call | Status | Notes |
|---|---|---|---|
| `yt items list <playlist-id>` | `playlistItems.list` | ✅ | |
| `yt items add <playlist-id> <video-id>...` | `playlistItems.insert` | 🎯 | 50 units per video. |
| `yt items remove <playlist-id> <item-id>...` | `playlistItems.delete` | 🎯 | Takes `playlistItemId`, not videoId. Add resolution helper. |
| `yt items move <playlist-id> <item-id> --to <position>` | `playlistItems.update` | 🎯 | 50 units. |
| `yt items sort <playlist-id> --by=title\|date\|duration\|channel [--reverse]` | local sort + `playlistItems.update` per moved item | 📋 | Mandatory `--dry-run` first. Quota-budget check before applying. |
| `yt items dedupe <playlist-id>` | `playlistItems.delete` for duplicates | 📋 | Detect duplicate videoIds; preserve earliest. |

---

### Videos (`videos` resource)

API methods: `list`, `insert`, `update`, `delete`, `rate`, `getRating`, `reportAbuse`. Cost: 1 / 1600 / 50 / 50 / 50 / 1 / 50.

| Command | API call | Status | Notes |
|---|---|---|---|
| `yt videos show <id>...` | `videos.list (id=...)` | 🎯 | snippet,contentDetails,statistics,status. |
| `yt videos mine` | `videos.list (myRating=like)` + own channel uploads | 📋 | Convenience for "my uploads". |
| `yt videos rate <id> --as=like\|dislike\|none` | `videos.rate` | 🎯 | 50 units. |
| `yt videos rating <id>...` | `videos.getRating` | 🎯 | |
| `yt videos update <id> [--title --description --tags --category]` | `videos.update` | 📋 | Patch semantics. |
| `yt videos delete <id>` | `videos.delete` | 📋 | Confirm prompt. |
| `yt videos report <id> --reason=<id> [--comment]` | `videos.reportAbuse` | ❌ | Out of scope — abuse reporting belongs in the YouTube UI, not an automation surface. |
| `yt videos upload <path>` | `videos.insert` | ❌ | 1600 units. Out of scope: this CLI is for organizing, not uploading. Revisit if a real use case appears. |

---

### Liked videos & watch history

| Command | Status | Notes |
|---|---|---|
| `yt liked list` | 🎯 | The `LL` playlist is still accessible via `playlistItems.list`. Roadmap item #4 in CLAUDE.md. |
| `yt liked add/remove <video-id>` | 📋 | Implemented as `videos.rate` wrappers — prefer that over playlistItems on `LL`. |
| Watch Later (`WL`) | ❌ | API access removed by Google in 2016. Hard constraint. |
| Watch History (`HL`) | ❌ | Same — no API access. |

A separate browser-automation path for `WL` is on the roadmap (CLAUDE.md), opt-in only, not part of this PRD.

---

### Subscriptions (`subscriptions` resource)

API methods: `list`, `insert`, `delete`. Cost: 1 / 50 / 50.

| Command | API call | Status | Notes |
|---|---|---|---|
| `yt subs list [--mine]` | `subscriptions.list (mine=true)` | 🎯 | |
| `yt subs add <channel-id>` | `subscriptions.insert` | 📋 | 50 units. |
| `yt subs remove <subscription-id>` | `subscriptions.delete` | 📋 | Takes subscription resource id, not channel id. |

---

### Channels (`channels` resource)

API methods: `list`, `update`. Cost: 1 / 50.

| Command | API call | Status | Notes |
|---|---|---|---|
| `yt channels show [<id>\|--mine]` | `channels.list` | 🎯 | snippet,contentDetails,statistics,brandingSettings. |
| `yt channels update --description --keywords --country` | `channels.update` | 📋 | Patch only changed branding fields. |

`channelBanners.insert`, `watermarks.set/unset`, `thumbnails.set` — ❌ out of scope (asset uploads, not the CLI's purpose).

---

### Channel sections (`channelSections` resource)

API methods: `list`, `insert`, `update`, `delete`. Cost: 1 / 50 / 50 / 50.

| Command | Status | Notes |
|---|---|---|
| `yt sections list [--mine]` | 📋 | Low priority. |
| `yt sections {create,update,delete}` | 📋 | Low priority — niche feature. |

---

### Search (`search.list`)

Cost: **100 units** per call — expensive. Per CLAUDE.md, document quota cost in `--help` and prefer cheaper alternatives where possible.

| Command | Status | Notes |
|---|---|---|
| `yt search "<query>" [--type=video\|channel\|playlist] [--max=50] [--channel <id>]` | 🎯 | One call = 100 units; warn loudly if `--max` would force pagination. |

---

### Comments & comment threads

API methods (Comments): `list`, `insert`, `update`, `setModerationStatus`, `delete`.
API methods (CommentThreads): `list`, `insert`.

| Command | Status | Notes |
|---|---|---|
| `yt comments list --video <id> \| --channel <id>` | 📋 | Listing is cheap (1). |
| `yt comments thread <thread-id>` | 📋 | Fetch full thread. |
| `yt comments reply <parent-id> --text "..."` | 📋 | `comments.insert`. |
| `yt comments post --video <id> --text "..."` | 📋 | `commentThreads.insert`. |
| `yt comments {update,delete,moderate}` | ❌ | Not a priority for the "organize my account" use case; can add later if asked. |

---

### Captions (`captions` resource)

API methods: `list`, `insert`, `update`, `download`, `delete`.

| Command | Status | Notes |
|---|---|---|
| `yt captions list <video-id>` | 📋 | |
| `yt captions download <caption-id> [--format=sbv\|srt\|vtt] [-o file]` | 📋 | Useful for archiving/transcripts. |
| `yt captions {insert,update,delete}` | ❌ | Authoring captions is out of scope. |

---

### Activities (`activities.list`)

| Command | Status | Notes |
|---|---|---|
| `yt activity [--mine \| --channel <id>] [--since <date>]` | 📋 | Recent uploads/likes/etc. Useful for agents auditing changes. |

---

### Reference data (cheap, read-only)

| API | Command | Status |
|---|---|---|
| `videoCategories.list` | `yt ref categories [--region=US]` | 📋 |
| `i18nLanguages.list` | `yt ref languages` | 📋 |
| `i18nRegions.list` | `yt ref regions` | 📋 |
| `videoAbuseReportReasons.list` | ❌ — only useful for `videos.reportAbuse`, which is out of scope. |

Group under `yt ref ...` to keep the top-level command list tidy.

---

### Memberships (`members`, `membershipsLevels`)

| Command | Status | Notes |
|---|---|---|
| `yt members list` | ❌ | Requires a YouTube channel with active memberships enabled. Not in scope for a personal-account CLI. |
| `yt members levels` | ❌ | Same. |

---

## Cross-cutting features

These apply to all commands that mutate state or do bulk reads.

### `--dry-run` (mandatory on every write command)

Print the planned mutation set + total estimated quota cost. Exit 0 without calling the API. Quota cost reference table lives in `internal/ytapi/quota.go` (to be added).

### Quota awareness

- Every command's `--help` documents its quota cost (e.g. `cost: 1 unit per call`).
- Commands that may exceed 1000 units in a single invocation prompt for confirmation unless `--yes`.
- A `yt quota` command (📋) summarizes today's estimated spend by reading a local rolling counter (best-effort; the API doesn't expose remaining quota).

### Read cache

🎯 Priority for `items list`, `items sort`, `items dedupe`, and any future bulk-read flows.

- On-disk cache in `os.UserCacheDir()/yt/` keyed by `(endpoint, params, etag)`.
- Honor `If-None-Match` so a refetch costs 0 units when unchanged.
- `--no-cache` flag, and a `yt cache clear` command.

### Output (already in place)

Default table / `--json` (raw API objects) / `--plain` (TSV). No per-command output flags — extend the global set if needed.

### Pagination

- Default: fetch all pages for `*.list` commands (most are 1 unit/page).
- `--max <n>` to cap. `--page-token <token>` to resume. JSON output includes `nextPageToken` so agents can paginate themselves.

### Error mapping

- `quotaExceeded` → exit 7, message points at https://developers.google.com/youtube/v3/determine_quota_cost.
- `playlistOperationUnsupported` (e.g. on `WL`) → exit 6, message references the hard constraint.
- `403 insufficientPermissions` → exit 5, suggest `yt auth login` with the new scope.

---

## Prioritized roadmap

The order below resolves the loose ordering in CLAUDE.md against the gaps above.

### Milestone 1 — playlist & item CRUD (next)
1. `playlists create` / `update` / `delete`
2. `items add` / `remove` / `move`
3. `videos show <id>...` (needed by the rest as a lookup helper)
4. Quota cost helper + `--dry-run` infrastructure shared across all writes

### Milestone 2 — agent-friendly bulk ops
5. Read cache (etag-aware) under `os.UserCacheDir()/yt/`
6. `items sort` (with mandatory `--dry-run`)
7. `items dedupe`

### Milestone 3 — discovery & ratings
8. `liked list`, `videos rate`, `videos rating`
9. `search "<query>"` with loud quota warning
10. `subs list`, `channels show`

### Milestone 4 — secondary surfaces
11. `subs add` / `remove`
12. `videos update` / `delete`
13. `channels update`
14. `activity`, `comments list/post/reply`, `captions list/download`
15. `ref categories|languages|regions`

### Out of scope (recorded so we don't re-litigate)
- `videos.insert` (uploads), `videos.reportAbuse`
- `channelBanners.insert`, `watermarks.set/unset`, `thumbnails.set`
- `members.*`, `membershipsLevels.*`
- `captions.{insert,update,delete}`, `comments.{update,delete,moderate}`
- Anything Watch Later / Watch History via the API (Google removed this in 2016 — browser-automation path is a separate project)

---

## Open questions

1. Should `items add` accept video URLs as well as IDs? (Convenient, trivial to parse — likely yes.)
2. `yt quota` — do we maintain a local counter, or just document costs and skip? (Lean toward skip until a user actually hits a wall.)
3. Cache TTL for `videos.list` responses on items containing video metadata — videos mutate (titles change, get deleted). Default 24h with `--refresh`?
