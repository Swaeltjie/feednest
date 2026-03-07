# FeedNest v2 Frontend Redesign

## Problem

The v1 frontend is functional but visually flat — plain white cards, no content snippets, no animations, boring sidebar, and a dark mode that's just gray. It feels like a bootstrap template. The goal is to make it visually competitive with (and better than) Feedly.

## Visual Direction

**Glassmorphism + bold visuals + adaptive dark layering.**

Frosted glass panels with physical depth, vibrant accent colors, gradient overlays on imagery, and layered dark surfaces that create a sense of dimension. Think Arc browser meets Apple News.

## Color System

### Dark Mode — Adaptive Depth Layers
Each UI layer has a distinct shade, creating physical depth:

| Layer | Color | Usage |
|-------|-------|-------|
| Base | `#0d1117` | Sidebar background |
| Surface | `#161b22` | Main content background |
| Card | `#1c2128` | Article cards, rows |
| Elevated | `#22272e` | Hover states, modals |
| Border | `rgba(255,255,255,0.06)` | Subtle card borders |
| Border hover | `rgba(255,255,255,0.12)` | Hover border brightening |
| Accent | `#3b82f6` → `#8b5cf6` | Blue-to-purple gradient |
| Accent glow | `rgba(59,130,246,0.15)` | Active state glow |

### Light Mode
| Layer | Color | Usage |
|-------|-------|-------|
| Base | `#f8f9fa` | Sidebar background |
| Surface | `#f0f2f5` | Main content background |
| Card | `#ffffff` | Article cards, rows |
| Border | `rgba(0,0,0,0.06)` | Card borders |
| Accent | `#2563eb` → `#7c3aed` | Blue-to-purple gradient |

## Content Area Redesign

### Featured Row (Top 2-3 Articles)
The highest-scored articles display as large hero cards:

```
+---------------------------+  +---------------------------+
|                           |  |                           |
|      [Full-bleed image    |  |      [Full-bleed image    |
|       with gradient       |  |       with gradient       |
|       overlay →           |  |       overlay →           |
|                           |  |                           |
|  ┌─ frosted glass bar ──┐|  |  ┌─ frosted glass bar ──┐|
|  │ Source · 2h · 4 min   │|  │ Source · 5h · 3 min   │|
|  └───────────────────────┘|  |  └───────────────────────┘|
|  Bold Article Title Here  |  |  Another Great Title      |
|  That Can Span Two Lines  |  |  Worth Reading Today      |
+---------------------------+  +---------------------------+
```

- Full-bleed thumbnail with CSS gradient overlay (`linear-gradient(transparent 40%, rgba(0,0,0,0.8))`)
- Bold white title text over the darkened image area
- Frosted glass metadata strip (`backdrop-filter: blur(12px)`, semi-transparent background)
- Hover: subtle scale(1.02), enhanced shadow, border brightens
- If no thumbnail: use a gradient background based on feed accent color

### Dense Article List (Remaining Articles)
Below the featured row, a space-efficient list:

```
┌────────────────────────────────────────────────────────┐
│ ┌──────┐                                               │
│ │thumb │  Article Title That Might Be Long             │
│ │64x64 │  First two lines of the article snippet...    │
│ │      │  [favicon] Source · 2h ago · 3 min    ☆       │
│ └──────┘                                               │
├────────────────────────────────────────────────────────┤
│ ┌──────┐                                               │
│ │thumb │  Another Article Title                        │
│ │64x64 │  Snippet text providing preview of content... │
│ │      │  [favicon] Source · 4h ago · 5 min    ★       │
│ └──────┘                                               │
└────────────────────────────────────────────────────────┘
```

- Rounded thumbnail (8px radius, 64x64)
- Title + 2-line snippet (from `content_clean`, stripped to plain text)
- Source with feed favicon, relative time, reading time
- Unread articles: colored left border (accent gradient, 3px)
- Starred articles: subtle gold shimmer on the star icon
- Read articles: slightly reduced opacity (0.65)
- Hover: row lifts with shadow, background transitions to frosted glass

### Skeleton Loading
Replace spinners with shimmer skeleton screens:
- Featured row: 2 large placeholder rectangles with gradient shimmer animation
- List rows: thumbnail placeholder + text line placeholders with shimmer
- CSS-only animation: `@keyframes shimmer` with gradient sweep

## Animations & Transitions

### Page Load
- Featured cards and list rows fade-in with stagger (`animation-delay` incremented per item)
- Duration: 300ms per item, 50ms stagger offset

### View Switching
- Card ↔ List toggle uses crossfade (old view fades out, new fades in, 200ms)

### Article Reader Transition
- Slide-in from right (transform: translateX(100%) → translateX(0))
- Duration: 300ms, ease-out curve
- Back navigation: reverse slide

### Hover Effects
- Cards: `transform: translateY(-2px)`, shadow deepens, border brightens
- List rows: `translateY(-1px)`, frosted background appears
- All transitions: 150ms ease

### Micro-interactions
- Star toggle: brief scale bounce (1.0 → 1.3 → 1.0, 200ms)
- Mark as read: smooth opacity transition
- Unread badge count changes: number briefly scales

## Toolbar Redesign

```
┌─────────────────────────────────────────────────────────┐
│ ☰  [All] [Unread] [Starred]          Sort ▾  ⊞ ≡  🌓  │
│    ─────────────────                                    │
│    gradient underline                                   │
└─────────────────────────────────────────────────────────┘
```

- Frosted glass header: `backdrop-filter: blur(16px)`, semi-transparent bg
- Filter tabs: no background toggle — instead, active tab has a gradient underline (accent gradient, 2px, animated slide)
- Sort: custom-styled dropdown with frosted glass panel
- Smooth transitions on tab switch (underline slides between tabs)

## Sidebar Polish

Keep it simple but elevated:
- Feed favicons (`icon_url` from feed metadata) displayed as 16x16 rounded images next to feed names
- Fallback: first-letter avatar with accent gradient background
- Active feed/category: accent gradient background with subtle glow (`box-shadow: 0 0 12px accent-glow`)
- Smooth collapse transition (width animates, content fades)
- Subtle section dividers between categories

## Article Reader Upgrade

- Slide-in-from-right transition (not a page navigation — feels like opening a panel)
- Frosted glass sticky header with `backdrop-filter: blur(16px)`
- Hero image spans full width with subtle parallax-lite (background-attachment or minor transform on scroll, CSS only)
- Typography: tighter heading letter-spacing (-0.02em), 1.75 line-height for body, slightly larger base font (18px)
- Prose content: enhanced blockquote styling (accent left border + gradient bg), code blocks with syntax-highlight-ready styling

## Auth Pages

- Centered card with frosted glass effect
- Subtle animated gradient background (slow-moving color shifts)
- FeedNest logo/wordmark at top

## Technical Approach

All changes are CSS/Svelte template only — no new dependencies except potentially one:
- **Optional**: `@tailwindcss/typography` (already installed) for prose styling
- All animations are CSS-based (keyframes, transitions) — no JS animation library
- Glassmorphism via `backdrop-filter: blur()` (supported in all modern browsers)
- Skeleton screens are CSS-only (gradient animation)
- Snippets require backend to return a `snippet` field (plain text, ~160 chars from `content_clean`)

## Backend Change

One small backend addition needed:
- Add a `snippet` field to the article list response — strip HTML from `content_clean`, truncate to 160 characters
- This keeps the list API response lightweight while providing preview text

## What We're NOT Changing

- Layout structure (sidebar + main content) stays the same
- Backend API (except snippet field addition)
- Routing and navigation logic
- Store/state management
- Keyboard shortcuts
