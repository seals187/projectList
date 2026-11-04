# ProjectList

Lean, self-hosted project management tool: **Customers → Projects → Checklists**.
Checklists can be created from reusable templates (as independent copies).
Changes per version: see [CHANGELOG.md](CHANGELOG.md).

- A single Go binary with an embedded SvelteKit frontend, SQLite (pure Go, no CGO)
- One container, target < 50 MB RAM
- Multi-user with roles (`admin`, `member`, `viewer`) and customer assignment
- User interface in English, German and French

## Quick start (Docker)

```bash
cp .env.example .env        # adjust ADMIN_EMAIL / ADMIN_PASSWORD!
docker compose up -d        # pulls the published image ghcr.io/seals187/projectlist:latest
```

Version via `PROJECTLIST_VERSION` in `.env`: `latest` = newest release (default), `edge` =
current development state (every push to `main`), or a fixed version such as `0.0.5`.
Update: `docker compose pull && docker compose up -d`.

Then sign in at <http://localhost:8080> with the admin credentials from `.env`.
The first admin is only created on the very first start (empty database).

Check the status:

```bash
docker compose ps                     # healthcheck status
curl http://localhost:8080/healthz    # {"status":"ok"}
docker compose logs -f projectlist    # structured JSON logs
```

### Published image (ghcr.io)

Every release has an image for **linux/amd64** and **linux/arm64** (e.g. Raspberry Pi):

```bash
docker pull ghcr.io/seals187/projectlist:latest   # or a fixed version, e.g. :0.0.5
```

`docker-compose.yml` already uses this image. Without Compose:

```bash
docker run -d --name projectlist -p 8080:8080 --env-file .env \
  -v projectlist-data:/data --memory 128m ghcr.io/seals187/projectlist:latest
```

Images are published automatically by GitHub Actions (`.github/workflows/docker.yml`): for every tag
`vX.Y.Z` as `:X.Y.Z`, `:X.Y` and `:latest`, and for every push to `main` (after green tests) as `:edge`.
An existing tag can be published afterwards via *Actions → CI and container image → Run workflow*.

For every version tag, the workflow `release.yml` creates the GitHub release with the notes from
`CHANGELOG.md` (for an existing tag: *Actions → Release → Run workflow*).

## Environment variables

| Variable              | Default                | Description |
|-----------------------|------------------------|-------------|
| `ADMIN_EMAIL`         | –                      | Email of the first admin (only with an empty database) |
| `ADMIN_PASSWORD`      | –                      | Password of the first admin (at least 8 characters) |
| `ADMIN_NAME`          | `Admin`                | Display name of the first admin |
| `APP_SUBTITLE`        | –                      | Optional subtitle below "ProjectList" in the sidebar (e.g. company or department name) |
| `PORT`                | `8080`                 | Host port (docker compose only) |
| `PROJECTLIST_VERSION` | `latest`               | Image version for docker compose: `latest`, `edge` or e.g. `0.0.5` |
| `LISTEN_ADDR`         | `:8080`                | Listen address inside the container |
| `DB_PATH`             | `/data/projectlist.db` | Path of the SQLite file |
| `COOKIE_SECURE`       | `true`                 | Send the session cookie over HTTPS only. On `http://localhost` this works in Chrome/Edge/Firefox; set `false` for plain HTTP access via an IP or hostname |
| `SESSION_TTL`         | `720h`                 | Session lifetime (extended while in use) |
| `LOG_LEVEL`           | `info`                 | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT`          | `json`                 | `json` or `text` |

For production, put a reverse proxy with TLS (Caddy, Traefik, nginx) in front of the container.
The proxy must pass the `Host` header through unchanged (origin check).

## Roles & permissions

| Role     | Visibility              | Permissions |
|----------|-------------------------|-------------|
| `admin`  | all customers           | everything, incl. user management (*Settings → Users*), creating/deleting customers, backup |
| `member` | assigned customers only | edit customers; create/change/delete/check projects, checklists and items; manage templates |
| `viewer` | assigned customers only | read only |

All permissions are checked server-side in every endpoint. Objects of other customers return `404`.
The API returns errors as language-neutral codes (e.g. `{"error":"validation","fields":{"name":"required"}}`);
the user interface translates them.

## Usage

- **Language** (English, Deutsch, Français), **theme** (system/light/dark/dark gray) and **font** in *Settings*:
  Inter (default, with optical size), Geist, Mona Sans, Instrument Sans, Figtree, Onest, Plus Jakarta Sans,
  Manrope, DM Sans, Schibsted Grotesk, IBM Plex Sans or system; headings in serif (Source Serif 4) or like the
  text; code in JetBrains Mono. All fonts are embedded (no external requests). The language is taken from the
  browser on the first visit
- **Overview** in five views: *Cards · Board · List · Calendar · Timeline*; the **default view** can be set in
  *Settings* (or "last used"), the view in use is kept until the app is closed. In *cards* and *list*, the projects of
  each customer can be collapsed (default: collapsed, "Expand all" at the top right, state remembered per browser):
  - *Cards*: one card per customer with location, progress ring, its projects (status, progress, due date) and a
    footer with overdue or open items, the next due date and the team; customers with overdue items come first;
    collapsible (collapsed: header and footer as summary)
  - *Board*: columns per status (or grouped by customer / person responsible), move projects by drag & drop
    between and within columns; calm cards (labels as colored dots, progress bar, due date, checklist count,
    avatars), quick actions on hover, "Add project" per column, *Compact* switch, WIP limit per status column
  - *List*: key figures (active projects, open/overdue items, due within 14 days, behind schedule, blocked) and a
    table grouped by customer: per project its checklists as sub-rows (state, progress, due date, person
    responsible), **time vs. progress** (progress bar with a marker for the time elapsed: on track / x% behind /
    overdue) and the latest activity
  - *Calendar*: month grid with the projects on their due date; drag a project onto another day to reschedule it
  - *Timeline*: Gantt bars from the start date (or creation date) to the due date, grouped by customer; drag a bar
    to move it, drag its ends to change start or due date; zoom weeks/months
- **Filters** for all overview views: customer (*All*/*None*), label and person responsible as dropdowns; saved per
  user in the database (the same on all devices). **Saved views** store the view with its filters under a name and
  appear in the sidebar
- **My tasks**: tab *Customers* (customers → projects → checklists you are involved in, with customer progress and
  due dates "yesterday/today/tomorrow") and tab *Personal* (assigned items to check off); filters open / today /
  done (done = last 30 days)
- **Customer progress**: per project, the pie chart icon sets whether it counts towards the customer's overall
  progress (admins and members; viewers only see the state). Excluded projects are shown in gray, and the customer
  shows "excl. N project(s)"; the same calculation is used in My tasks, the customer list and the overview
- **Customers** overview as a list (expandable with projects, checklist progress, due date) or cards; search,
  filters (all, with active projects, critical, without project), sorting (last activity, name, progress, due date,
  open tasks); status per customer: *New* (no projects), *Critical* (project blocked or due/overdue within 3 days),
  *Completed*, otherwise *On track*. Location, contact person and email can be edited on the customer page
- **Project page**: breadcrumb, status, favorite star and actions (duplicate, archive, delete, "+ Checklist"),
  progress in plain words ("37% · 19 of 31 items · 50 days left"), tabs *Checklists · Activity · Files · Notes* and a
  properties panel (status, customer, person responsible, start/end, labels, watchers, Easy Redmine ticket) that is
  edited in place, plus the latest activity. "New project" in the overview can create the first checklist from a
  template right away
- **Activity log** per project: checked-off items (grouped), checklists created/completed/deleted, status, dates,
  person responsible, labels, Redmine ticket and files
- **Files** per project (upload by button or drag & drop, max. 25 MB per file); stored in the database, so they are
  part of every backup. Downloads are always served as attachments
- **Duplicate a project** with description, person responsible, labels and all checklists (items unchecked)
- **Easy Redmine**: ticket number per project; an admin sets the Redmine address once (*Settings → Projects*)
- **Custom status names, colors and WIP limits**: admins change them for everyone directly on the overview board
  (click a column title); empty = default name and color, no limit
- **Tags**: label projects with colored tags (54 preset colors or any color), shown in all overview views;
  assign or create them on the project page, rename, recolor or delete them in Settings → Projects (admin)
- **Person responsible** per project (project page, board card)
- **Context menus** (board and cards): right-click a project (or its "…" button) to open it, open or add checklists, rename it,
  change status, start/due date, person responsible, tags or favorite; right-click a customer to open it, add a
  project or filter the overview (only this customer, hide)
- **Board order**: drag cards between columns (status) and within a column; the order is shared by all users
- **Sidebar**: search and **+** (new project), **favorites** (star on a project in the sidebar or on the project
  page) and **saved views** (both stored per user in the database, so they are the same on every device); sections
  collapsible; at the bottom: user (→ settings), theme toggle, sign out. Customers are opened via *Customers*
- **Ctrl+K** / **⌘K** opens the search (customers, projects, checklists)
- Click the title of a customer, project or checklist to edit it – **Enter** saves, **Esc** cancels

### Block editor (checklists & templates)

Checklists consist of **blocks** that are edited like a document. Changes are saved automatically
(indicator "saved" at the top right).

| Block type | Shortcut while typing | Note |
|---|---|---|
| Checklist item | `[] ` | Checkbox, assignee, due date – only these count towards progress |
| Heading 1–4 | `# ` … `#### ` | H1 forms sections with a counter (also in the sidebar) |
| Normal text | – | |
| Bulleted / numbered list | `- ` / `1. ` | |
| Quote | `> ` | |
| Code block | ` ``` ` | Syntax highlighting (Bash, PowerShell, SQL, JSON, YAML, JS, Python), line numbers, *Copy*, background light gray/dark gray/black (in *Settings → Checklists*) |
| Divider | `---` | |

- **`/`** at the start of a line opens the command menu (e.g. `/hea` → headings); **+** in the left margin
  inserts a block below and opens the menu
- **Enter** splits the block or creates a new one of the same type; Enter in an empty list item ends the list.
  **Backspace** at the start of a line converts to text or merges with the block above.
  **Tab / Shift+Tab** indents/outdents (e.g. a code block below an item). **↑/↓** move between blocks.
- **Toolbar**: undo/redo (**Ctrl+Z / Ctrl+Y**), block type, H1–H4, **bold** (Ctrl+B), *italic* (Ctrl+I),
  underline (Ctrl+U), strikethrough, `inline code` (Ctrl+E), text color, highlight, lists, quote, code block,
  link, divider, clear formatting; on the right **+ Heading** / **+ Item**
- In a code block: **Tab** indents, **Esc** or **Ctrl+Enter** jumps to the next block
- Pasting multiple lines creates one block per line
- **Collapse**: the arrow in front of a heading collapses its section (up to the next heading of the same or a
  higher level, so H2 also collapses its H3/H4). Collapsed headings show progress and the number of hidden blocks;
  at the top right there is *Collapse all / Expand all*. The state is stored per browser (not in the database);
  moving a collapsed heading moves its section along
- **Links**: shown in blue. Clicking a link (or the link button) opens a dialog for **text** and **address
  (URL)**, with *Open* and *Remove link*; without selected text, a new link is inserted at the cursor.
  Ctrl+click opens the link directly
- **Row color**: via the droplet icon on the right of each row (on hover); takes precedence over the section
  color. Inside a colored section there are also *Same as section* (inherit the color) and *No color* (row stays
  white, even inside a colored section)
- **Color shades** like in Word/Excel: every color (gray, yellow, green, blue, red, purple) comes in light,
  medium and strong (highlighter colors such as `#FFFF00`) – for rows, sections, text color and highlight,
  everywhere in the same grid (hues side by side, shades below each other)
- **Convert to checklist item**: the checkbox icon on the right of text, list, quote and heading rows (on hover)
  turns the row into an item with a checkbox (Ctrl+Z undoes it)
- **Status per item**: a free text field next to the checkbox (max. 60 characters) with suggestions such as
  *Fulfilled*, *Achieved*, *Partially*; values already used in the checklist are suggested too.
  The status is not copied into templates
- **Grid lines** around every row like in a table, on/off via the table icon in the toolbar or in
  *Settings → Checklists* (default: on). The notes column never takes the section color
- **Assignee for a whole checklist**: in the checklist header via *Assign* (only users with access to the
  customer). Assigned checklists appear in *My tasks* and on the project page with an avatar
- **Collapse the sidebar**: icon next to the logo or **Ctrl+\\** (remembered per browser)
- **Sidebar width**: drag its right edge (200–480 px, double-click = default width; remembered per browser)
- **Display** (personal, per browser): checkbox **right or left** and **square or round** (default: right,
  square) in *Settings → Checklists*; the **size** of the checklist (80–160 % in 5 % steps) via **− / +** in the
  toolbar (clicking the percentage resets to 100 %)
- **Section color**: the droplet icon on the right of a heading gives the whole section a background (gray,
  yellow, green, blue, red, purple). The color is stored, applies to all users and is copied into templates;
  a nested colored section takes precedence
- On the right is the **notes column** (also formattable); clicking the colored bar cycles the color
  (gray → green → orange → red → indigo)
- Assignee and due date: on hover via the 👤/📅 icons; "today" and overdue dates are marked in red
- Reorder blocks and checklists via the handle (⋮⋮) by drag & drop (touch works too)
- Formatting is stored as a small HTML subset and sanitized server-side with an allow list
  (`b i u s code a span[data-color] mark[data-bg] br`)
- "From template" shows a preview; the items are **copied** – later template changes do not affect existing
  checklists
- "Save as template" in a checklist header saves it as a new template

## Backup & restore

**Backup via CLI** (consistent even while running, via `VACUUM INTO`):

```bash
docker compose exec projectlist /projectlist backup
# → /data/backups/projectlist-YYYYMMDD-HHMMSS.db (inside the volume)

# copy backups to the host:
docker compose cp projectlist:/data/backups/. ./backups/
```

With a custom target: `/projectlist backup /data/backups/before-update.db`.

**Backup via browser:** as an admin in *Settings → Download backup*
(endpoint `GET /api/admin/backup`).

**Restore:**

The runtime image contains no shell (distroless), so the file is copied into the volume with a short-lived
helper container (backup file in `./backups/` on the host):

```bash
docker compose stop projectlist
docker run --rm -v projectlist_projectlist-data:/data -v "$PWD/backups:/backup:ro" alpine sh -c \
  "cp /backup/projectlist-XXXX.db /data/projectlist.db && rm -f /data/projectlist.db-wal /data/projectlist.db-shm && chown 65532:65532 /data/projectlist.db"
docker compose start projectlist
```

The old `-wal`/`-shm` files must be removed, otherwise SQLite would apply them to the restored file.

## Update

```bash
docker compose exec projectlist /projectlist backup   # back up first
docker compose pull && docker compose up -d
```

Database migrations run automatically on startup (goose, embedded). Data lives in the volume
`projectlist-data` and is kept across updates.

## Development

Requirements: Go 1.27+, Node 26+ – or just Docker (every step also runs in containers).

```bash
# backend tests (in-memory SQLite)
go test ./...
# … or without a local Go installation:
docker build --target test .

# build the image locally and start it with Compose (without GitHub):
docker build -t ghcr.io/seals187/projectlist:dev --build-arg VERSION=dev .
PROJECTLIST_VERSION=dev docker compose up -d

# frontend
cd web && npm install && npm run dev     # Vite dev server, /api → localhost:8080
npm run check                             # svelte-check
npm run build                             # → web/build (embedded into the binary)

# run the server locally
ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=password-123 COOKIE_SECURE=false \
  LOG_FORMAT=text go run ./cmd/projectlist

# changed SQL queries? regenerate the sqlc code:
docker run --rm -v "$PWD:/src" -w /src sqlc/sqlc:1.31.1 generate
```

Add new migrations as `migrations/0000N_name.sql` in goose format.

### Project structure

```
cmd/projectlist/      entry point: serve | backup | migrate | healthcheck | version
internal/config/      environment configuration
internal/db/          open SQLite (WAL, FK, busy_timeout), migrations, transactions
internal/db/queries/  SQL sources for sqlc
internal/db/sqlc/     generated, type-safe query code
internal/auth/        Argon2id, sessions, login limiter, admin bootstrap
internal/authz/       roles & customer assignment
internal/service/     apply a template / save a checklist as template (transactions)
internal/richtext/    allow-list sanitizing of inline formatting
internal/httpapi/     JSON API, middleware, validation
internal/static/      serving the embedded frontend (br/gzip, cache headers)
migrations/           goose migrations (embedded)
web/                  SvelteKit + Tailwind (adapter-static)
```

## Technical details

- **Security:** Argon2id (19 MiB, t=2, OWASP), server-side sessions (only the SHA-256 of the token is stored),
  cookie `HttpOnly; Secure; SameSite=Lax`, CSRF token header + origin check, login rate limit,
  CSP with script hashes, distroless image running as `nonroot`
- **Performance:** the project view loads all checklists with their items in two queries; progress in the sidebar
  and customer views via SQL aggregation; assets pre-compressed at build time (Brotli/gzip), hashed files with
  `Cache-Control: immutable`
- **Operations:** graceful shutdown (SIGTERM), WAL checkpoint on exit, structured logs (`log/slog`),
  `/healthz`, memory limit 128 MB + `GOMEMLIMIT` in the Compose file

## API (overview)

All endpoints under `/api`, JSON. Writing requests need the header `X-CSRF-Token`
(from `POST /api/auth/login` or `GET /api/auth/me`).

| Area        | Endpoints |
|-------------|-----------|
| Auth        | `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`, `PUT /auth/password` |
| Users       | `GET/POST /users`, `PATCH/DELETE /users/{id}`, `PUT /users/{id}/customers` (admin) |
| Customers   | `GET/POST /customers`, `GET/PATCH/DELETE /customers/{id}` |
| Projects    | `GET/POST /customers/{id}/projects`, `GET/PATCH/DELETE /projects/{id}` (incl. `start_date`, `redmine_ticket`), `PUT /projects/{id}/tags` (`tag_ids`), `PUT /projects/{id}/watchers` (`user_ids`), `POST /projects/{id}/duplicate`, `GET /projects/{id}/activity`, `GET/POST /projects/{id}/files` (multipart `file`), `GET/DELETE /files/{id}`, `GET /customers/{id}/assignees` |
| Settings    | `GET /settings/app`, `PUT /settings/app` (`redmine_url`; admin) |
| Tags        | `POST /tags` (members), `PATCH/DELETE /tags/{id}`, `PUT /status-labels/{status}` (`label`, `color`, `wip_limit`; admin); catalog, status names, colors and WIP limits come with `GET /sidebar` |
| Checklists  | `GET/POST /projects/{id}/checklists`, `POST /projects/{id}/checklists/from-template`, `PUT /projects/{id}/checklists/order`, `GET/PATCH/DELETE /checklists/{id}`, `POST /checklists/{id}/save-as-template` |
| Items       | `POST /checklists/{id}/items`, `PUT /checklists/{id}/items/order`, `PATCH/DELETE /items/{id}`, `POST /items/{id}/done` |
| Templates   | `GET/POST /templates`, `GET/PATCH/DELETE /templates/{id}`, `POST /templates/{id}/items`, `PUT /templates/{id}/items/order`, `PATCH/DELETE /template-items/{id}` |
| Sidebar     | `PUT/DELETE /favorites/{projectId}`, `PUT /board/order` (`status`, `ids`), `PUT /prefs/board-hidden` (`customer_ids`), `PUT /prefs/{overview,saved_views}` (any JSON, per user); favorites and settings come with `GET /sidebar` |
| Other       | `GET /sidebar`, `GET /overview`, `GET /my-tasks`, `GET /my-checklists`, `GET /search?q=`, `GET /admin/backup`, `GET /healthz` |

## License

[GNU Affero General Public License v3.0](LICENSE)
