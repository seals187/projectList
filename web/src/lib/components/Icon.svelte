<script lang="ts">
  // Small inline SVG icons (instead of an icon library).
  const paths = {
    menu: 'M4 6h16M4 12h16M4 18h16',
    sidebar: 'M4 5h16v14H4zM9 5v14',
    rows: 'M4 5h16v14H4zM4 9.7h16M4 14.3h16',
    checkSquare: 'M4 4h16v16H4zM8 12l3 3 5-6',
    search: 'M21 21l-4.3-4.3M11 18a7 7 0 1 1 0-14 7 7 0 0 1 0 14z',
    chevron: 'M9 6l6 6-6 6',
    chevronDown: 'M6 9l6 6 6-6',
    droplet: 'M12 3s6 6.5 6 11a6 6 0 0 1-12 0c0-4.5 6-11 6-11z',
    plus: 'M12 5v14M5 12h14',
    minus: 'M5 12h14',
    trash: 'M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3',
    grip: 'M9 6h.01M15 6h.01M9 12h.01M15 12h.01M9 18h.01M15 18h.01',
    logout: 'M15 4h4v16h-4M10 8l-4 4 4 4M6 12h11',
    template: 'M7 3h7l5 5v13H7zM14 3v5h5M10 13h6M10 17h6',
    users: 'M16 19v-1a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v1M9.5 10a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM21 19v-1a4 4 0 0 0-3-3.9M15 4.1a3 3 0 0 1 0 5.8',
    user: 'M20 21v-1a5 5 0 0 0-5-5H9a5 5 0 0 0-5 5v1M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
    tasks: 'M10 6h10M10 12h10M10 18h10M4 6l1 1 2-2M4 12l1 1 2-2M4 18l1 1 2-2',
    settings:
      'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z',
    calendar: 'M4 6h16v14H4zM4 10h16M8 3v4M16 3v4',
    x: 'M6 6l12 12M18 6 6 18',
    check: 'M5 12l5 5L20 7',
    home: 'M3 11l9-7 9 7M5 10v10h14V10',
    download: 'M12 4v12M7 11l5 5 5-5M5 20h14',
    folder: 'M3 6h6l2 2h10v11H3z',
    copy: 'M9 9h11v11H9zM5 15H4V4h11v1',
    undo: 'M9 14L4 9l5-5M4 9h11a5 5 0 0 1 0 10h-4',
    redo: 'M15 14l5-5-5-5M20 9H9a5 5 0 0 0 0 10h4',
    external: 'M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5',
    link: 'M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1',
    eraser: 'M4 20h16M6 11l7-7 6 6-7 7H8l-3-3zM10 7l6 6',
    star: 'M12 3.5l2.6 5.3 5.9.9-4.3 4.1 1 5.8-5.2-2.7-5.2 2.7 1-5.8-4.3-4.1 5.9-.9z',
    clock: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 7v5l3 2',
    sun: 'M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4',
    moon: 'M20 14.5A8 8 0 1 1 9.5 4a6.5 6.5 0 0 0 10.5 10.5z',
    monitor: 'M3 4h18v12H3zM8 20h8M12 16v4',
    zap: 'M13 2 4 14h7l-1 8 9-12h-7z',
    braces: 'M8 4H7a2 2 0 0 0-2 2v4l-2 2 2 2v4a2 2 0 0 0 2 2h1M16 4h1a2 2 0 0 1 2 2v4l2 2-2 2v4a2 2 0 0 1-2 2h-1',
    arrowRight: 'M5 12h14M13 6l6 6-6 6',
    briefcase: 'M3 8h18v12H3zM9 8V5h6v3M3 13h18',
    board: 'M4 4h16v16H4zM9 8v8M14 8v4',
    grid: 'M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z',
    list: 'M8 6h12M8 12h12M8 18h12M4 6h.01M4 12h.01M4 18h.01',
    filter: 'M4 5h16l-6 7.5V19l-4-2v-4.5z',
    timeline: 'M4 4v16M8 7h7M11 12h8M7 17h6',
    columns: 'M4 4h16v16H4zM9.3 4v16M14.7 4v16',
    archive: 'M3 4h18v4H3zM5 8v12h14V8M10 12h4',
    alert: 'M12 3.5 21.5 20h-19L12 3.5zM12 10v4.5M12 17.5h.01',
    image: 'M4 5h16v14H4zM4 16l5-5 4 4 3-3 4 4M15.5 9.5h.01',
    pencil: 'M4 20h4L19 9l-4-4L4 16v4zM13.5 6.5l4 4',
    eye: 'M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12zM12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z',
    flag: 'M5 21V4M5 4h11l-2 4 2 4H5',
    tag: 'M3 12V4h8l10 10-8 8L3 12zM7.5 8.5h.01',
    dots: 'M5 12h.01M12 12h.01M19 12h.01',
    pieChart: 'M21.2 15.9A10 10 0 1 1 8 2.8M22 12A10 10 0 0 0 12 2v10z',
    pieChartOff: 'M21.2 15.9A10 10 0 1 1 8 2.8M22 12A10 10 0 0 0 12 2v10zM3 3l18 18',
    fileText: 'M14 3H7v18h10V6zM14 3v3h3M9.5 12h5M9.5 16h5',
    // lucide: chevrons-down-up, chevrons-up-down, arrow-up-down, list-checks
    chevronsDownUp: 'M7 20l5-5 5 5M7 4l5 5 5-5',
    chevronsUpDown: 'M7 15l5 5 5-5M7 9l5-5 5 5',
    arrowUpDown: 'M21 16l-4 4-4-4M17 20V4M3 8l4-4 4 4M7 4v16',
    listChecks: 'M13 5h8M13 12h8M13 19h8M3 17l2 2 4-4M3 7l2 2 4-4',
    // settings: palette, type (T), database, mail, lock, clipboard list
    palette: 'M12 21a9 9 0 1 1 9-9c0 2.5-2 3.5-4 3.5h-2a2 2 0 0 0-1.4 3.4c.9.9.2 2.1-1.6 2.1zM7.5 10.5h.01M10.5 7h.01M15 7.5h.01M17 11h.01',
    type: 'M5 7V5h14v2M12 5v14M9 19h6',
    database: 'M4 6c0-1.7 3.6-3 8-3s8 1.3 8 3-3.6 3-8 3-8-1.3-8-3zM4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3',
    mail: 'M3 6h18v12H3zM3 7l9 6 9-6',
    lock: 'M6 11h12v10H6zM8.5 11V7.5a3.5 3.5 0 0 1 7 0V11',
    clipboard: 'M9 4h6v3H9zM9 5.5H6V21h12V5.5h-3M9 12h6M9 16h4',
    // lucide: building-2, bookmark (entity types, see $lib/entityTypes)
    building2: 'M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18zM6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2M10 6h4M10 10h4M10 14h4M10 18h4',
    bookmark: 'M19 21l-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z',
    // lucide: info, github
    info: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zM12 16v-4M12 8h.01',
    github: 'M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.4 5.4 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4M9 18c-4.51 2-5-2-7-2'
  } as const;

  let { name, size = 16, class: cls = '' }: { name: keyof typeof paths; size?: number; class?: string } = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width={name === 'grip' || name === 'dots' ? 3 : 1.8}
  stroke-linecap="round"
  stroke-linejoin="round"
  class="shrink-0 {cls}"
  aria-hidden="true"><path d={paths[name]} /></svg>
