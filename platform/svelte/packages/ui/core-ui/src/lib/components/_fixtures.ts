/**
 * Unified component fixtures catalogue.
 *
 * Single source of truth for ALL demo data used across component previews,
 * detail pages, and card listings. Import from here — never duplicate.
 *
 * Backward-compatible aliases at the bottom preserve existing imports.
 */

// ── Team ──────────────────────────────────────────────────────────────

export interface TeamMember {
  name: string;
  role: string;
  initials: string;
  color: string;
  description: string;
  stats: { label: string; value: number }[];
  links: string[];
}

export const TEAM: TeamMember[] = [
  {
    name: 'Andrei Solovev',
    role: 'Lead Engineer',
    initials: 'AS',
    color: '#a78bfa',
    description: 'Full-stack architect specializing in distributed systems and developer tooling.',
    stats: [
      { label: 'repos', value: 42 },
      { label: 'commits', value: 1847 }
    ],
    links: ['Profile', 'Projects']
  },
  {
    name: 'Maya Patel',
    role: 'Designer',
    initials: 'MP',
    color: '#22d3ee',
    description: 'UI/UX designer focused on design systems and accessible interfaces.',
    stats: [
      { label: 'repos', value: 18 },
      { label: 'commits', value: 923 }
    ],
    links: ['Profile', 'Dribbble']
  },
  {
    name: 'Sam Torres',
    role: 'DevOps',
    initials: 'ST',
    color: '#4ade80',
    description: 'Infrastructure engineer managing CI/CD pipelines and cloud deployments.',
    stats: [
      { label: 'repos', value: 31 },
      { label: 'commits', value: 2104 }
    ],
    links: ['Profile', 'Infra']
  },
  {
    name: 'Dora Okafor',
    role: 'The Explorer',
    initials: 'DO',
    color: '#f472b6',
    description: 'Technical writer and documentation specialist exploring new frameworks.',
    stats: [
      { label: 'repos', value: 27 },
      { label: 'commits', value: 1356 }
    ],
    links: ['Profile', 'Docs']
  },
  {
    name: 'Rui Nakamura',
    role: 'Backend',
    initials: 'RN',
    color: '#fbbf24',
    description: 'API architect building high-performance Go services and data pipelines.',
    stats: [
      { label: 'repos', value: 55 },
      { label: 'commits', value: 3201 }
    ],
    links: ['Profile', 'API']
  },
  {
    name: 'Leila Amari',
    role: 'Frontend',
    initials: 'LA',
    color: '#fb7185',
    description: 'Svelte specialist building component libraries and interactive experiences.',
    stats: [
      { label: 'repos', value: 23 },
      { label: 'commits', value: 1102 }
    ],
    links: ['Profile', 'Components']
  },
];

// ── Tech ──────────────────────────────────────────────────────────────

export interface TechItem {
  name: string;
  icon: string;
  category: string;
  color: string;
  description: string;
  url?: string;
}

export const TECH: TechItem[] = [
  {
    name: 'SvelteKit',
    icon: '\u26A1',
    category: 'Framework',
    color: '#ff3e00',
    description: 'Full-stack web framework with SSR, routing, and form actions.',
    url: 'kit.svelte.dev'
  },
  {
    name: 'Go',
    icon: '\uD83D\uDD27',
    category: 'Backend',
    color: '#00add8',
    description: 'Systems language for APIs, CLI tools, and infrastructure.',
    url: 'go.dev'
  },
  {
    name: 'PostgreSQL',
    icon: '\uD83D\uDC18',
    category: 'Database',
    color: '#336791',
    description: 'Relational database with PostGIS, pgvector extensions.',
    url: 'postgresql.org'
  },
  {
    name: 'Redis',
    icon: '\u26A1',
    category: 'Cache',
    color: '#dc382d',
    description: 'In-memory store for sessions, queues, and real-time data.',
    url: 'redis.io'
  },
  {
    name: 'Cloudflare',
    icon: '\u2601\uFE0F',
    category: 'CDN',
    color: '#f6821f',
    description: 'Edge network for DNS, R2 storage, and asset delivery.',
    url: 'cloudflare.com'
  },
  {
    name: 'Docker',
    icon: '\uD83D\uDC33',
    category: 'Infra',
    color: '#2496ed',
    description: 'Container runtime for local dev and production deployment.',
    url: 'docker.com'
  },
];

// ── Link Preview ──────────────────────────────────────────────────────

export interface LinkPreviewFixture {
  href: string;
  title: string;
  domain: string;
  description: string;
  label: string;
}

export const LINK_PREVIEW: LinkPreviewFixture = {
  href: '/process',
  title: 'Development Process',
  domain: 'shredbx.com',
  description: 'Governed SDLC methodology with full traceability from requirements to deployment.',
  label: 'development process',
};

// ── Text Labels (text animation previews) ─────────────────────────────

export interface TextLabel {
  id: number;
  full: string;
  before: string;
  target: string;
  after: string;
  highlightPos: 'end' | 'inner';
  // Quad split (for QuadSplit animation)
  quadBefore: string;
  quadTarget: string;
  quadAfter: string;
}

export const TEXT_LABELS: TextLabel[] = [
  { id: 1, full: 'shredbx',         before: 'shred',  target: 'bx', after: '',              highlightPos: 'end',   quadBefore: '',    quadTarget: 'S', quadAfter: 'hredbx' },
  { id: 2, full: 'eXperimental Lab', before: 'e',      target: 'X',  after: 'perimental Lab', highlightPos: 'inner', quadBefore: 'e',   quadTarget: 'X', quadAfter: 'perimental Lab' }
];

/**
 * Split text around the highlight target.
 */
export function splitLabel(label: TextLabel): { before: string; target: string; after: string } {
  if (!label.target) return { before: '', target: label.full, after: '' };
  const idx = label.full.indexOf(label.target);
  if (idx === -1) return { before: '', target: label.full, after: '' };
  return {
    before: label.full.slice(0, idx),
    target: label.target,
    after: label.full.slice(idx + label.target.length)
  };
}

// ── Typewriter Suggestions (auto-cycling typewriter) ─────────────────

export interface TypewriterSuggestion {
  full: string;
  before?: string;
  target?: string;
  after?: string;
}

export const TYPEWRITER_SUGGESTIONS: TypewriterSuggestion[] = [
  { full: 'shredbx',          before: 'shred',  target: 'bx', after: '' },
  { full: 'eXperimental Lab', before: 'e',      target: 'X',  after: 'perimental Lab' },
  { full: 'build to last',    before: 'build to ', target: 'last', after: '' },
  { full: 'ship it',          before: 'ship ',  target: 'it',  after: '' },
];

// ── Card Demos (interactive border cards) ─────────────────────────────

export const CARD_DEMOS = {
  mouseFollowing: {
    title: 'Interactive Borders',
    body: 'The gradient origin follows your cursor, creating a spotlight effect that responds to user interaction in real time.',
    hint: 'Hover and move your mouse over this card'
  },
  doubleBorder: {
    title: 'Dual Rotation',
    body: 'Counter-rotating conic gradients at different speeds produce a shimmering, organic border that feels alive and unpredictable.'
  },
  rainbow: {
    tag: 'Service',
    title: 'Governed Precision',
    body: 'Every decision traceable, every component governed. Framework-driven development that eliminates guesswork and enforces quality.'
  },
} as const;

// ── CTA Labels ────────────────────────────────────────────────────────

export const CTA_LABELS = {
  moving: 'Get Started',
  thin: 'View Process',
} as const;

// ── NavBar Demo ───────────────────────────────────────────────────────

export interface NavGroup {
  name: string;
  items: string[];
}

export const NAV_DEMO = {
  brand: 'shredbx',
  groups: [
    { name: 'explore',  items: ['Portfolio', 'Blog'] },
    { name: 'build',    items: ['Components', 'Docs'] },
    { name: 'connect',  items: ['Team', 'Contact'] }
  ] as NavGroup[],
  flat: ['Portfolio', 'Blog', 'Components', 'Docs', 'Team', 'Contact'] as string[],
} as const;

// ── Link Preview Prose (ComponentDemo inline text) ────────────────────

export const LP_PROSE = {
  linkPreview: { before: 'Our development workflow is built on a governed SDLC methodology. You can explore the full', trigger: 'development process', after: 'to understand how we deliver software with traceability at every step, from requirements through to deployment.', href: '/process', title: 'Development Process' },
  split: { before: 'Learn about the full', trigger: 'delivery methodology', after: 'that governs every project — from initial goal definition through model-driven implementation to automated verification.', href: '/process', title: 'Development Process', description: 'Governed SDLC with full traceability from requirements to deployment.', domain: 'shredbx.com' },
} as const;

// ── Protocol Block (terminal HUD) ────────────────────────────────────

export const PROTOCOL_BLOCK = {
  line1Pre: 'PRO',
  line1Target: 'COL',
  line2: '>> TRANSMISSION_ACTIVE',
  accent: '#10b981',
} as const;

// ── Stat Headline (editorial stat) ──────────────────────────────────

export const STAT_HEADLINE = {
  line1Accompany: 'Fifty ',
  line1Target: 'Projects',
  line2Target: 'Two',
  line2Accompany: ' Decades',
  accent: '#c0392b',
} as const;

// ── Manifesto Stack (stacked motto) ─────────────────────────────────

export const MANIFESTO_STACK = {
  words: [
    { target: 'M', after: 'ake', effect: 'split' as const },
    { before: 'Meas', target: 'ure', effect: 'split' as const },
    { target: 'Ship', effect: 'glitch' as const },
  ],
  accent: '#c0392b',
} as const;

export const DEMO_HINTS = {
  tooltip: 'hover any avatar to see the tooltip',
  linkPreview: 'hover the link text to see the preview animation',
} as const;

// ── Hero Backgrounds ─────────────────────────────────────────────────

export interface HeroBgVariant {
  id: string;
  label: string;
  description: string;
  suggestedPage: string;
}

export const HERO_BACKGROUNDS: HeroBgVariant[] = [
  { id: 'hero-bg-grid',         label: 'Grid Pattern',         description: 'CSS grid lines with radial center fade and subtle glow.',                suggestedPage: 'Process' },
  { id: 'hero-bg-dots',         label: 'Dot Pattern',          description: 'Radial dot grid with cursor-following glow.',                            suggestedPage: 'Blog' },
  { id: 'hero-bg-spotlight',    label: 'Cursor Spotlight',     description: 'Grid with dual-layer cursor-following glow.',                            suggestedPage: 'Contact' },
  { id: 'hero-bg-dual-beam',    label: 'Dual Beam Spotlight',  description: 'Animated light beams with sinusoidal motion.',                           suggestedPage: 'Homepage' },
  { id: 'hero-bg-highlight',    label: 'Hero Highlight',       description: 'Grid with accent-colored cursor glow.',                                  suggestedPage: 'Services' },
  { id: 'hero-bg-dot-ripple',   label: 'Dot + Cursor Ripple',  description: 'Dot grid with cursor glow and click-expanding rings.',                   suggestedPage: 'Portfolio' },
  { id: 'hero-bg-mesh',         label: 'Gradient Mesh',        description: 'Floating gradient blobs with cursor glow.',                              suggestedPage: 'About' },
  { id: 'hero-bg-electric',     label: 'Electric Grid Pulse',  description: 'Canvas-based electric pulse propagating along grid lines on click.',     suggestedPage: 'Contact/Alt' },
];

// ── Terminal Block fixture ────────────────────────────────────────────

export interface TerminalEntryFixture {
  segments: { type: 'prompt' | 'command' | 'output' | 'highlight' | 'info'; text: string }[];
}

export const TERMINAL_BLOCK = {
  title: 'Terminal',
  entries: [
    {
      segments: [
        { type: 'prompt' as const,  text: '$ ' },
        { type: 'command' as const, text: 'sbx sdlc init "feature" --action new-feature' },
      ],
    },
    {
      segments: [
        { type: 'output' as const,    text: 'SDLC initialized: ' },
        { type: 'highlight' as const, text: 'task 2604-013' },
      ],
    },
    {
      segments: [
        { type: 'output' as const,    text: 'Steps: ' },
        { type: 'info' as const,      text: '31' },
        { type: 'output' as const,    text: ' | Next: ' },
        { type: 'highlight' as const, text: 'FDD1.G' },
      ],
    },
  ],
} as const;

// ── Process Step fixture ──────────────────────────────────────────────

export const PROCESS_STEP = {
  step: 1,
  title: 'Design',
  description: 'Define goals, user stories, and acceptance criteria before writing any code.',
  command: 'sbx sdlc next',
} as const;

// ── Project Card fixture ──────────────────────────────────────────────

export const PROJECT_CARD = {
  icon: '🖥',
  name: 'AI-Augmented SDLC',
  description: 'Governed software delivery pipeline with full traceability from requirements to deployment.',
  tags: ['Go', 'SvelteKit', 'PostgreSQL'],
  href: '/development',
} as const;

// ── Blog Post Card fixture ────────────────────────────────────────────

export interface BlogPostCardFixture {
  href: string;
  title: string;
  excerpt: string;
  author: string;
  date: string;
  readingTime: number;
  categoryLabel: string;
  featured: boolean;
}

export const BLOG_POST_CARD: BlogPostCardFixture = {
  href: '/blog/decision-records',
  title: 'Decision Records as a First-Class Artifact',
  excerpt:
    'How immutable, source-backed decision records replace tribal knowledge and make AI-augmented teams faster without sacrificing traceability.',
  author: 'Claude',
  date: 'April 15, 2026',
  readingTime: 7,
  categoryLabel: 'SDLC & Process',
  featured: true,
};

// ── Sidebar Navigation Fixtures ──────────────────────────────────────

export const SIDEBAR_HEADER = {
  title: 'Components',
  showSearch: true,
  searchPlaceholder: 'Search components...',
} as const;

export const SIDEBAR_SECTION = {
  label: 'Browse',
} as const;

export const COLLECTION_ITEMS = [
  { icon: '⬛', label: 'All Components', count: 52, active: false },
  { icon: '⭐', label: 'Recently Updated', count: 8, active: true },
  { icon: '📦', label: 'Favorites', count: 12, active: false },
] as const;

export const CONTENT_NAV_GROUPS = [
  {
    label: 'Animations',
    open: true,
    items: [
      { label: 'Text Effects', href: '/components/animations/text', icon: 'T', color: '#a78bfa' },
      { label: 'Backgrounds', href: '/components/animations/background', icon: '◈', color: '#22d3ee' },
      { label: 'Hero Backgrounds', href: '/components/animations/hero-backgrounds', icon: '⬡', color: '#4ade80' },
    ],
  },
  {
    label: 'Components',
    open: false,
    items: [
      { label: 'Navigation', href: '/components/navigation', icon: '≡', color: '#fbbf24' },
      { label: 'Buttons', href: '/components/buttons', icon: '◉', color: '#fb7185' },
      { label: 'Cards', href: '/components/cards', icon: '▣', color: '#60a5fa' },
    ],
  },
] as const;

// ── Hero Sections ────────────────────────────────────────────────────

export const HERO_SECTIONS = {
  fullBleed: {
    tag: 'AI-Augmented Development',
    headline: 'Build to Last',
    subtitle: 'Governed precision at every layer. Framework-driven development that eliminates guesswork and enforces quality.',
    primaryCta: 'Get Started',
    primaryHref: '#',
    ghostCta: 'View Process',
    ghostHref: '/process',
    showScrollHint: true,
  },
  splitScreen: {
    overline: 'ShredBX Framework',
    headline: 'Ship With Confidence',
    body: 'Model-driven development with built-in governance. Every line of code traces back to a requirement.',
    primaryCta: 'Start Building',
    primaryHref: '#',
    ghostCta: 'View Process',
    ghostHref: '/process',
    visualLabel: 'hero-visual.svg',
  },
  textDominant: {
    headlineLine1: 'Governed Precision.',
    headlineLine2: 'Ship With Confidence.',
    ctaCommand: 'sbx sdlc init "feature" --action new-feature',
    bgNumber: '03',
  },
  cardHero: {
    tag: 'Introducing SBX v2',
    headline: 'The Framework for Governed Development',
    body: 'From model to production in one pipeline. Every decision tracked, every deployment traced.',
    primaryCta: 'Get Started',
    primaryHref: '#',
    ghostCta: 'Documentation',
    ghostHref: '/docs',
    bgImage: 'https://media.shredbx.com/website/backgrounds/posters/dna-001-neurons-poster.jpg',
  },
} as const;

// ── Landing Block Fixtures ────────────────────────────────────────────

export const TIMELINE_ITEMS = [
  { step: '01', title: 'Define Goals', description: 'Establish requirements, user stories, and acceptance criteria before writing any code.' },
  { step: '02', title: 'Model Domain', description: 'Design the domain model, package structure, and data flow with full traceability.' },
  { step: '03', title: 'Implement', description: 'Build bottom-up: domain types → repositories → API routes → UI components.' },
  { step: '04', title: 'Verify & Ship', description: 'Build passes, tests green, conformance checked. Commit with governance trailers.' },
] as const;

export const PROCESS_STEPS_ITEMS = [
  { number: 1, title: 'Requirements', description: 'User stories with measurable acceptance criteria.' },
  { number: 2, title: 'Design', description: 'Domain model and sequence diagrams before any code.' },
  { number: 3, title: 'Implement', description: 'Layer-by-layer implementation following the model.' },
  { number: 4, title: 'Verify', description: 'Automated tests, build checks, and conformance gate.' },
] as const;

// ── FDD Phases fixture (shared by PhasePipeStrip, PhaseSpecTable, ProcessLiveTerminal) ──

export type FDDGate = 'pass' | 'active' | 'pending';

export interface FDDArtifact {
  name: string;
  primary?: boolean;
}

export interface FDDStep {
  name: string;
  detail: string;
}

export interface FDDPhase {
  idx: string;
  id: string;
  label: string;
  name: string;
  colorVar: string;
  darkText?: boolean;
  description: string;
  artifacts: FDDArtifact[];
  gate: FDDGate;
  entry_criteria: string[];
  exit_criteria: string[];
  deliverables: string[];
  knowledge_refs: string[];
  example_outputs: string[];
  steps: FDDStep[];
}

export const FDD_PHASES: FDDPhase[] = [
  {
    idx: '01',
    id: 'FDD1',
    label: 'MODEL',
    name: 'Develop Overall Model',
    colorVar: '--color-info',
    description:
      'Build domain model. Define entities, relationships, and business rules as the foundation for all features.',
    artifacts: [
      { name: 'Domain Model', primary: true },
      { name: 'User Stories' },
      { name: 'Goals' },
    ],
    gate: 'pass',
    entry_criteria: [
      'Business objective or feature request defined',
      'Stakeholders identified',
    ],
    exit_criteria: [
      'All stories have acceptance criteria',
      'Domain model reviewed and validated',
      'No ambiguous terms in shared vocabulary',
    ],
    deliverables: [
      'Goal definition with measurable success criteria',
      'User stories with actors, scenarios, acceptance criteria',
      'Domain model with types, fields, relationships',
      'Sequence diagrams for key workflows',
    ],
    knowledge_refs: ['modeling-standard', 'entity-modeling', 'ai-agent-governance', 'context-engineering'],
    example_outputs: [
      "Goal: 'Enable secure multi-tenant package delivery tracking' with KPIs",
      "User Story: 'As a tenant admin, I can view all packages for my building'",
      'Domain Model: Package, Tenant, Building, DeliveryEvent entities',
      'Sequence: Package intake → notification → pickup → confirmation',
    ],
    steps: [
      { name: 'Goals & Vision', detail: 'measurable objectives, success criteria' },
      { name: 'User Stories & Actors', detail: 'scenarios, acceptance criteria' },
      { name: 'Domain Model & Sequences', detail: 'entities, relationships, shared vocabulary' },
    ],
  },
  {
    idx: '02',
    id: 'FDD2',
    label: 'DESIGN',
    name: 'Build Features List',
    colorVar: '--color-success',
    description:
      'Design by feature. Package models, API contracts, page routes, and infrastructure for each feature.',
    artifacts: [
      { name: 'Package Spec', primary: true },
      { name: 'Page Models' },
      { name: 'Migrations' },
    ],
    gate: 'pass',
    entry_criteria: ['FDD1 domain model complete', 'Stories prioritized for iteration'],
    exit_criteria: [
      'Every entity has a package.yml model',
      'All API routes documented',
      'Migration scripts reviewed',
    ],
    deliverables: [
      'Package YAML models with types, exports, dependencies',
      'Page route definitions with data flow',
      'Database migration scripts',
      'Infrastructure requirements (services, ports, volumes)',
    ],
    knowledge_refs: ['coding-standard', 'layered-service-exposure', 'sveltekit-server-load-pattern', 'type-safe-persistence'],
    example_outputs: [
      'package.yml: Package model with types, functions, exports',
      'Page Model: /packages/[id] with server load, breadcrumb',
      'Migration 006_packages.up.sql: CREATE TABLE packages',
      'Infrastructure: PostgreSQL schema allocation, port registry',
    ],
    steps: [
      { name: 'Package Models', detail: 'types, exports, dependencies' },
      { name: 'Page Models', detail: 'routes, data flow, breadcrumbs' },
      { name: 'Persistence & Infrastructure', detail: 'migrations, schemas, ports' },
    ],
  },
  {
    idx: '03',
    id: 'FDD3',
    label: 'PLAN',
    name: 'Plan by Feature',
    colorVar: '--color-warning',
    darkText: true,
    description:
      'Plan by feature. Test specs, acceptance criteria, fixtures, and a consolidated plan defined before code.',
    artifacts: [
      { name: 'Test Specs', primary: true },
      { name: 'NFR' },
      { name: 'Fixtures' },
    ],
    gate: 'pass',
    entry_criteria: ['Package and page models finalized', 'Infrastructure provisioned or mocked'],
    exit_criteria: [
      'All requirements have test coverage mapped',
      'User has approved the plan',
      'Fixture data prepared',
    ],
    deliverables: [
      'Functional requirements checklist per function',
      'Non-functional requirements (performance, security, accessibility)',
      'Test plan with cases, fixtures, assertions',
      'Consolidated plan for user review',
    ],
    knowledge_refs: ['defensive-programming', 'go-api-credential-masking', 'test-driven-development', 'nfr-catalogue'],
    example_outputs: [
      "FR: 'Package.Create must validate tenant ownership before insert'",
      "NFR: 'Package list API responds < 200ms for 1000 packages'",
      "Test: 'TestCreatePackage_InvalidTenant_Returns403'",
      'Fixture: 3 tenants, 10 packages, 5 delivery events',
    ],
    steps: [
      { name: 'Functional Requirements', detail: 'per-function checklists, acceptance' },
      { name: 'Non-Functional Requirements', detail: 'security, performance, accessibility' },
      { name: 'Test Cases & Fixtures', detail: 'TDD assertions, fixture data' },
      { name: 'User Checkpoint', detail: 'mandatory approval gate before code' },
    ],
  },
  {
    idx: '04',
    id: 'FDD4',
    label: 'BUILD',
    name: 'Implement per Layer',
    colorVar: '--color-accent-amber',
    darkText: true,
    description:
      'Build bottom-up, layer by layer. Pure domain → repositories → API routes → UI pages. Governance enforced at every step.',
    artifacts: [
      { name: 'Source', primary: true },
      { name: 'Repositories' },
      { name: 'Pages' },
    ],
    gate: 'active',
    entry_criteria: ['Plan approved at FDD3 checkpoint', 'All models and test fixtures ready'],
    exit_criteria: [
      'All layers implemented per model',
      'Each layer has passing unit tests',
      'No lint or type errors',
    ],
    deliverables: [
      'Type definitions and business logic (pure functions)',
      'Repository implementations with typed queries',
      'API routes and adapter integrations',
      'Svelte pages with server-side data loading',
      'Cross-cutting concerns (auth, logging, errors)',
    ],
    knowledge_refs: ['layered-service-exposure', 'coding-standard', 'sveltekit-server-load-pattern', 'go-api-credential-masking'],
    example_outputs: [
      'PD: Package type with validation rules, DeliveryStatus enum',
      'MD: PackageRepository with Create/Read/Update/Delete methods',
      'SI: /api/v1/packages endpoints with auth middleware',
      'UI: /packages/+page.svelte with server-loaded data grid',
    ],
    steps: [
      { name: 'Pure Domain', detail: 'types, logic, zero I/O' },
      { name: 'Model Domain', detail: 'repositories, typed queries' },
      { name: 'Service Integration', detail: 'APIs, error handling, credential masking' },
      { name: 'User Interface', detail: 'pages, server-side data loading' },
    ],
  },
  {
    idx: '05',
    id: 'FDD5',
    label: 'VERIFY',
    name: 'Verify + Commit',
    colorVar: '--color-accent',
    description:
      'Verify conformance. Automated tests, model completeness, build validation, and a governed commit with traceability trailers.',
    artifacts: [
      { name: 'Tests', primary: true },
      { name: 'Conformance' },
      { name: 'Commit' },
    ],
    gate: 'pending',
    entry_criteria: ['All FDD4 layers complete', 'No known failing tests'],
    exit_criteria: [
      'Build passes',
      'All test suites green',
      'Conformance check passes',
      'Commit pushed with trailers',
    ],
    deliverables: [
      'Clean build output (zero warnings)',
      'All tests green (unit + integration + e2e)',
      'Model-code conformance report',
      'Git commit with governance trailers',
    ],
    knowledge_refs: ['validate-before-reporting', 'bi-directional-conformance', 'coding-standard', 'ai-agent-governance'],
    example_outputs: [
      "Build: 'sbx build --platform go --dir pkg/packages' exits 0",
      "Tests: '12 passed, 0 failed, 0 skipped' across 3 suites",
      "Conformance: 'package.yml exports match Go public API — 100%'",
      "Commit: 'feat(packages): delivery tracking — Task: 2603-300'",
    ],
    steps: [
      { name: 'Build Verification', detail: 'zero warnings, clean compile' },
      { name: 'Test Execution', detail: 'unit, integration, e2e' },
      { name: 'Conformance Check', detail: 'model-code bi-directional alignment' },
      { name: 'Governed Commit', detail: 'trailers, task ID, decision refs' },
    ],
  },
];

// ── Backward-compatible aliases ───────────────────────────────────────

export { TEAM as TEAM_FIXTURE, TECH as TECH_FIXTURE };
export { LINK_PREVIEW as DEMO_FIXTURE };
