export type Role = 'admin' | 'member' | 'viewer';
export type ProjectStatus = 'planning' | 'active' | 'review' | 'blocked' | 'done' | 'archived';
export type Kind = 'item' | 'h1' | 'h2' | 'h3' | 'h4' | 'text' | 'bullet' | 'numbered' | 'quote' | 'code' | 'divider';
export type NoteColor = '' | 'green' | 'orange' | 'red' | 'indigo';
// '' = same as section, 'none' = explicitly none, otherwise values from $lib/palette (e.g. "yellow-strong")
export type BgColor = string;

export interface User {
  id: number;
  email: string;
  name: string;
  role: Role;
}

/** Favorite in the sidebar (per user). */
export type FavoriteKind = 'customer' | 'project' | 'checklist' | 'template';
export interface Favorite {
  kind: FavoriteKind;
  id: number;
  group_id: number | null; // favorites group (null = without group)
  position: number; // order within the group / the ungrouped favorites
}
/** User-defined group of favorites (sidebar). */
export interface FavoriteGroup {
  id: number;
  name: string;
  position: number;
  collapsed: boolean;
}

export interface AdminUser extends User {
  created_at: string;
  customer_ids: number[];
  last_seen_at: string | null; // last request or sign-in (written at most every 5 min)
  invited: boolean; // invitation not accepted yet
}

/** Phone number of a customer ("Zentrale", "+41 41 123 45 67"). */
export interface CustomerPhone {
  label: string;
  number: string;
}

export interface Customer {
  id: number;
  name: string;
  notes: string;
  location: string;
  contact_name: string;
  contact_email: string;
  phones: CustomerPhone[]; // phone numbers (label optional)
  created_at: string;
}

/** /api/customers: customer with last activity. */
export interface CustomerListItem extends Customer {
  last_activity: string;
}

export interface Project {
  id: number;
  customer_id: number;
  name: string;
  description: string;
  status: ProjectStatus;
  start_date: string | null; // YYYY-MM-DD (timeline); null = creation date
  due_date: string | null;
  include_in_customer_progress: boolean;
  owner_id: number | null; // person responsible
  redmine_ticket: string; // Easy Redmine ticket (number or URL), "" = none
  created_at: string;
}

export interface ProjectWithProgress extends Project {
  total: number;
  done: number;
}

/** Common fields of checklist and template items. */
export interface Row {
  id: number;
  kind: Kind;
  title: string;
  description: string;
  note_color: NoteColor;
  bg_color?: BgColor;
  indent: number;
  lang: string;
  position: number;
}

export interface Item extends Row {
  checklist_id: number;
  assignee_id: number | null;
  assignee_name: string | null;
  due_date: string | null;
  done_at: string | null;
  done_by: number | null;
  done_by_name: string | null;
  status: string; // free text, e.g. "Erfüllt"
}

export interface TemplateItem extends Row {
  template_id: number;
}

export interface Checklist {
  id: number;
  project_id: number;
  title: string;
  template_id: number | null;
  position: number;
  total: number;
  done: number;
  assignee_id: number | null;
  assignee_name: string | null;
  items: Item[];
}

export interface ChecklistSummary {
  id: number;
  project_id: number;
  title: string;
  template_id: number | null;
  position: number;
  total: number;
  done: number;
  due_date: string;
  assignee_id: number | null;
  assignee_name: string; // empty = nobody
}

export interface Assignee {
  id: number;
  name: string;
  email: string;
}

export interface ChecklistDetail {
  checklist: Checklist;
  project: { id: number; name: string; status: ProjectStatus };
  customer: { id: number; name: string };
  assignees: Assignee[];
}

export interface ProjectDetail {
  project: Project;
  customer: { id: number; name: string };
  checklists: ChecklistSummary[];
  tables: TableSummary[];
  progress: { total: number; done: number };
  tag_ids: number[];
  assignees: Assignee[]; // possible persons responsible
  watchers: { id: number; name: string }[];
  files_count: number;
  redmine_url: string; // Easy Redmine base URL (admin setting), "" = not set
}

/** Project tag with a free color (#rrggbb). */
export interface Tag {
  id: number;
  name: string;
  color: string;
}

export interface Template {
  id: number;
  name: string;
  description: string;
  category_id?: number | null;
  kind?: TemplateKind; // missing = checklist
  sheet_data?: string; // table templates: the workbook (JSON, see $lib/sheetData)
}

/** A template creates a checklist (items) or a table (workbook). */
export type TemplateKind = 'checklist' | 'table';

export interface TemplateSummary extends Template {
  item_count: number;
  used_count?: number; // applications (manually or by automations)
  last_used?: string; // RFC 3339, "" = never / unknown
}

/** Category of the template library (admins manage them). */
export interface TemplateCategory {
  id: number;
  name: string;
  color: string; // card color name, "" = gray
  position: number;
}

/** Variable for {{name}} in templates. source: "system", "custom", an integration or "unknown". */
export interface TemplateVariable {
  id?: number;
  name: string;
  source: string;
  description: string;
  desc_key?: string; // system: i18n key
  default_value: string;
  example?: string;
  used_in?: number;
  templates?: number[];
}

export interface AutomationCondition {
  field: string;
  op: 'equals' | 'contains';
  value: string;
}

export interface AutomationSchedule {
  freq: 'yearly' | 'monthly' | 'weekly';
  month?: number;
  day?: number;
  weekday?: number; // 1 = Monday … 7 = Sunday
}

export interface AutomationRule {
  id: number;
  trigger_kind: string;
  source: string;
  conditions: AutomationCondition[];
  schedule: AutomationSchedule | null;
  template_id: number;
  target_project_id: number | null;
  checklist_title: string;
  enabled: boolean;
  run_count: number;
  last_run_on: string;
  template_name: string;
  target_name: string;
}

export interface AutomationRun {
  id: number;
  rule_id: number | null;
  project_id: number | null;
  checklist_id: number | null;
  status: 'ok' | 'warning' | 'error';
  code: string;
  template_name: string;
  target: string;
  detail: string;
  created_at: string;
}

export interface SidebarChecklist {
  id: number;
  title: string;
  total: number;
  done: number;
  next_due: string; // next due date of open items, "" = none
}

export interface SidebarProject {
  id: number;
  name: string;
  status: ProjectStatus;
  start_date: string | null; // YYYY-MM-DD, null = created_at
  due_date: string | null;
  created_at: string; // YYYY-MM-DD
  include_in_customer_progress: boolean;
  tag_ids: number[];
  owner_id: number | null;
  owner_name: string; // empty = nobody
  board_position: number; // order within the board column (shared)
  background: string; // "", "g<n>" or "img:<id>" (inherits customer, then user)
  total: number;
  done: number;
  checklists: SidebarChecklist[];
  tables: SidebarTable[];
}

/** Table of a project in the sidebar data (breadcrumb, project tab). */
export interface SidebarTable {
  id: number;
  title: string;
}

export interface SidebarCustomer {
  id: number;
  name: string;
  location: string; // e.g. town, "" = none
  background: string; // "", "g<n>" or "img:<id>"
  color: string; // shared customer color; with color_everywhere it wins in all views (all users)
  color_everywhere: boolean;
  projects: SidebarProject[];
}

export interface Task {
  id: number;
  title: string;
  description: string;
  note_color: NoteColor;
  due_date: string | null;
  done_at: string | null;
  checklist_id: number;
  checklist_title: string;
  project_id: number;
  project_name: string;
  customer_id: number;
  customer_name: string;
}

export interface SearchResult {
  customers: { id: number; name: string }[];
  projects: { id: number; name: string; customer_id: number; customer_name: string }[];
  checklists: { id: number; title: string; project_id: number; project_name: string; customer_id: number }[];
}

/** API operations the list editor uses for checklists or templates. */
export interface Ops<R> {
  create: (body: Record<string, unknown>) => Promise<R>;
  patch: (id: number, body: Record<string, unknown>) => Promise<R>;
  remove: (id: number) => Promise<void>;
  reorder: (ids: number[]) => Promise<void>;
  toggle?: (id: number, done: boolean) => Promise<R>;
}

/** Interface between editor and toolbar. */
export interface EditorApi {
  readonly canUndo: boolean;
  readonly canRedo: boolean;
  readonly kind: Kind | null;
  readonly marks: { bold: boolean; italic: boolean; underline: boolean; strike: boolean; code: boolean };
  undo(): void;
  redo(): void;
  setKind(kind: Kind): void;
  format(cmd: "bold" | "italic" | "underline" | "strikeThrough" | "code"): void;
  color(c: string): void;
  highlight(c: string): void;
  saveSelection(): void;
  openLink(): void;
  clear(): void;
  add(kind: Kind): void;
}

/** Checklist assigned to a person (GET /my-checklists). */
export interface MyChecklist {
  id: number;
  title: string;
  project_id: number;
  project_name: string;
  customer_id: number;
  customer_name: string;
  total: number;
  done: number;
  due_date: string;
}

/** /api/overview: extras per project ID for the overview. */
export interface OverviewExtra {
  team: { id: number; name: string }[];
  next_step: string; // rich-text HTML
  updated_at: string | null; // latest activity
  checklists: Record<string, { assignee_id: number | null; assignee_name: string; overdue: number }>;
}

/** Entry of a project's activity log (newest first). */
export interface ActivityEntry {
  id: number;
  user_id: number | null;
  user_name: string;
  kind: string;
  subject: string;
  detail: string;
  created_at: string;
}

/** File attached to a project. */
export interface ProjectFile {
  id: number;
  name: string;
  mime: string;
  size: number;
  uploaded_by: number | null;
  uploaded_by_name: string;
  created_at: string;
}

// ---- Tables (spreadsheets) -------------------------------------------------------

/** Table of a project in lists (without the workbook). */
export interface TableSummary {
  id: number;
  project_id: number;
  title: string;
  template_id: number | null;
  position: number;
  version: number;
  updated_at: string; // RFC 3339
  sheets: number;
  updated_by_name: string;
}

/** Table with its workbook (sheets as Fortune-sheet stores them, see $lib/sheetData). */
export interface Table extends Omit<TableSummary, 'sheets'> {
  data: import('./sheetData').StoredSheet[];
}

export interface TableDetail {
  table: Table;
  project: { id: number; name: string; status: ProjectStatus };
  customer: { id: number; name: string };
}
