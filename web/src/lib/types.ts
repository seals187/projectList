export type Role = 'admin' | 'member' | 'viewer';
export type ProjectStatus = 'planning' | 'active' | 'review' | 'blocked' | 'done' | 'archived';
export type Kind = 'item' | 'h1' | 'h2' | 'h3' | 'h4' | 'text' | 'bullet' | 'numbered' | 'quote' | 'code' | 'divider';
export type NoteColor = '' | 'green' | 'orange' | 'red' | 'indigo';
// '' = wie Abschnitt, 'none' = ausdrücklich ohne, sonst Werte aus $lib/palette (z. B. "yellow-strong")
export type BgColor = string;

export interface User {
  id: number;
  email: string;
  name: string;
  role: Role;
}

export interface AdminUser extends User {
  created_at: string;
  customer_ids: number[];
}

export interface Customer {
  id: number;
  name: string;
  notes: string;
  location: string;
  contact_name: string;
  contact_email: string;
  created_at: string;
}

/** /api/customers: Kunde mit letzter Aktivität. */
export interface CustomerListItem extends Customer {
  last_activity: string;
}

export interface Project {
  id: number;
  customer_id: number;
  name: string;
  description: string;
  status: ProjectStatus;
  due_date: string | null;
  include_in_customer_progress: boolean;
  created_at: string;
}

export interface ProjectWithProgress extends Project {
  total: number;
  done: number;
}

/** Gemeinsame Felder von Checklisten- und Template-Einträgen. */
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
  status: string; // Freitext, z. B. "Erfüllt"
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
  assignee_name: string; // leer = niemand
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
  progress: { total: number; done: number };
}

export interface Template {
  id: number;
  name: string;
  description: string;
}

export interface TemplateSummary extends Template {
  item_count: number;
}

export interface SidebarChecklist {
  id: number;
  title: string;
  total: number;
  done: number;
  next_due: string; // nächste Fälligkeit offener Einträge, "" = keine
}

export interface SidebarProject {
  id: number;
  name: string;
  status: ProjectStatus;
  due_date: string | null;
  include_in_customer_progress: boolean;
  total: number;
  done: number;
  checklists: SidebarChecklist[];
}

export interface SidebarCustomer {
  id: number;
  name: string;
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

/** API-Operationen, die der Listen-Editor für Checklisten bzw. Templates nutzt. */
export interface Ops<R> {
  create: (body: Record<string, unknown>) => Promise<R>;
  patch: (id: number, body: Record<string, unknown>) => Promise<R>;
  remove: (id: number) => Promise<void>;
  reorder: (ids: number[]) => Promise<void>;
  toggle?: (id: number, done: boolean) => Promise<R>;
}

/** Schnittstelle zwischen Editor und Toolbar. */
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

/** Einer Person zugewiesene Checkliste (GET /my-checklists). */
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

/** /api/overview: Ergänzungen je Projekt-ID für die Übersicht. */
export interface OverviewExtra {
  team: { id: number; name: string }[];
  next_step: string; // Rich-Text-HTML
}

/** Zuletzt besuchte Seite (Seitenleiste). */
export interface RecentEntry {
  kind: 'customer' | 'project' | 'checklist' | 'template';
  id: number;
}

/** Vom Server aufgelöster Eintrag: Titel und übergeordnetes Element (Kunde bzw. Projekt). */
export interface RecentItem extends RecentEntry {
  title: string;
  parent: string;
}
