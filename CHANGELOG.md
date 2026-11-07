# Changelog

All notable changes to ProjectList are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Changes until the next release are collected under **Unreleased**.

## [Unreleased]

## [0.1.8] – 2026-11-07

### Changed

- The person responsible for a project gets its checklists: new checklists (also from templates, automations,
  duplicates) are assigned to them; when the person responsible is set or changes, the checklists without a
  person and those of the previous person responsible go to the new one (others stay as they are). Migration
  00029 assigns the existing checklists without a person once

## [0.1.7] – 2026-11-07

### Added

- Reporting (new menu entry): four views over the customers the user may see, with filters period (7 / 30 / 90
  days, 12 months), customer, person and label (saved views possible)
  - The reporting counts checklists (as a whole: done when all items are checked, due at the latest due date of
    its items) and projects – no single items
  - Dashboard: completed / open / overdue checklists (vs. the previous period), projects done, completed vs.
    created checklists per week (line chart with hover), projects by status, due in the next 14 days
  - Customers: projects, progress (completed checklists), open and overdue checklists, next due date; preview
    of the status report and "PDF" (print view, save as PDF in the browser)
  - Workload: open checklists per person (open / due this week / overdue), done in the period, checklists
    without a person
  - Templates: applications, average duration, planned duration, on time, checklists that took longest,
    automations
  - Export: CSV of the current view; "Schedule e-mail" (admins and members): weekly or monthly report e-mail
    via SMTP with the current filters and the recipient's access (admins choose the recipient)
  - Checklists and items now record their creation time (migration 00028); durations and "created" figures
    build up from this version on (checklists from templates: since they were applied)

### Changed

- Project color: project rows in the overview "Cards" and "List", My tasks, Customers and the customer page
  are now tinted completely (like the board cards) instead of showing a stripe

## [0.1.6] – 2026-11-07

### Added

- Project color (personal, set in the project menu – now in every view, also on right-click in the overview
  "List"): besides the board card it shows as a colored stripe on the project row in the overview "Cards" and
  "List", My tasks, Customers (expanded projects) and the customer page; in the board a stripe also appears
  when the shared customer color tints the card

## [0.1.5] – 2026-11-07

### Added

- Overview (cards, board, calendar, timeline): "Delete project" in the project menu (right-click or "…");
  asks for confirmation and warns how many checklists and items are deleted with it
- Card colors in the overview: every user sets their own colors (only visible to them, synced across their
  devices) for customers in "Cards" and "List" (right-click on the customer → "Card color") and for projects in
  "Board" (project menu → "Card color"); the card is tinted (light and dark theme), each of the 13 colors in two
  shades: light and strong (darker in the light theme, lighter in the dark theme). On the customer page (edit
  mode) a shared customer color for all users with the switch "Use in all views" – when on, it replaces the
  personal colors in Cards, List and Board (also for its projects) and also tints the customer under Customers
  (cards and list rows) and My tasks (customer blocks). Migrations 00024 and 00025
- Overview "List": right-click on a customer opens the customer menu (open, new project, filters, card color)
- Right-click menus outside the overview: on the projects of a customer page and of the customers list (same
  project menu as in the overview: rename, status, schedule, responsible, tags, delete …), on customers in the
  customers list, on templates (open, rename, favorite, delete with confirmation) and in My tasks on customers,
  projects and checklists (checklist: open, open project, rename, favorite, save as template, delete with
  confirmation)
- Project and customer menus: "Add to favorites" / "Remove from favorites"
- Template library: the templates page has three views
  - Library: templates grouped by category (colored chips filter a category), items, "used" count and "last
    used", "Use" (dialog: customer → project or new project, checklist title, variables → checklist is created
    and opened) and the template menu (now also "Use" and "Category"). Categories (name, color) are managed by
    admins
  - Variables: `{{name}}` in templates is replaced when applying – system variables (customer, project, due
    date, today, user) and custom variables (admins; asked when applying, default value), also asked in
    "From template" on the project page; values in rich text are escaped
  - Automations (admins): rules "trigger → template" for project created (optional name/customer condition),
    tag added, status changed and schedule (yearly/monthly/weekly into a project); on/off switch, run counter,
    latest runs. Easy Redmine and OTOBO triggers are prepared (integration interface, event fields, customer
    mapping) and will be enabled with the integration
  - System variables have English names (`customer.name`, `customer.location`, `customer.contact`,
    `customer.email`, `project.name`, `project.ticket`, `project.start`, `project.due`, `today`, `user.name`);
    migration 00027 renames the first (German) names in existing templates and rules, which also keep working
  - The view switcher (Library / Automations / Variables) keeps its size when switching views
  - Migration 00026
  - Project page: checklists created by an automation (e.g. after adding a tag) appear right away without
    reloading; the activity shows "Automation" as the author

### Changed

- Customers, list view: clicking the customer icon also expands and collapses the projects (like the arrow);
  name and location still open the customer
- New projects always start with the status "Planning", wherever they are created (customer page, dialog "New
  project", board column, duplicate); before, the customer page created them as "Active" and the board used the
  status of the column

## [0.1.4] – 2026-11-07

### Added

- Overview "Timeline": new "60 days" view (starting today) between "30 days" and "Months"

## [0.1.3] – 2026-11-07

### Added

- Appearance → Background: "Darken content" with the levels Off / Light / Medium / Strong. Darkens everything in
  the main area (Overview, My tasks, Customers, Templates, checklists, settings …) – with or without a background
  image; the sidebar and the back bar stay unchanged, menus and dialogs are shown above it. Applies to all devices
  of the user

## [0.1.2] – 2026-11-07

### Added

- Overview "Calendar": calendar weeks (ISO) in a narrow first column "CW"

### Changed

- Overview "Calendar": the current month stands out more clearly – days of the neighboring months are greyer in
  the light theme and darker in the dark theme, days of the month slightly lighter in the dark theme; day numbers
  of the month bolder. Also applies with a background image
- Overview "Board": project names on the cards slightly smaller (13 px), the same size collapsed and expanded
- Overview "Calendar": fits completely into the window again with the back bar and calendar weeks (no scrolling)
- Back bar: the history shows the icon and name of the overview view that was open (e.g. "Overview · Calendar"),
  the path only shows "Overview" with the view icon; switching the view updates the entry

## [0.1.1] – 2026-11-07

### Added

- Consistent type icons and colors for customers (green, building), projects (blue, folder), checklists (violet,
  list) and templates (orange, document) – in the sidebar, search (Ctrl K), breadcrumbs, page headers, overview
  (cards, list, board), customer list, My tasks, activity log, template library and template picker; separate
  colors in the dark theme
- Favorites redesigned: two-line entries with type icon, name and "type · context" (project → customer,
  checklist → project, customer → number of projects, template → number of items); custom groups with bookmark
  icon and count, collapsible (saved per user); right-click or long press: Open, Open in new tab, Move to group
  (also to a new group), Remove from favorites; rename and delete groups (entries move to "No group"); drag & drop
  within and between groups
- Templates can be saved as favorites (star in the header of the template page)
- Back bar above every page: back and forward plus the path of the current page with type icons (e.g. Overview /
  Customers / Customer / Project / Checklist, each clickable; the overview as start page always comes first).
  A long press on ← or Alt+← shows the history ("Back to" with type or "page"), also usable with the arrow keys
  and Enter, with a trash button to clear the history; the history shows all visited pages, including the main
  pages and the settings with their sub page (e.g. "Settings · Appearance"); it survives reloading the tab.
  The previous path information in the page headers is gone
- Checklists as tabs: every opened checklist appears as a tab above the checklist (type icon, name, progress, red
  dot for overdue items). Click switches, ✕ or the middle mouse button closes (then the neighboring tab), "+"
  searches and opens another checklist, tabs can be reordered via drag & drop; the tabs are stored per user in the
  browser (max. 12)
- Customers: phone numbers (several, each with a label such as "Main" or "Mobile") on the customer page; add,
  change and remove them in edit mode, otherwise shown as clickable phone links (migration 00023)
- Settings → "About" (at the bottom of the navigation): current version, link to the GitHub repository and to the
  release notes

### Changed

- Favorites are only set via the star in the header of customer, project, checklist and template (no longer on
  board cards and in the context menus); the favorites list no longer shows a star
- "Customers" in the sidebar and in the header of the customer list: building icon (uncolored) like the customer
  type icon
- Customers show the customer icon instead of the short code ("AB") everywhere: customer page header, overview
  (cards, list, board, calendar, timeline), My tasks, customer list (list and cards), customer filter, context
  menus, project page and user management
- Checklists: grid lines considerably darker (like cell borders in Excel), also towards the notes column
- A second click on your own name at the bottom left closes the settings again and returns to the page that was
  open before
- Overview "Timeline": views "This week" (Mon–Sun), "14 days" and "30 days" (starting today) and "Months"; the
  days use the full width, the arrows move by one view length, the selected view is remembered (per browser).
  Bars starting before the visible range show their name at the left edge
- Project page: "Latest activity" only shows the newest entry and can be expanded (then the last five); the
  "Easy Redmine" field was removed from the properties (a stored ticket number stays in the database)

### Fixed

- Overview: the boxes (cards, list, board columns) were rounder than on My tasks, Customers and Templates; all now
  have the same corners
- With a background image and the glass effect, the colored type icons in the sidebar and bars were too strong;
  they are now softly tinted there
- Dark theme with a background image: checklists, template editor and settings were slightly translucent; their
  content area is now opaque
- Input fields in menus (e.g. "New group …" in the favorites) did not get focus when opened
- Background image with the glass effect turned off: the sidebar, back bar and header band of customer and project
  pages were dark even in the light theme; they now look like without a background image (light or dark)
- Header band (Overview, My tasks, Customers, Templates): with a font size other than 100 %, the bands with quick
  filter chips (My tasks, Customers) were slightly taller than the others; all now have the same height
- Overview (and My tasks, Customers, Templates): after switching back to a default setting (e.g. from "Timeline"
  back to "Cards"), the old setting stayed in the address; reloading the page therefore jumped back to the
  previous view
- Checklists with grid lines: the vertical line towards the notes column was 1 px further right on headings
  (H1–H4) than on rows with a checkbox
- Groups and order of the favorites are now stored in the database (migration 00022); existing folders are taken
  over automatically
- Checklists and templates: the controls of a row (left "+" and drag handle, right status, person, due date,
  color) have considerably more contrast in the light and dark theme; the handle is slightly larger

## [0.1.0] – 2026-11-06

### Added

- Restore a backup directly in the app: *Settings → Backup & Restore → Restore*. The file is checked first
  (SQLite, integrity, ProjectList database, not newer than the running version, at least one admin); then the app
  stores a safety copy of the current state in the data directory (`pre-restore-<time>.db`) and applies the backup
  without a restart. Older backups are migrated automatically. The admin stays signed in if they are an admin in
  the backup, otherwise they sign in again
- Saved views: folders and order like the favorites – move via drag & drop, create, rename and delete folders via
  right-click and move views into folders; applies to all devices

### Changed

- Settings: "Backup" is now called "Backup & Restore"

## [0.0.9] – 2026-11-06

### Added

- Settings rebuilt: on the left a dedicated navigation with search ("Search settings …") and the groups "Personal"
  (Profile & account, Appearance, Font, Checklists, Overview) and "Workspace" (Tags, Users & permissions, Backup;
  admins only), on the right the page with cards and rows
- Appearance: color scheme "System" (follows the operating system), accent color (violet, blue, teal, green,
  orange, red, pink), glass effect on/off, darken background image, row spacing (compact/normal/airy) and font
  size of the whole app (80–120 %). Language, color scheme and appearance now apply to all devices of the user
- Font & checklists with a live preview of a checklist
- Favorites: besides projects, customers and checklists can now be saved as favorites too (star next to the title
  of the customer, project or checklist page, for customers and projects also in the context menu). In the
  sidebar they are listed in the order they were added (checklists with their name only). Existing project
  favorites are kept
- Arrange favorites: move them up or down via drag & drop; create folders via right-click in the favorites list
  (rename via right-click or double-click, expand and collapse, delete – the favorites are kept). Favorites can be
  dragged into folders or moved via right-click "Move to folder", folders can be moved as well. Order and folders
  apply to all devices of the user
- Tags: filter, "+ Tag", menu "…" per tag with edit (name and color) and delete
- Users & permissions: search, role filter, column "Last active", role editable directly in the table, detail
  pane with role, customer access ("Select all"), "Reset password" and "Remove user"
- Invite users: admins invite by email; the link (valid for 7 days) leads to a page to set the password. "Reset
  password" also sends a link (valid for 24 hours). Without configured email sending, the app shows the link to
  copy. New environment variables `APP_URL` and `SMTP_*` (see README)

### Changed

- Users are no longer created with a password set by the admin, but invited
- Overview "Board": collapsed cards are more compact (less spacing between and inside the cards, project name
  slightly smaller); layout and design stay the same
- Checklists: the status field (e.g. "Done", "Terminée", "Erledigt") is placed directly right of the checkbox and is
  visible as "Status" even when empty; before, it was invisible without content until hovering the row. With the
  checkbox on the left, it sits between checkbox and text
- The size of the checklist view (100 % …) is only set in the checklist toolbar; the settings offer the font size
  of the whole app instead

## [0.0.8] – 2026-11-06

### Added

- Consistent header for Overview, My tasks, Customers and Templates (`PageHeader`): grey header card with page
  icon, title, subtitle, view switcher and main action; below it a strip with search, filters, quick filters,
  "Save view", sorting and "Collapse / expand all". With little space, the filters are combined into
  "Filter (n)", then the view switcher only shows icons
- New: search in Overview, My tasks and Templates; sorting in Overview, My tasks and Templates; filters
  "Customer" and "Due date" in My tasks, filter "Status" for customers
- View, filters and sorting are part of the URL (shareable links) and are stored per page and user; "Save view"
  now works on all four pages
- Frontend tests with Vitest (run as part of the Docker build)
- Overview "List" redesigned: one card per customer with short code, name, summary ("3 projects · 5/51 steps ·
  46 open · next 13.11.") and progress ring; project rows with status, progress, steps, due date and assignee,
  whole row clickable. In a wide window steps as a separate column and tags next to the name, otherwise steps next
  to the percentage and tags below; tags are shortened to the available width ("+N")
- "Display" button in the header of all pages: toggles per page and view (e.g. steps, progress in %, all tags,
  due date, assignee), stored per user on the server (fallback: browser)

### Changed

- Content (cards, lists, tiles) is now opaque white or dark grey with a background image; only the page
  background and the header card stay colored/translucent
- Default status colors (without a custom color): In progress green, Acceptance orange, Done dark green
- German: "Zuständig" instead of "Verantwortlich" (filters, columns, project)
- Overview "Calendar": fits into the window including the list "Without due date" without scrolling (week rows
  share the height, full days scroll inside the cell); more compact day numbers and entries
- Overview "Cards": project rows rearranged – name with folder icon and status pill on the right, below it the
  labels (as many as fit, then "+N"), below that the progress bar with percentage and due date (dd.mm.)
- Clicking "Overview" in the sidebar resets the filters (customer, label, responsible) to "All"; saved views still
  apply their own filters
- Overview: "Group by" in the board is now a dropdown in the same style as the customer, label and responsible
  filters (same button, same menu); highlighted only when not grouped by status
- Status badges (Overview "List", Customers, Project, My tasks) have the same style for every status: background in
  the board color of the status, text always black (in both themes)
- Customer list: with expanded projects, the project status is shown in the "Status" column (instead of under
  "Projects"), the percentage next to the progress bar
- Only two themes left: Light and Dark ("Dark grey" becomes Dark; without a choice, the system setting decides
  once). The button in the sidebar switches directly between Light and Dark
- Dropdown and context menus (e.g. Customer, Label, Responsible, Group by) are no longer dark with a background
  image in the light theme, but follow the selected theme
- With a background image, the tiles in Overview and "My tasks" have the same brightness as in Customers and
  Templates
- With a background image, the sidebar and header band are now dark (almost opaque) in the dark theme instead of
  semi-transparent as in the light theme
- Dropdown and context menus are no longer pure white in the light theme, but slightly translucent (frosted glass
  like the tiles); slightly translucent in the dark theme as well
- With a background image, the content block of checklists, template editor and settings has the same tint as the
  sidebar in the dark theme (before: opaque black)
- Dark theme without a background image: the content area (Overview, My tasks, Settings etc.) has the same color as
  the sidebar instead of almost black
- Native select lists are readable when open, also with a background image and in the dark theme
- Favorites in the sidebar no longer show the customer short code on the right

## [0.0.7] – 2026-11-05

### Added

- Checklists and templates: the width of the notes column can be dragged left or right at its left edge
  (double-click: default width, also via arrow keys); the width is saved

### Changed

- Consistent colors for selected and hover: sidebar, view switcher (Cards/Board/List/…, Customers Cards/List),
  tabs in "My tasks", timeline zoom, settings and filter chips. Selected is always a violet background with violet
  text, hover always a neutral grey area; filter chips are the same everywhere (round). Sidebar search field
  lighter in the dark theme and on backgrounds
- Backgrounds: filters, search field, chips and buttons in the header band are light glass instead of dark boxes;
  the header band always has the same height on Overview, My tasks, Customers and Templates; "New template" sits on
  a glass surface
- Backgrounds: board columns, lists, tables and tiles less transparent again, so they stay readable on busy images

### Fixed

- Some light accent colors were missing in the dark theme (e.g. "Group" in the board was hardly readable)

## [0.0.6] – 2026-11-05

### Added

- Backgrounds: 25 gradients or custom images (JPG, PNG, WebP, GIF; max. 5 MB, larger images are downscaled
  automatically before saving). Selectable per user for the whole app (settings, the same on all devices), per
  customer (button on the customer page) and per project ("…" menu of the project page); the most specific one
  applies: project before customer before user. Uploaded images are available to everyone; the uploader and admins
  can delete them. Displayed like in Planka: the image lies behind the whole app, the sidebar and the narrow header
  band (title, views, filters, tabs) are translucent dark glass, board columns, lists, tables, calendar, timeline
  and the tiles in them translucent light glass; in between, the image is fully visible. Checklist editor, template
  editor and settings sit on an opaque surface. Migration 00019

### Changed

- My tasks: projects you are responsible for now appear as well (marked "Responsible"), even if no checklist and
  no item in them is assigned to you
- Project page: checklists can be assigned and deleted (with confirmation) directly in the list, without opening
  them
- Customer page redesigned: header with short code, name, "Edit" and "+ Project", location/contact/email (email as
  link; editing via the button), notes, projects as a table with status (custom name and color as in the board),
  labels, progress, due date and responsible person
- Overview, List: column "Assignee" instead of "Team / Assignee"; the project row shows the responsible person
  instead of the team, the checklist rows show the assigned person as before
- Overview, Board: "Collapse all" / "Expand all" instead of the "Compact" toggle (collapsed cards only show
  customer, project name and progress)
- Overview, Cards: project name across the full width (up to two lines), below it status, progress bar, number of
  completed of all items (e.g. "5/10") and due date; the status color appears only once
- Saved views can be renamed (pencil or double-click in the sidebar)
- Checklists: completed (struck-through) items are darker and therefore more readable

### Fixed

- Sidebar: removing/renaming favorites and saved views was not reachable on devices without a mouse (tablet,
  phone); the buttons are now always visible there
- Select lists (e.g. role when creating a user) and date fields partly showed invisible entries when the theme of
  the app and of the operating system differed (e.g. app light, Windows dark). The browser controls now follow the
  app theme

## [0.0.5] – 2026-11-04

### Added

- Overview: new views **Calendar** (projects on their due date, reschedule via drag & drop) and **Timeline** (Gantt:
  bars from start to due date, dragging moves, dragging the ends changes start/due date; zoom weeks/months);
  consistent view switcher Cards · Board · List · Calendar · Timeline
- Start date per project (project page, context menu "Schedule", timeline); without a start date, the creation
  date applies
- Filter bar for all overview views: customer (with "All"/"None"), label and responsible as dropdowns; in the board
  additionally "Group by" status, customer or responsible and "Compact". The filters are stored per user in the
  database and are the same on all devices
- Saved views: save the current view with filters under a name; they appear in the sidebar
- Board: calmer cards (labels as color dots, max. two plus "+N"; progress bar; due date, number of checklists,
  responsible person and team at the bottom), quick actions on hover (responsible, schedule, favorite, menu),
  "Add project" at the end of each column
- Board: WIP limit per status column (e.g. "WIP 2/2", red once reached)
- Status (Planning, Active, Review, Blocked, Done) renamable for everyone, with a custom color and WIP limit:
  directly in the board by clicking the column title (admins only; can be reset to the default)
- Tags (labels) for projects with a freely selectable color (54 predefined colors or a custom one): assign in the
  project or create directly, visible in all overview views and in the customer's project list; management
  (rename, recolor, delete) under Settings → Projects
- Board and cards of the overview: right-clicking a project opens a menu to open it, for checklists (open or
  create), rename, status, schedule, responsible person, tags and favorite; on touch devices via the "…" button of
  the card. Right-clicking a customer: open customer, new project, show only this customer, hide
- Board: cards can also be moved within a column (the order is saved and applies to all users)
- Responsible person per project (project page, board card and context menu)
- New project page: header with path, status, favorite and actions (Duplicate, Archive, Delete, "+ Checklist"),
  progress in plain text ("37 % · 19 of 31 items · 50 days left"), tabs Checklists · Activity · Files · Notes and on
  the right a properties panel (status, customer, responsible, start/end, labels, watchers, Easy Redmine) with
  editing directly in the panel plus "Latest activity"
- Activity log per project: completed items (grouped), completed/created/deleted checklists, status, schedule,
  responsible, labels, Redmine ticket and files; already completed items appear retroactively
- Upload files per project (also via drag & drop, max. 25 MB per file), download and delete them; the files are
  stored in the database and are therefore included in every backup
- Duplicate project: copy with description, responsible, labels and all checklists (items not completed, without
  due dates), status Planning
- Watchers per project (people with access to the customer)
- Easy Redmine ticket per project; an admin sets the Redmine address under Settings → Projects, the link is built
  from it
- Overview cards view redesigned: one card per customer with location, progress ring, projects (status, progress,
  due date) and footer (overdue or open items, next due date, team); customers with overdue items come first and
  have a red border; empty customers with "+ Project from template". Cards can be collapsed individually or with
  "Collapse/expand all"; collapsed, the header and footer remain as a summary
- Overview list view redesigned: key figures (active projects, open/overdue items, due in 14 days, behind
  schedule, blocked), per project the checklists as sub rows (status "not started/running/n overdue/done",
  progress, due date, assignee), column **Time vs. progress** (progress bar with a marker for the elapsed time,
  "on track" / "x % behind" / "overdue") and "Updated" (latest activity)

### Changed

- Sidebar: collapsible "Customers" section removed (customers via the "Customers" menu item)
- Sidebar: "Recently visited" removed (incl. the server endpoint `/api/recent` and the table `user_recent`,
  migration 00017); "Saved views" instead
- Long status names are no longer cut off in the board columns but wrapped
- README in English
- Overview, My tasks, Customers and Templates have the same page header (title, subtitle, actions on the right) and
  the same spacing; "My tasks" now uses the full width like the other pages
- Server: read timeout 3 minutes instead of 30 seconds, so file uploads also work over slow connections
- Database migrations 00012–00018; they are applied automatically on startup (make a backup first)

## [0.0.4] – 2026-11-04

### Added

- A GitHub release is created automatically for every version tag, with the notes from this file (workflow
  `release.yml`; optional announcement on Discord via the secret `DISCORD_WEBHOOK`)

### Changed

- GitHub Actions on current versions (no more Node 20 warnings)
- Test data and code examples only use neutral example names

### Fixed

- When started manually, the workflows accept the tag with or without "v" (`0.0.3` as well as `v0.0.3`)

## [0.0.3] – 2026-11-04

### Added

- Image `:edge` with the current development state, on every push to `main` after green tests

### Changed

- `docker-compose.yml` only uses the published image `ghcr.io/seals187/projectlist`, without `build`/`args`;
  version via `PROJECTLIST_VERSION` in the `.env` (`latest` = newest release, `edge` = development state, or e.g.
  `0.0.2`)
- All code comments in English (UI texts unchanged)

### Fixed

- Short descriptions of the fonts in the settings now appear in the selected language (were always German)
- "Recently visited": visits within the same millisecond were sometimes shown in the wrong order; this also made
  the test on GitHub Actions fail randomly

## [0.0.2] – 2026-11-04

### Added

- `CHANGELOG.md` with the version history
- `.gitattributes`: consistent LF line endings (also on Windows), images and fonts as binary files
- Container image on GitHub (`ghcr.io/seals187/projectlist`) for linux/amd64 and linux/arm64, automatically on
  every release tag (`:X.Y.Z`, `:X.Y`, `:latest`); existing tags can be published afterwards manually
- GitHub Actions: tests (go vet, go test, frontend build) on every push to `main` and on pull requests

### Changed

- The Dockerfile builds frontend and Go natively and cross-compiles for the target architecture (fast multi-arch
  builds without emulation); OCI labels (source, license, version) link the image to the repository

## [0.0.1] – 2026-11-04

First version.

### Added

**Basics**

- Self-hosted project management for multiple users: customers → projects → checklists, templates as copy
  templates
- A single Go binary (net/http, SQLite, sqlc, goose) with an embedded SvelteKit frontend; distroless Docker image
  (~29 MB, ~6 MB RAM in operation), Compose with a 128 MB limit and health check
- Users with the roles Admin, Member and Viewer plus customer assignment; Argon2id, sessions, CSRF and origin
  protection, strict Content Security Policy
- Backup via command (`projectlist backup`) and as a download for admins
- UI in English, German and French
- Database migrations 00001–00011

**Overview**

- Views Cards, List (with key figures) and Board (status columns, drag & drop, customer filter); default view
  selectable in the settings, the view in use is kept for the session
- Project status Planning, Active, Review, Blocked, Done and Archived; due date, team (assigned people) and next
  open step per project
- Projects collapsible per customer (default: collapsed)
- "New project" dialog with customer, status, due date and optionally a first checklist from a template

**Sidebar**

- Search (Ctrl+K) and "+" for a new project
- Favorites and "Recently visited" (customers, projects, checklists, templates), stored per user in the database
  and therefore the same on all devices
- Customer tree with short code, project count and status dots; customers and projects expand/collapse on click
- Collapsible sections, resizable sidebar width, sidebar fully collapsible (Ctrl+\)
- At the bottom the user (→ settings), theme toggle and sign out; optional subtitle via `APP_SUBTITLE`

**Customers**

- Customer overview as a list (expandable with projects) or cards; search, filters (all, with active projects,
  critical, without project) and sorting (latest activity, name, progress, due date, open tasks)
- Status per customer: New, On track, Critical, Completed
- Location, contact person and email per customer
- Configurable per project whether it counts towards the customer's overall progress; excluded projects are shown
  in grey, the customer shows "without N project(s)"

**My tasks**

- Tab "Customers": customers → projects → checklists you are involved in, with due date
  (Yesterday/Today/Tomorrow)
- Tab "Personal": assigned items to check off
- Filters Open, Today and Done (last 30 days)

**Checklist editor**

- Blocks: headings H1–H4 (collapsible), text, bulleted list, numbered list, quote, code block with syntax
  highlighting, divider and items with a checkbox
- Items with assignees, due date, status text (e.g. "Done") and notes; convert a block into an item with one click
- Section and row colors in three levels (like Word/Excel), text and highlight colors,
  bold/italic/underline/strikethrough, links with text and URL
- Grid lines (on/off), zoom 80–160 %, undo/redo, drag & drop
- Whole checklists assignable to a person; save a checklist as a template

**Appearance**

- Themes Light, Dark, Dark grey and System
- 12 embedded fonts (incl. Inter, Geist, Mona Sans), headings in serif or like the body text, code in
  JetBrains Mono; no external requests
- Checkbox left or right, square or round; code block background light grey, dark grey or black

[Unreleased]: https://github.com/seals187/projectList/compare/v0.1.8...HEAD
[0.1.8]: https://github.com/seals187/projectList/compare/v0.1.7...v0.1.8
[0.1.7]: https://github.com/seals187/projectList/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/seals187/projectList/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/seals187/projectList/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/seals187/projectList/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/seals187/projectList/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/seals187/projectList/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/seals187/projectList/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/seals187/projectList/compare/v0.0.9...v0.1.0
[0.0.9]: https://github.com/seals187/projectList/compare/v0.0.8...v0.0.9
[0.0.8]: https://github.com/seals187/projectList/compare/v0.0.7...v0.0.8
[0.0.7]: https://github.com/seals187/projectList/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/seals187/projectList/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/seals187/projectList/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/seals187/projectList/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/seals187/projectList/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/seals187/projectList/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/seals187/projectList/releases/tag/v0.0.1
