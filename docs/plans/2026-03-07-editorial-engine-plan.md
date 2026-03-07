# Editorial Engine Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform FeedNest's frontend into a cinematic, keyboard-driven, magazine-style RSS reader with spring animations, command palette, shared element transitions, and mobile gestures.

**Architecture:** Five layers implemented bottom-up: motion system (CSS foundation) → command layer (keyboard + palette) → reading experience (progress bar, collapsing header, article nav) → visual polish (dynamic colors, celebrations, blur-up images) → mobile gestures (swipe, pull-to-refresh). Each layer builds on the previous.

**Tech Stack:** SvelteKit 5, Svelte 5 runes, TailwindCSS 4, View Transitions API, Intersection Observer, Canvas API (color extraction), pure CSS animations (no libraries).

**Codebase:** ~2,829 lines across 16 frontend files. All files use tab indentation.

---

### Task 1: Spring Motion System Foundation

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/app.css` (lines 1-243)

**Step 1: Add spring CSS custom properties and timing presets**

Add after the dark theme block (after line 34) in `app.css`:

```css
/* Spring motion presets */
:root {
  --spring-snappy: cubic-bezier(0.22, 1, 0.36, 1);
  --spring-smooth: cubic-bezier(0.22, 1, 0.36, 1);
  --spring-dramatic: cubic-bezier(0.34, 1.56, 0.64, 1);
  --duration-snappy: 150ms;
  --duration-smooth: 300ms;
  --duration-dramatic: 500ms;
}

@media (prefers-reduced-motion: reduce) {
  :root {
    --duration-snappy: 0ms;
    --duration-smooth: 0ms;
    --duration-dramatic: 0ms;
  }
}
```

**Step 2: Replace existing transition timings throughout app.css**

Update `.glass-card` (line 52-58) to use spring variables:
```css
.glass-card {
  transition: transform var(--duration-snappy) var(--spring-snappy),
              box-shadow var(--duration-snappy) var(--spring-snappy);
}
```

Update `.hover-lift` (lines 230-236):
```css
.hover-lift {
  transition: transform var(--duration-snappy) var(--spring-snappy),
              box-shadow var(--duration-snappy) var(--spring-snappy);
}
```

Update `fadeInUp` keyframe (line 108) to use spring dramatic timing:
```css
.fade-in-up {
  animation: fadeInUp var(--duration-smooth) var(--spring-smooth) both;
}
```

Update `star-bounce` (line 128) to use spring:
```css
.star-bounce {
  animation: starBounce var(--duration-snappy) var(--spring-dramatic);
}
```

**Step 3: Add theme transition support**

Add to `app.css`:
```css
/* Smooth theme transitions */
html.theme-transitioning,
html.theme-transitioning *,
html.theme-transitioning *::before,
html.theme-transitioning *::after {
  transition: background-color 300ms var(--spring-smooth),
              color 300ms var(--spring-smooth),
              border-color 300ms var(--spring-smooth),
              box-shadow 300ms var(--spring-smooth) !important;
}
```

**Step 4: Update ThemeToggle to trigger theme transition class**

Modify `/mnt/d/git/feednest/frontend/src/lib/stores/settings.ts` — in `setTheme()`, add class toggle:

```typescript
setTheme(theme: Theme) {
  document.documentElement.classList.add('theme-transitioning');
  // ... existing theme logic ...
  setTimeout(() => {
    document.documentElement.classList.remove('theme-transitioning');
  }, 350);
}
```

**Step 5: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`
Expected: No errors

```bash
git add -A && git commit -m "feat: add spring motion system foundation with CSS custom properties"
```

---

### Task 2: Viewport Fade-In with Intersection Observer

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/utils/viewport.ts`
- Modify: `/mnt/d/git/feednest/frontend/src/app.css`

**Step 1: Create viewport observer utility**

```typescript
// viewport.ts
export function viewportFadeIn(node: HTMLElement, options?: { delay?: number }) {
  const delay = options?.delay ?? 0;

  // Check reduced motion preference
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    node.style.opacity = '1';
    node.style.transform = 'none';
    return { destroy() {} };
  }

  node.style.opacity = '0';
  node.style.transform = 'translateY(12px)';
  node.style.transition = `opacity var(--duration-smooth) var(--spring-smooth) ${delay}ms, transform var(--duration-smooth) var(--spring-smooth) ${delay}ms`;

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          node.style.opacity = '1';
          node.style.transform = 'translateY(0)';
          observer.unobserve(node);
        }
      });
    },
    { threshold: 0.1 }
  );

  observer.observe(node);

  return {
    destroy() {
      observer.disconnect();
    }
  };
}
```

**Step 2: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add viewport fade-in Svelte action using Intersection Observer"
```

---

### Task 3: Scroll-Linked Header Collapse

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/routes/+page.svelte` (header section, lines ~320-430)

**Step 1: Add scroll tracking state**

Add to the reactive state block (around line 19-44):

```typescript
let scrollY = $state(0);
let headerCompact = $derived(scrollY > 80);
```

Add a scroll listener to the main content area (not window, to avoid sidebar interference). Find the main scrollable container and add:

```svelte
<svelte:window bind:scrollY={scrollY} />
```

**Step 2: Update header markup for compact/full modes**

The sticky header (around line 320) should transition between states:

```svelte
<header
  class="sticky top-0 z-30 glass transition-all {headerCompact ? 'py-2' : 'py-3'}"
  style="transition: padding var(--duration-snappy) var(--spring-snappy);"
>
```

When `headerCompact` is true:
- Hide the filter tabs row (fade out)
- Compress vertical padding
- Show a condensed breadcrumb (active feed/category name + article count)

When `headerCompact` is false:
- Full header with all controls visible

The filter tabs section should get:
```svelte
<div
  class="transition-all overflow-hidden"
  style="max-height: {headerCompact ? '0px' : '60px'}; opacity: {headerCompact ? 0 : 1}; transition: max-height var(--duration-snappy) var(--spring-snappy), opacity var(--duration-snappy) var(--spring-snappy);"
>
  <!-- existing filter tabs -->
</div>
```

**Step 3: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add scroll-linked header collapse with spring animation"
```

---

### Task 4: Parallax Hero Cards

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/utils/parallax.ts`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleCard.svelte`

**Step 1: Create parallax Svelte action**

```typescript
// parallax.ts
export function parallax(node: HTMLElement, options?: { rate?: number }) {
  const rate = options?.rate ?? 0.3;

  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return { destroy() {} };
  }

  const img = node.querySelector('img') as HTMLElement | null;
  if (!img) return { destroy() {} };

  // Ensure overflow hidden on container and extra height on image
  node.style.overflow = 'hidden';
  img.style.willChange = 'transform';

  function onScroll() {
    const rect = node.getBoundingClientRect();
    const viewportHeight = window.innerHeight;
    if (rect.bottom < 0 || rect.top > viewportHeight) return;

    const centerOffset = rect.top - viewportHeight / 2;
    const translateY = centerOffset * rate;
    img.style.transform = `translateY(${translateY}px) scale(1.1)`;
  }

  window.addEventListener('scroll', onScroll, { passive: true });
  onScroll();

  return {
    destroy() {
      window.removeEventListener('scroll', onScroll);
    }
  };
}
```

**Step 2: Create magnetic hover Svelte action**

Add to same file:

```typescript
export function magneticHover(node: HTMLElement, options?: { strength?: number }) {
  const strength = options?.strength ?? 5;

  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return { destroy() {} };
  }

  const img = node.querySelector('img') as HTMLElement | null;
  if (!img) return { destroy() {} };

  function onMouseMove(e: MouseEvent) {
    const rect = node.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width - 0.5) * strength;
    const y = ((e.clientY - rect.top) / rect.height - 0.5) * strength;
    img.style.transform = `translate(${x}px, ${y}px) scale(1.05)`;
  }

  function onMouseLeave() {
    img.style.transform = 'translate(0, 0) scale(1)';
    img.style.transition = `transform var(--duration-snappy) var(--spring-snappy)`;
  }

  function onMouseEnter() {
    img.style.transition = 'none';
  }

  node.addEventListener('mouseenter', onMouseEnter);
  node.addEventListener('mousemove', onMouseMove);
  node.addEventListener('mouseleave', onMouseLeave);

  return {
    destroy() {
      node.removeEventListener('mouseenter', onMouseEnter);
      node.removeEventListener('mousemove', onMouseMove);
      node.removeEventListener('mouseleave', onMouseLeave);
    }
  };
}
```

**Step 3: Apply parallax and magnetic hover to ArticleCard**

In `ArticleCard.svelte`, import and use:

```svelte
<script lang="ts">
  import { parallax, magneticHover } from '$lib/utils/parallax';
  // ... existing code
</script>

<!-- On the card container with an image -->
<div use:parallax={{ rate: 0.3 }} use:magneticHover={{ strength: 5 }}>
  <!-- existing image markup -->
</div>
```

**Step 4: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add parallax scroll and magnetic hover effects on article cards"
```

---

### Task 5: Enhanced Keyboard System & Vim Navigation

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/lib/utils/keyboard.ts` (22 lines → ~80 lines)
- Modify: `/mnt/d/git/feednest/frontend/src/routes/+page.svelte` (keyboard section, lines 178-237)

**Step 1: Upgrade keyboard utility with chord support and Cmd+K detection**

Rewrite `keyboard.ts`:

```typescript
export interface KeyboardShortcuts {
  [key: string]: (e: KeyboardEvent) => void;
}

export function setupKeyboardShortcuts(shortcuts: KeyboardShortcuts) {
  let chordBuffer = '';
  let chordTimeout: ReturnType<typeof setTimeout> | null = null;

  const handler = (e: KeyboardEvent) => {
    const target = e.target as HTMLElement;
    const tag = target.tagName;
    const isInput = tag === 'INPUT' || tag === 'TEXTAREA' || target.isContentEditable;

    // Cmd+K / Ctrl+K always fires (even in inputs)
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      shortcuts['cmd+k']?.(e);
      return;
    }

    // ? always fires for help overlay (unless in input)
    if (e.key === '?' && !isInput) {
      e.preventDefault();
      shortcuts['?']?.(e);
      return;
    }

    if (isInput) return;

    const key = e.key.toLowerCase();

    // Handle chord sequences (e.g., "gg")
    if (chordBuffer) {
      const chord = chordBuffer + key;
      if (chordTimeout) clearTimeout(chordTimeout);
      chordBuffer = '';
      if (shortcuts[chord]) {
        e.preventDefault();
        shortcuts[chord](e);
        return;
      }
    }

    // Check if this key starts a chord
    const possibleChords = Object.keys(shortcuts).filter(
      (k) => k.length === 2 && k.startsWith(key) && !k.includes('+')
    );

    if (possibleChords.length > 0) {
      chordBuffer = key;
      chordTimeout = setTimeout(() => {
        // No chord completed — fire single key if exists
        if (shortcuts[chordBuffer]) {
          shortcuts[chordBuffer]({} as KeyboardEvent);
        }
        chordBuffer = '';
      }, 300);

      // Don't fire single key yet — wait for potential chord
      if (shortcuts[key] && possibleChords.length > 0) {
        e.preventDefault();
        return;
      }
    }

    if (shortcuts[key]) {
      e.preventDefault();
      shortcuts[key](e);
    }
  };

  document.addEventListener('keydown', handler);
  return () => document.removeEventListener('keydown', handler);
}
```

**Step 2: Add new shortcuts to +page.svelte**

Extend the shortcuts object in `setupKeyboardShortcuts()` call (lines 178-237):

Add these new shortcuts:
- `'cmd+k'`: toggle command palette (Task 6)
- `'?'`: toggle keyboard hints overlay (Task 7)
- `'gg'`: jump to top of list → `selectedIndex = 0; scrollSelectedIntoView()`
- `'shift+g'` or `'G'`: jump to bottom → `selectedIndex = articles.length - 1`
- `'r'`: refresh feeds → `feeds.load()`
- `'1'`: set view mode hybrid
- `'2'`: set view mode cards
- `'3'`: set view mode list

**Step 3: Add animated selection cursor**

Add to `app.css`:
```css
.selection-cursor {
  position: relative;
}
.selection-cursor::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: linear-gradient(to bottom, var(--color-accent), var(--color-accent-end));
  border-radius: 2px;
  transition: top var(--duration-snappy) var(--spring-snappy),
              height var(--duration-snappy) var(--spring-snappy);
}
```

**Step 4: Add smooth scroll-into-view helper**

In `+page.svelte`, add helper function:

```typescript
function scrollSelectedIntoView() {
  const el = document.querySelector(`[data-article-index="${selectedIndex}"]`);
  el?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}
```

Update `j`/`k` handlers to call `scrollSelectedIntoView()` after index change.

**Step 5: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: upgrade keyboard system with chord support, G/gg/r/1-2-3 shortcuts"
```

---

### Task 6: Command Palette

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/components/CommandPalette.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/routes/+page.svelte` (add component + toggle state)

**Step 1: Create CommandPalette component**

```svelte
<script lang="ts">
  import { articles } from '$lib/stores/articles';
  import { feeds, categories } from '$lib/stores/feeds';

  interface CommandItem {
    id: string;
    label: string;
    category: 'navigate' | 'article' | 'action';
    icon: string;
    shortcut?: string;
    action: () => void;
  }

  let { open = $bindable(false), onSelectFeed, onSelectCategory, onSelectAll, onRefresh, onToggleTheme }: {
    open: boolean;
    onSelectFeed?: (id: number) => void;
    onSelectCategory?: (id: number) => void;
    onSelectAll?: () => void;
    onRefresh?: () => void;
    onToggleTheme?: () => void;
  } = $props();

  let query = $state('');
  let selectedIdx = $state(0);
  let inputEl: HTMLInputElement;

  // Build command list
  let commands = $derived.by(() => {
    const items: CommandItem[] = [];

    // Navigation commands
    items.push({
      id: 'nav-all',
      label: 'All Articles',
      category: 'navigate',
      icon: '📰',
      action: () => { onSelectAll?.(); close(); }
    });

    for (const cat of $categories) {
      items.push({
        id: `nav-cat-${cat.id}`,
        label: `Category: ${cat.name}`,
        category: 'navigate',
        icon: '📁',
        action: () => { onSelectCategory?.(cat.id); close(); }
      });
    }

    for (const feed of $feeds) {
      items.push({
        id: `nav-feed-${feed.id}`,
        label: feed.title,
        category: 'navigate',
        icon: '🔗',
        action: () => { onSelectFeed?.(feed.id); close(); }
      });
    }

    // Action commands
    items.push({
      id: 'action-refresh',
      label: 'Refresh Feeds',
      category: 'action',
      icon: '🔄',
      shortcut: 'r',
      action: () => { onRefresh?.(); close(); }
    });

    items.push({
      id: 'action-theme',
      label: 'Toggle Theme',
      category: 'action',
      icon: '🎨',
      action: () => { onToggleTheme?.(); close(); }
    });

    // Filter by query
    if (!query.trim()) return items;
    const q = query.toLowerCase();
    return items.filter((item) => item.label.toLowerCase().includes(q));
  });

  $effect(() => {
    if (open) {
      query = '';
      selectedIdx = 0;
      requestAnimationFrame(() => inputEl?.focus());
    }
  });

  // Reset selection when filtered results change
  $effect(() => {
    if (commands.length > 0 && selectedIdx >= commands.length) {
      selectedIdx = 0;
    }
  });

  function close() {
    open = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedIdx = (selectedIdx + 1) % commands.length;
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedIdx = (selectedIdx - 1 + commands.length) % commands.length;
    } else if (e.key === 'Enter') {
      e.preventDefault();
      commands[selectedIdx]?.action();
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-[100] flex items-start justify-center pt-[20vh]"
    onkeydown={handleKeydown}
  >
    <!-- Backdrop -->
    <div
      class="absolute inset-0 bg-black/50 backdrop-blur-sm"
      onclick={close}
      role="presentation"
    ></div>

    <!-- Palette -->
    <div
      class="relative w-full max-w-lg mx-4 rounded-2xl glass border border-[var(--color-border)] shadow-2xl overflow-hidden fade-in-up"
      style="animation-duration: var(--duration-snappy);"
    >
      <!-- Search input -->
      <div class="flex items-center gap-3 px-4 py-3 border-b border-[var(--color-border)]">
        <svg class="w-5 h-5 text-[var(--color-text-tertiary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input
          bind:this={inputEl}
          bind:value={query}
          type="text"
          placeholder="Type a command or search..."
          class="flex-1 bg-transparent text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] outline-none text-base"
        />
        <kbd class="px-2 py-0.5 text-xs rounded bg-[var(--color-surface)] text-[var(--color-text-tertiary)] border border-[var(--color-border)]">Esc</kbd>
      </div>

      <!-- Results -->
      <div class="max-h-80 overflow-y-auto py-2">
        {#if commands.length === 0}
          <div class="px-4 py-8 text-center text-[var(--color-text-tertiary)]">
            No results found
          </div>
        {:else}
          {#each commands as item, i}
            <button
              class="w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors {i === selectedIdx ? 'bg-[var(--color-accent-glow)] text-[var(--color-accent)]' : 'text-[var(--color-text-primary)] hover:bg-[var(--color-elevated)]'}"
              onclick={() => item.action()}
              onmouseenter={() => selectedIdx = i}
            >
              <span class="text-lg">{item.icon}</span>
              <span class="flex-1 text-sm font-medium">{item.label}</span>
              {#if item.shortcut}
                <kbd class="px-1.5 py-0.5 text-xs rounded bg-[var(--color-surface)] text-[var(--color-text-tertiary)] border border-[var(--color-border)]">{item.shortcut}</kbd>
              {/if}
              <span class="text-xs text-[var(--color-text-tertiary)] capitalize">{item.category}</span>
            </button>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}
```

**Step 2: Wire into +page.svelte**

Add state: `let commandPaletteOpen = $state(false);`

Add to keyboard shortcuts:
```typescript
'cmd+k': () => { commandPaletteOpen = !commandPaletteOpen; }
```

Add component in template (after modals):
```svelte
<CommandPalette
  bind:open={commandPaletteOpen}
  onSelectFeed={(id) => selectFeed(id)}
  onSelectCategory={(id) => selectCategory(id)}
  onSelectAll={() => selectAll()}
  onRefresh={() => refreshFeeds()}
  onToggleTheme={() => { /* cycle theme */ }}
/>
```

**Step 3: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add command palette with fuzzy search and keyboard navigation"
```

---

### Task 7: Keyboard Hints Overlay

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/components/KeyboardHints.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/routes/+page.svelte`

**Step 1: Create KeyboardHints component**

```svelte
<script lang="ts">
  let { open = $bindable(false) }: { open: boolean } = $props();

  const groups = [
    {
      title: 'Navigation',
      shortcuts: [
        { keys: ['j'], desc: 'Next article' },
        { keys: ['k'], desc: 'Previous article' },
        { keys: ['o', 'Enter'], desc: 'Open article' },
        { keys: ['Esc'], desc: 'Close / deselect' },
        { keys: ['g g'], desc: 'Jump to top' },
        { keys: ['G'], desc: 'Jump to bottom' },
      ]
    },
    {
      title: 'Actions',
      shortcuts: [
        { keys: ['s'], desc: 'Toggle star' },
        { keys: ['m'], desc: 'Toggle read/unread' },
        { keys: ['d'], desc: 'Dismiss article' },
        { keys: ['r'], desc: 'Refresh feeds' },
      ]
    },
    {
      title: 'Views & Search',
      shortcuts: [
        { keys: ['/'], desc: 'Focus search' },
        { keys: ['1'], desc: 'Hybrid view' },
        { keys: ['2'], desc: 'Card view' },
        { keys: ['3'], desc: 'List view' },
        { keys: ['Cmd+K'], desc: 'Command palette' },
        { keys: ['?'], desc: 'Toggle this help' },
      ]
    },
  ];

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' || e.key === '?') {
      e.preventDefault();
      open = false;
    }
  }
</script>

{#if open}
  <div class="fixed inset-0 z-[100] flex items-center justify-center" onkeydown={handleKeydown}>
    <div class="absolute inset-0 bg-black/50 backdrop-blur-sm" onclick={() => open = false} role="presentation"></div>
    <div class="relative w-full max-w-md mx-4 rounded-2xl glass border border-[var(--color-border)] shadow-2xl p-6 fade-in-up" style="animation-duration: var(--duration-snappy);">
      <h2 class="text-lg font-bold text-[var(--color-text-primary)] mb-4">Keyboard Shortcuts</h2>
      {#each groups as group}
        <h3 class="text-xs font-semibold text-[var(--color-text-tertiary)] uppercase tracking-wider mt-4 mb-2">{group.title}</h3>
        <div class="space-y-1.5">
          {#each group.shortcuts as shortcut}
            <div class="flex items-center justify-between">
              <span class="text-sm text-[var(--color-text-secondary)]">{shortcut.desc}</span>
              <div class="flex gap-1">
                {#each shortcut.keys as key}
                  <kbd class="px-2 py-0.5 text-xs rounded bg-[var(--color-surface)] text-[var(--color-text-tertiary)] border border-[var(--color-border)] font-mono">{key}</kbd>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {/each}
    </div>
  </div>
{/if}
```

**Step 2: Wire into +page.svelte**

Add state: `let keyboardHintsOpen = $state(false);`

Add to keyboard shortcuts:
```typescript
'?': () => { keyboardHintsOpen = !keyboardHintsOpen; }
```

Add component in template.

**Step 3: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add keyboard hints overlay with ? shortcut"
```

---

### Task 8: Reading Progress Bar

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleReader.svelte` (267 lines)
- Modify: `/mnt/d/git/feednest/frontend/src/app.css` (already has `.reading-progress` class)

**Step 1: Add scroll progress tracking to ArticleReader**

Add state and scroll handler in the script section:

```typescript
let readingProgress = $state(0);
let contentEl: HTMLElement;

function updateReadingProgress() {
  if (!contentEl) return;
  const rect = contentEl.getBoundingClientRect();
  const windowHeight = window.innerHeight;
  const totalScrollable = rect.height - windowHeight;
  if (totalScrollable <= 0) { readingProgress = 100; return; }
  const scrolled = -rect.top;
  readingProgress = Math.min(100, Math.max(0, (scrolled / totalScrollable) * 100));
}
```

**Step 2: Add progress bar and scroll listener to template**

At the very top of the reader panel (inside the fixed container, before the header):

```svelte
<!-- Reading progress bar -->
{#if visible && !loading}
  <div
    class="fixed top-0 left-0 right-0 z-[60] h-0.5 reading-progress"
    style="width: {readingProgress}%; transition: width 100ms linear; opacity: {readingProgress > 0 ? 1 : 0};"
  ></div>
{/if}
```

Add scroll listener to the scrollable content wrapper:
```svelte
<div bind:this={contentEl} onscroll={updateReadingProgress}>
```

Or use a `$effect` with scroll event on the panel's scrollable area.

**Step 3: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add reading progress bar to article reader"
```

---

### Task 9: Collapsing Reader Header

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleReader.svelte`

**Step 1: Add scroll-direction tracking**

```typescript
let readerScrollY = $state(0);
let lastScrollY = 0;
let readerHeaderCompact = $state(false);

function handleReaderScroll(e: Event) {
  const target = e.target as HTMLElement;
  readerScrollY = target.scrollTop;

  if (readerScrollY > 120 && readerScrollY > lastScrollY) {
    readerHeaderCompact = true;
  } else if (readerScrollY < lastScrollY) {
    readerHeaderCompact = false;
  }
  lastScrollY = readerScrollY;

  updateReadingProgress();
}
```

**Step 2: Update reader header markup**

The reader header (around line 105-140) should transition:

```svelte
<header
  class="sticky top-0 z-10 glass border-b border-[var(--color-border)] transition-all"
  style="padding: {readerHeaderCompact ? '8px 16px' : '12px 16px'}; transition: padding var(--duration-snappy) var(--spring-snappy);"
>
  <div class="flex items-center gap-3">
    <!-- Back button (always visible) -->
    <button onclick={handleClose}>←</button>

    {#if readerHeaderCompact}
      <!-- Compact: truncated title + actions -->
      <span class="flex-1 text-sm font-medium truncate text-[var(--color-text-primary)]">
        {article?.title}
      </span>
    {:else}
      <!-- Full: feed name -->
      <span class="flex-1 text-sm text-[var(--color-text-secondary)]">
        {article?.feed_title}
      </span>
    {/if}

    <!-- Star + Mark read buttons (always visible) -->
    <button onclick={handleStar}><!-- star icon --></button>
  </div>
</header>
```

**Step 3: Wire scroll handler to panel's scrollable container**

Add `onscroll={handleReaderScroll}` to the main scrollable div in the reader.

**Step 4: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add collapsing reader header on scroll direction"
```

---

### Task 10: Article Navigation Within Reader (j/k)

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleReader.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/routes/+page.svelte`

**Step 1: Pass article list context to reader**

Add props to ArticleReader:

```typescript
let {
  articleId,
  onClose,
  articleIds = [],
  onNavigate
}: {
  articleId: number;
  onClose: () => void;
  articleIds?: number[];
  onNavigate?: (id: number) => void;
} = $props();
```

**Step 2: Add j/k navigation in reader's keydown handler**

Update `handleKeydown()` in ArticleReader:

```typescript
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    handleClose();
  } else if (e.key === 'j' || e.key === 'k') {
    const currentIdx = articleIds.indexOf(articleId);
    if (currentIdx === -1) return;
    const nextIdx = e.key === 'j' ? currentIdx + 1 : currentIdx - 1;
    if (nextIdx >= 0 && nextIdx < articleIds.length) {
      e.preventDefault();
      onNavigate?.(articleIds[nextIdx]);
    }
  }
}
```

**Step 3: Add slide transition between articles**

When `articleId` changes, animate: current slides out left, new slides in from right.

```typescript
let slideDirection = $state<'left' | 'right' | null>(null);

// Watch for articleId changes
$effect(() => {
  if (articleId) {
    slideDirection = 'right'; // new article slides in from right
    // Reset after animation
    setTimeout(() => { slideDirection = null; }, 300);
  }
});
```

Add CSS class:
```css
.slide-in-right {
  animation: slideFromRight var(--duration-smooth) var(--spring-smooth);
}

@keyframes slideFromRight {
  from { transform: translateX(40px); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}
```

**Step 4: Pass articleIds from +page.svelte**

In the main page, pass the list of currently displayed article IDs:

```svelte
<ArticleReader
  articleId={selectedArticleId}
  onClose={closeArticle}
  articleIds={displayedArticles.map(a => a.id)}
  onNavigate={(id) => { selectedArticleId = id; }}
/>
```

**Step 5: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add j/k article navigation within reader with slide transitions"
```

---

### Task 11: Dynamic Feed Accent Colors

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/utils/color.ts`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleCard.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleList.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/Sidebar.svelte`

**Step 1: Create color extraction utility**

```typescript
// color.ts

// Cache extracted colors to avoid re-processing
const colorCache = new Map<string, string>();

export function hashToColor(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = Math.abs(hash) % 360;
  return `hsl(${hue}, 65%, 55%)`;
}

export function extractDominantColor(imageUrl: string): Promise<string> {
  if (colorCache.has(imageUrl)) {
    return Promise.resolve(colorCache.get(imageUrl)!);
  }

  return new Promise((resolve) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';

    img.onload = () => {
      try {
        const canvas = document.createElement('canvas');
        canvas.width = 16;
        canvas.height = 16;
        const ctx = canvas.getContext('2d');
        if (!ctx) { resolve(hashToColor(imageUrl)); return; }

        ctx.drawImage(img, 0, 0, 16, 16);
        const data = ctx.getImageData(0, 0, 16, 16).data;

        let r = 0, g = 0, b = 0, count = 0;
        for (let i = 0; i < data.length; i += 4) {
          // Skip very dark and very light pixels
          const brightness = (data[i] + data[i + 1] + data[i + 2]) / 3;
          if (brightness > 30 && brightness < 225) {
            r += data[i];
            g += data[i + 1];
            b += data[i + 2];
            count++;
          }
        }

        if (count === 0) { resolve(hashToColor(imageUrl)); return; }

        const color = `rgb(${Math.round(r / count)}, ${Math.round(g / count)}, ${Math.round(b / count)})`;
        colorCache.set(imageUrl, color);
        resolve(color);
      } catch {
        resolve(hashToColor(imageUrl));
      }
    };

    img.onerror = () => resolve(hashToColor(imageUrl));
    img.src = imageUrl;
  });
}

export function getFeedColor(iconUrl?: string, feedUrl?: string): Promise<string> {
  if (iconUrl) return extractDominantColor(iconUrl);
  return Promise.resolve(hashToColor(feedUrl || 'default'));
}
```

**Step 2: Apply accent colors to article cards and list items**

In ArticleCard and ArticleList, add a reactive color:

```typescript
import { getFeedColor } from '$lib/utils/color';

let feedAccentColor = $state('');

$effect(() => {
  getFeedColor(article.icon_url, article.feed_url).then(c => { feedAccentColor = c; });
});
```

Use `feedAccentColor` for the left border / unread indicator:
```svelte
<div style="border-left-color: {feedAccentColor};" class="border-l-3">
```

**Step 3: Apply to sidebar feed items**

Similar pattern — each feed item gets a colored dot or left accent from its icon color.

**Step 4: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add dynamic feed accent colors from favicon extraction"
```

---

### Task 12: Animated Unread Badges (Odometer Effect)

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/components/AnimatedCount.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/Sidebar.svelte`

**Step 1: Create AnimatedCount component**

```svelte
<script lang="ts">
  let { value }: { value: number } = $props();

  let displayValue = $state(value);
  let digits = $derived(String(displayValue).split(''));
  let targetDigits = $derived(String(value).split(''));

  $effect(() => {
    // Animate digit by digit
    if (value !== displayValue) {
      const step = value > displayValue ? 1 : -1;
      const interval = setInterval(() => {
        displayValue += step;
        if (displayValue === value) clearInterval(interval);
      }, 40);
      return () => clearInterval(interval);
    }
  });
</script>

<span class="inline-flex overflow-hidden tabular-nums">
  {#each targetDigits as _, i}
    <span class="inline-block transition-transform" style="transition: transform var(--duration-snappy) var(--spring-snappy);">
      {digits[i] ?? '0'}
    </span>
  {/each}
</span>
```

**Step 2: Replace static unread counts in Sidebar with AnimatedCount**

Find all `{feed.unread_count}` and similar in Sidebar.svelte, replace with:
```svelte
<AnimatedCount value={feed.unread_count} />
```

**Step 3: Add pulse glow on count change**

When a feed has new articles (count increases), briefly apply the `pulseGlow` animation class.

**Step 4: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add animated odometer unread badges in sidebar"
```

---

### Task 13: Star Particle Burst & Confetti Celebrations

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/utils/particles.ts`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleCard.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleList.svelte`

**Step 1: Create particle system utility**

```typescript
// particles.ts

export function starBurst(x: number, y: number) {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

  const colors = ['#fbbf24', '#f59e0b', '#d97706', '#fcd34d', '#fef3c7'];
  const count = 6;

  for (let i = 0; i < count; i++) {
    const particle = document.createElement('div');
    const angle = (Math.PI * 2 * i) / count;
    const velocity = 30 + Math.random() * 20;
    const size = 4 + Math.random() * 3;

    Object.assign(particle.style, {
      position: 'fixed',
      left: `${x}px`,
      top: `${y}px`,
      width: `${size}px`,
      height: `${size}px`,
      borderRadius: '50%',
      background: colors[i % colors.length],
      pointerEvents: 'none',
      zIndex: '9999',
      transition: 'all 500ms cubic-bezier(0.22, 1, 0.36, 1)',
    });

    document.body.appendChild(particle);

    requestAnimationFrame(() => {
      particle.style.transform = `translate(${Math.cos(angle) * velocity}px, ${Math.sin(angle) * velocity}px)`;
      particle.style.opacity = '0';
    });

    setTimeout(() => particle.remove(), 600);
  }
}

export function confettiBurst() {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

  const colors = ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#06b6d4'];
  const count = 30;

  for (let i = 0; i < count; i++) {
    const particle = document.createElement('div');
    const x = window.innerWidth / 2 + (Math.random() - 0.5) * 300;
    const size = 6 + Math.random() * 4;

    Object.assign(particle.style, {
      position: 'fixed',
      left: `${x}px`,
      top: '-10px',
      width: `${size}px`,
      height: `${size * 1.5}px`,
      borderRadius: '2px',
      background: colors[Math.floor(Math.random() * colors.length)],
      pointerEvents: 'none',
      zIndex: '9999',
      transform: `rotate(${Math.random() * 360}deg)`,
      transition: `all ${800 + Math.random() * 400}ms cubic-bezier(0.22, 1, 0.36, 1)`,
    });

    document.body.appendChild(particle);

    requestAnimationFrame(() => {
      particle.style.top = `${window.innerHeight + 20}px`;
      particle.style.left = `${x + (Math.random() - 0.5) * 100}px`;
      particle.style.opacity = '0';
      particle.style.transform = `rotate(${Math.random() * 720}deg)`;
    });

    setTimeout(() => particle.remove(), 1500);
  }
}
```

**Step 2: Wire star burst to star button clicks**

In ArticleCard and ArticleList `handleStar` functions, add:

```typescript
import { starBurst } from '$lib/utils/particles';

function handleStarWithBurst(e: MouseEvent) {
  handleStar(e);
  if (!article.is_starred) { // Was not starred, now starring
    starBurst(e.clientX, e.clientY);
  }
}
```

**Step 3: Wire confetti to "mark all read" completion**

In +page.svelte, after bulk mark-read action succeeds:

```typescript
import { confettiBurst } from '$lib/utils/particles';

// After successful bulk mark-all-read
confettiBurst();
```

**Step 4: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add star particle burst and confetti celebrations"
```

---

### Task 14: Blur-Up Image Loading

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/utils/blurload.ts`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleCard.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleReader.svelte`

**Step 1: Create blur-up Svelte action**

```typescript
// blurload.ts
export function blurUp(node: HTMLImageElement) {
  if (node.complete) return { destroy() {} };

  node.style.filter = 'blur(10px)';
  node.style.transform = 'scale(1.05)';
  node.style.transition = `filter var(--duration-smooth) var(--spring-smooth), transform var(--duration-smooth) var(--spring-smooth)`;

  function onLoad() {
    node.style.filter = 'blur(0)';
    node.style.transform = 'scale(1)';
  }

  function onError() {
    node.style.filter = 'blur(0)';
    node.style.transform = 'scale(1)';
  }

  node.addEventListener('load', onLoad);
  node.addEventListener('error', onError);

  return {
    destroy() {
      node.removeEventListener('load', onLoad);
      node.removeEventListener('error', onError);
    }
  };
}
```

**Step 2: Apply to all article images**

In ArticleCard hero images and ArticleReader hero images:

```svelte
<img use:blurUp src={article.image_url} alt="" />
```

**Step 3: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add blur-up image loading effect"
```

---

### Task 15: Shared Element Transitions (View Transitions API)

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleCard.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleList.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleReader.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/app.css`

**Step 1: Add view-transition-name to article cards**

In ArticleCard, on the card container:
```svelte
<div style="view-transition-name: article-{article.id};">
```

In ArticleList, on the list item:
```svelte
<div style="view-transition-name: article-{article.id};">
```

**Step 2: Add view-transition-name to reader**

In ArticleReader, on the main panel:
```svelte
<div style="view-transition-name: article-{articleId};">
```

**Step 3: Use document.startViewTransition when opening/closing articles**

In +page.svelte `openArticle` function:

```typescript
function openArticle(id: number) {
  if (document.startViewTransition) {
    document.startViewTransition(() => {
      selectedArticleId = id;
    });
  } else {
    selectedArticleId = id;
  }
}
```

Similarly for `closeArticle`.

**Step 4: Add view transition CSS**

In `app.css`:
```css
::view-transition-old(*),
::view-transition-new(*) {
  animation-duration: var(--duration-smooth);
  animation-timing-function: var(--spring-smooth);
}

::view-transition-old(root) {
  animation: none;
}

::view-transition-new(root) {
  animation: none;
}
```

**Step 5: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add shared element transitions using View Transitions API"
```

---

### Task 16: Mobile Swipe Gestures

**Files:**
- Create: `/mnt/d/git/feednest/frontend/src/lib/utils/swipe.ts`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleList.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleCard.svelte`

**Step 1: Create swipe gesture Svelte action**

```typescript
// swipe.ts

interface SwipeOptions {
  onSwipeRight?: () => void;
  onSwipeLeft?: () => void;
  threshold?: number;
}

export function swipeable(node: HTMLElement, options: SwipeOptions) {
  const threshold = options.threshold ?? 80;
  let startX = 0;
  let startY = 0;
  let currentX = 0;
  let isDragging = false;

  // Create action indicator elements
  const leftIndicator = document.createElement('div');
  const rightIndicator = document.createElement('div');

  leftIndicator.innerHTML = '✓';
  rightIndicator.innerHTML = '★';

  const indicatorStyle = {
    position: 'absolute',
    top: '0',
    bottom: '0',
    width: '60px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '20px',
    opacity: '0',
    transition: 'opacity 100ms',
    zIndex: '-1',
  };

  Object.assign(leftIndicator.style, { ...indicatorStyle, left: '0', background: 'rgba(34, 197, 94, 0.2)', color: '#22c55e' });
  Object.assign(rightIndicator.style, { ...indicatorStyle, right: '0', background: 'rgba(250, 204, 21, 0.2)', color: '#facc15' });

  node.style.position = 'relative';
  node.appendChild(leftIndicator);
  node.appendChild(rightIndicator);

  function onTouchStart(e: TouchEvent) {
    startX = e.touches[0].clientX;
    startY = e.touches[0].clientY;
    currentX = 0;
    isDragging = false;
    node.style.transition = 'none';
  }

  function onTouchMove(e: TouchEvent) {
    const dx = e.touches[0].clientX - startX;
    const dy = e.touches[0].clientY - startY;

    // Only horizontal swipe
    if (!isDragging && Math.abs(dy) > Math.abs(dx)) return;
    isDragging = true;

    currentX = dx;
    const dampened = currentX * 0.6;
    node.style.transform = `translateX(${dampened}px)`;

    // Show indicators
    leftIndicator.style.opacity = currentX > 20 ? '1' : '0';
    rightIndicator.style.opacity = currentX < -20 ? '1' : '0';

    // Scale indicator at threshold
    if (Math.abs(currentX) > threshold) {
      const indicator = currentX > 0 ? leftIndicator : rightIndicator;
      indicator.style.transform = 'scale(1.2)';
    }
  }

  function onTouchEnd() {
    node.style.transition = `transform var(--duration-snappy) var(--spring-dramatic)`;
    node.style.transform = 'translateX(0)';

    leftIndicator.style.opacity = '0';
    rightIndicator.style.opacity = '0';
    leftIndicator.style.transform = 'scale(1)';
    rightIndicator.style.transform = 'scale(1)';

    if (currentX > threshold) {
      options.onSwipeRight?.();
    } else if (currentX < -threshold) {
      options.onSwipeLeft?.();
    }

    isDragging = false;
  }

  node.addEventListener('touchstart', onTouchStart, { passive: true });
  node.addEventListener('touchmove', onTouchMove, { passive: false });
  node.addEventListener('touchend', onTouchEnd);

  return {
    destroy() {
      node.removeEventListener('touchstart', onTouchStart);
      node.removeEventListener('touchmove', onTouchMove);
      node.removeEventListener('touchend', onTouchEnd);
      leftIndicator.remove();
      rightIndicator.remove();
    },
    update(newOptions: SwipeOptions) {
      options = newOptions;
    }
  };
}
```

**Step 2: Apply to ArticleList items**

```svelte
<div use:swipeable={{
  onSwipeRight: () => toggleRead(article.id, !article.is_read),
  onSwipeLeft: () => toggleStar(article.id, !article.is_starred),
  threshold: 80
}}>
  <!-- existing article list item content -->
</div>
```

**Step 3: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: add mobile swipe gestures for mark-read and star"
```

---

### Task 17: Enhanced Empty States

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/routes/+page.svelte` (empty state sections, around lines 460-500)
- Modify: `/mnt/d/git/feednest/frontend/src/app.css`

**Step 1: Add floating animation keyframe**

In `app.css`:
```css
@keyframes gentleFloat {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}

.animate-float {
  animation: gentleFloat 3s ease-in-out infinite;
}

@media (prefers-reduced-motion: reduce) {
  .animate-float {
    animation: none;
  }
}
```

**Step 2: Update empty state markup in +page.svelte**

Replace static empty state icons with animated versions:

```svelte
<div class="flex flex-col items-center justify-center py-20 text-center fade-in-up">
  <div class="animate-float mb-6">
    <div class="w-20 h-20 rounded-2xl accent-gradient flex items-center justify-center shadow-lg">
      <svg class="w-10 h-10 text-white" ...><!-- RSS icon --></svg>
    </div>
  </div>
  <h3 class="text-xl font-bold text-[var(--color-text-primary)] mb-2">
    {emptyStateTitle}
  </h3>
  <p class="text-[var(--color-text-secondary)] mb-6 max-w-sm">
    {emptyStateMessage}
  </p>
  <!-- CTA button if applicable -->
</div>
```

**Step 3: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: enhance empty states with floating animation"
```

---

### Task 18: Staggered Spring Entrance Animations

**Files:**
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleCard.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/lib/components/ArticleList.svelte`
- Modify: `/mnt/d/git/feednest/frontend/src/app.css`

**Step 1: Update entrance animation to use spring timing**

In `app.css`, update the `fadeInUp` keyframe and class:

```css
@keyframes springIn {
  0% {
    opacity: 0;
    transform: translateY(16px) scale(0.98);
  }
  100% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.spring-in {
  animation: springIn var(--duration-smooth) var(--spring-dramatic) both;
}
```

**Step 2: Update ArticleCard stagger timing**

Change from `fade-in-up` with `animation-delay: {index * 60}ms` to:
```svelte
<div class="spring-in" style="animation-delay: {index * 40}ms;">
```

**Step 3: Update ArticleList stagger timing**

Change from `fade-in-up` with `animation-delay: {index * 30}ms` to:
```svelte
<div class="spring-in" style="animation-delay: {index * 30}ms;">
```

**Step 4: Verify and commit**

Run: `cd /mnt/d/git/feednest/frontend && npm run check`

```bash
git add -A && git commit -m "feat: upgrade entrance animations to spring physics"
```

---

### Task 19: Integration Testing & Polish Pass

**Files:**
- All modified files from Tasks 1-18

**Step 1: Run full TypeScript/Svelte check**

```bash
cd /mnt/d/git/feednest/frontend && npm run check
```

Fix any type errors.

**Step 2: Build test**

```bash
cd /mnt/d/git/feednest/frontend && npm run build
```

Fix any build errors.

**Step 3: Visual review in browser**

```bash
cd /mnt/d/git/feednest && docker compose up --build -d
```

Test checklist:
- [ ] Spring animations feel natural (not jarring)
- [ ] Cmd+K opens palette, fuzzy search works
- [ ] j/k navigation moves selection with visible cursor
- [ ] gg/G jump to top/bottom
- [ ] ? shows keyboard hints
- [ ] 1/2/3 switches views
- [ ] Reader opens with progress bar
- [ ] Reader header collapses on scroll down, expands on scroll up
- [ ] j/k navigates between articles in reader
- [ ] Feed accent colors show on cards/list
- [ ] Star button creates particle burst
- [ ] Unread badges animate count changes
- [ ] Images blur-up on load
- [ ] Theme toggle crossfades smoothly
- [ ] Empty states have floating animation
- [ ] Reduced motion preference disables all animations
- [ ] Mobile: swipe right marks read, swipe left stars
- [ ] View transitions work when opening/closing articles (Chrome)
- [ ] Fallback works in Firefox/Safari

**Step 4: Fix any issues found and commit**

```bash
git add -A && git commit -m "fix: polish pass and integration fixes for Editorial Engine"
```

---

## Summary

| Task | Description | New Files | Modified Files |
|------|-------------|-----------|----------------|
| 1 | Spring motion system | — | app.css, settings.ts |
| 2 | Viewport fade-in | viewport.ts | — |
| 3 | Scroll-linked header | — | +page.svelte |
| 4 | Parallax hero cards | parallax.ts | ArticleCard.svelte |
| 5 | Enhanced keyboard | — | keyboard.ts, +page.svelte |
| 6 | Command palette | CommandPalette.svelte | +page.svelte |
| 7 | Keyboard hints | KeyboardHints.svelte | +page.svelte |
| 8 | Reading progress bar | — | ArticleReader.svelte |
| 9 | Collapsing reader header | — | ArticleReader.svelte |
| 10 | Article nav in reader | — | ArticleReader.svelte, +page.svelte |
| 11 | Dynamic feed colors | color.ts | ArticleCard, ArticleList, Sidebar |
| 12 | Animated unread badges | AnimatedCount.svelte | Sidebar.svelte |
| 13 | Particle celebrations | particles.ts | ArticleCard, ArticleList, +page |
| 14 | Blur-up images | blurload.ts | ArticleCard, ArticleReader |
| 15 | Shared element transitions | — | ArticleCard, ArticleList, ArticleReader, app.css |
| 16 | Mobile swipe gestures | swipe.ts | ArticleList, ArticleCard |
| 17 | Enhanced empty states | — | +page.svelte, app.css |
| 18 | Spring entrance animations | — | ArticleCard, ArticleList, app.css |
| 19 | Integration & polish | — | All |

**New files:** 8 (5 utilities, 3 components)
**Modified files:** 10 existing files
**Estimated commits:** 19
