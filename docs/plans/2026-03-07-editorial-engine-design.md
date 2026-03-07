# FeedNest "Editorial Engine" Redesign

## Overview

A comprehensive frontend redesign that transforms FeedNest into a cinematic, keyboard-driven, magazine-style RSS reader. Three experience layers — browsing (magazine feel), reading (minimal zen), and command (Superhuman speed) — work together without competing.

Tech approach: Pure CSS + vanilla Svelte. View Transitions API with CSS fallback. Canvas for color extraction. Intersection Observer for scroll-linked effects. No heavy animation libraries.

## 1. Motion System

Foundation for all interaction feedback and transitions.

### Spring Physics
- Replace all `ease`/`ease-in-out` with spring-based cubic-bezier curves
- Three presets:
  - `snappy`: 150ms, `cubic-bezier(0.22, 1, 0.36, 1)` — interactive elements
  - `smooth`: 300ms, `cubic-bezier(0.22, 1, 0.36, 1)` — page transitions
  - `dramatic`: 500ms, `cubic-bezier(0.34, 1.56, 0.64, 1)` — hero entrances
- CSS custom properties: `--spring-snappy`, `--spring-smooth`, `--spring-dramatic`

### Shared Element Transitions
- Clicking article card morphs into reader panel (card image becomes reader hero, title slides into position)
- Uses View Transitions API with CSS fallback for unsupported browsers
- Reader close reverses the morph back to card

### Staggered Entrances
- Articles cascade in with spring physics, 40ms stagger
- Sidebar items animate in on first load (one-time)
- Filter tab switches trigger subtle content crossfade

### Scroll-Linked Animations
- Parallax on hero card images (0.3x scroll rate) in magazine/hybrid view
- Header shrinks from full (64px) to compact (48px) on scroll, title fades to breadcrumb
- Articles fade in as they enter viewport (Intersection Observer)

### Reduced Motion
- Respects `prefers-reduced-motion` — all animations collapse to instant
- No rotation, zoom, or continuous looping animations

## 2. Command Layer & Keyboard System

### Command Palette (Cmd+K / Ctrl+K)
- Full-screen overlay with frosted glass backdrop
- Search input at top, results below with fuzzy matching
- Categories: Navigate (feeds, categories), Articles (search by title), Actions (mark all read, refresh, toggle theme, import/export OPML)
- Each result: icon + label + shortcut hint
- Arrow keys to navigate, Enter to execute, Esc to close
- Recent actions shown when palette opens empty

### Vim-Style Navigation
- `j` / `k` — move selection down/up through article list
- `o` or `Enter` — open selected article in reader
- `Escape` — close reader, deselect
- `s` — toggle star
- `m` — toggle read/unread
- `r` — refresh feeds
- `g g` — jump to top (chord sequence)
- `G` — jump to bottom
- `/` — focus search input
- `1` / `2` / `3` — switch view modes

### Keyboard Hints Overlay (press `?`)
- Modal showing all shortcuts grouped by category
- Glassmorphic style matching command palette
- Dismiss with Esc or `?`

### Navigation Indicators
- Selected article: prominent left accent bar + subtle glow
- Selection smoothly slides between items (animated indicator)
- Auto-scrolls selected item into view

## 3. Reading Experience

### Reader Transitions
- Opens via shared element morph from card
- Content paragraphs fade in with 50ms stagger per block
- Close reverses morph back to list position

### Reading Progress Bar
- 2px accent gradient bar fixed at top of viewport
- Tracks scroll position through article content
- Disappears when at top (0%)

### Scroll-Linked Header
- Full: back button, feed name, article title, star/mark-read actions
- On scroll down: collapses to compact bar (back arrow + truncated title + actions)
- On scroll up: re-expands
- Uses `snappy` spring timing

### Typography & Focus
- Max-width 672px, 18px body, 1.8 line-height
- Subtle link underlines with accent hover
- Code blocks with syntax styling
- Images: rounded corners + fade-in on load

### Article Navigation Within Reader
- `j` / `k` switch to next/previous article without closing reader
- Current article slides out left, new slides in from right
- Preloads adjacent articles for instant feel

### Mobile Reading
- Full-screen reader, no sidebar
- Swipe right from left edge to go back
- Pull down at top to close

## 4. Visual Polish

### Dynamic Feed Accent Colors
- Extract dominant color from feed favicon (canvas-based color extraction)
- Cards/list items get subtle tint on left border and unread indicator
- Single-feed view: header gradient shifts to feed's accent color
- Fallback: hash feed URL to generate consistent hue

### Parallax Hero Cards
- Featured images shift at 0.3x scroll rate
- Hover: image shifts toward cursor (magnetic, 3-5px max)
- Shadow deepens + lift on hover with spring physics

### Animated Unread Badges
- Count rolls up like odometer (digit-by-digit transition)
- Feed items with new articles get brief pulse glow

### Celebration Moments
- Star: 5-6 gold sparkle particles burst and fade (200ms)
- Clear all unreads: brief confetti shower (1 second)
- Reduced-motion: just state change, no particles

### Empty States
- Floating/bobbing animated RSS icon
- Contextual messaging with CTA button
- Subtle background pattern animation

### Image Treatment
- Blur-up lazy loading (tiny placeholder to sharp)
- Broken images: colored placeholder with feed initial

### Theme Transitions
- 300ms crossfade on all colors when switching light/dark
- Background, text, surfaces all transition together

## 5. Mobile & Gesture System

### Swipe Gestures (Article List)
- Swipe right: mark read/unread (green indicator reveals)
- Swipe left: toggle star (gold indicator reveals)
- Rubber-band spring physics, snaps back after action
- 80px threshold to trigger

### Pull-to-Refresh
- Pull down at top of list to refresh
- Animated RSS icon spins during refresh
- Spring bounce on release

### Mobile Navigation
- Sidebar swipes in from left edge (enhanced with spring physics)
- Reader swipes away to right
- Bottom safe area respected

### Visual Feedback
- Scale pulse on revealed swipe icon at trigger threshold
