// Very small syntax highlighter (regex-based) for the usual admin
// languages. The result is escaped HTML with <span class="tk-…"> tokens.

import { escapeHtml } from './rich';

export const CODE_LANGS = [
  { id: 'bash', label: 'Bash' },
  { id: 'powershell', label: 'PowerShell' },
  { id: 'sql', label: 'SQL' },
  { id: 'json', label: 'JSON' },
  { id: 'yaml', label: 'YAML' },
  { id: 'javascript', label: 'JavaScript' },
  { id: 'python', label: 'Python' },
  { id: 'text', label: 'Plain text' }
] as const;

type Rule = [cls: string, re: string];

const STR = `"(?:\\\\.|[^"\\\\])*"|'(?:\\\\.|[^'\\\\])*'`;
const NUM = `\\b\\d+(?:\\.\\d+)?\\b`;

const kw = (words: string) => `\\b(?:${words.split(' ').join('|')})\\b`;

const RULES: Record<string, Rule[]> = {
  bash: [
    ['com', `#.*$`],
    ['str', STR],
    ['var', `\\$\\{?[\\w@#?*!-]+\\}?`],
    ['kw', kw('if then else elif fi for in do done while case esac function return export local sudo')],
    ['flag', `(?<=^|\\s)--?[A-Za-z][\\w-]*`],
    ['cmd', `(?<=^\\s*|\\|\\s*|&&\\s*|;\\s*)[A-Za-z_./][\\w./-]*`],
    ['num', NUM]
  ],
  powershell: [
    ['com', `#.*$`],
    ['str', STR],
    ['var', `\\$[\\w:]+`],
    ['cmd', `\\b[A-Z][a-zA-Z]+-[A-Z][A-Za-z]+\\b`],
    ['flag', `(?<=^|\\s)-[A-Za-z][\\w]*`],
    ['kw', kw('if else elseif foreach for while function return param try catch finally switch')],
    ['num', NUM]
  ],
  sql: [
    ['com', `--.*$`],
    ['str', `'(?:''|[^'])*'`],
    ['kw', `(?i)` + kw('select from where and or not insert into values update set delete create table alter drop join left right inner outer on group by order having limit as null is in like distinct primary key foreign references index view')],
    ['num', NUM]
  ],
  json: [
    ['flag', `"(?:\\\\.|[^"\\\\])*"(?=\\s*:)`],
    ['str', `"(?:\\\\.|[^"\\\\])*"`],
    ['kw', kw('true false null')],
    ['num', `-?${NUM}`]
  ],
  yaml: [
    ['com', `#.*$`],
    ['flag', `^\\s*[\\w.-]+(?=\\s*:)`],
    ['str', STR],
    ['kw', kw('true false null yes no')],
    ['num', NUM]
  ],
  javascript: [
    ['com', `\\/\\/.*$`],
    ['str', STR + '|`(?:\\\\.|[^`\\\\])*`'],
    ['kw', kw('const let var function return if else for while do switch case break continue new class extends import from export default async await try catch finally throw typeof instanceof true false null undefined')],
    ['num', NUM]
  ],
  python: [
    ['com', `#.*$`],
    ['str', STR],
    ['kw', kw('def return if elif else for while in import from as class try except finally with lambda yield None True False and or not is pass break continue')],
    ['num', NUM]
  ]
};

const compiled = new Map<string, RegExp>();

function regexFor(lang: string): RegExp | null {
  const rules = RULES[lang];
  if (!rules) return null;
  if (!compiled.has(lang)) {
    let flags = 'gm';
    const src = rules
      .map(([cls, re]) => {
        if (re.startsWith('(?i)')) {
          flags = 'gmi';
          re = re.slice(4);
        }
        return `(?<${cls}>${re})`;
      })
      .join('|');
    compiled.set(lang, new RegExp(src, flags));
  }
  return compiled.get(lang)!;
}

/** Highlights one line (line by line, so line numbers stay simple). */
export function highlightLine(line: string, lang: string): string {
  const re = regexFor(lang);
  if (!re || !line) return escapeHtml(line);
  let out = '';
  let last = 0;
  re.lastIndex = 0;
  for (const m of line.matchAll(re)) {
    if (!m[0]) continue;
    const cls = Object.entries(m.groups ?? {}).find(([, v]) => v !== undefined)?.[0];
    out += escapeHtml(line.slice(last, m.index)) + `<span class="tk-${cls}">${escapeHtml(m[0])}</span>`;
    last = m.index! + m[0].length;
  }
  return out + escapeHtml(line.slice(last));
}

/** Guesses the language if none is selected. */
export function guessLang(code: string): string {
  if (/\b[A-Z][a-z]+-[A-Z][A-Za-z]+\b/.test(code) || /\$\w+\s*=/.test(code)) return 'powershell';
  if (/^\s*[{[]/.test(code)) return 'json';
  if (/\b(select|insert|update|delete)\b[\s\S]*\b(from|into|set)\b/i.test(code)) return 'sql';
  return 'bash';
}
