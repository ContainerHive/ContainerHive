# Design System Specification: ContainerHive

## 1. Overview & Creative North Star

**Creative North Star: "The Precision Monolith"**

ContainerHive's UI rejects the cluttered, Bootstrap-boxy aesthetic that plagues most container registries. Instead, we operate as a **precision instrument**—a high-density, editorialized technical environment where data is curated rather than merely displayed.

The system bridges two worlds: a **light theme** ("The Precision Architect") built for long-form reading and cognitive clarity, and a **dark theme** ("The Observability Monolith") carved from deep slate for terminal-native developers who live in the dark. Both themes share the same structural DNA: intentional asymmetry, tonal layering, and the "No-Line" rule.

We don't just show SHA hashes and image tags. We present technical truth with authority.

---

## 2. Themes

ContainerHive ships with two first-class themes. All token references below are scoped per theme.

### 2.1 Light Theme — "The Precision Architect"

A hyper-clean, light-flooded aesthetic that prioritizes mental clarity. White space is treated as a **structural element**, not a void. Designed for the developer who reads documentation and diffs in broad daylight.

| Token | Value | Role |
|---|---|---|
| `surface` | `#f7f9fb` | Base canvas for the entire application |
| `surface_container_low` | `#f2f4f6` | Sidebars, secondary navigation |
| `surface_container_lowest` | `#ffffff` | Primary content cards, code blocks ("pop" layer) |
| `surface_container` | `#edf0f2` | Mid-tier grouping containers |
| `surface_container_high` | `#e6e8ea` | Hover states, active selection backgrounds |
| `surface_container_highest` | `#e0e3e5` | Code blocks, high-contrast technical zones |
| `primary` | `#006591` | Primary brand, CTAs, active accents |
| `primary_container` | `#0ea5e9` | Gradient endpoint, keyword highlights, status indicators |
| `on_primary` | `#ffffff` | Text on primary backgrounds |
| `on_primary_container` | `#003751` | Text on primary_container backgrounds |
| `secondary_container` | `#b8dffe` | Chip backgrounds |
| `on_secondary_container` | `#3d637d` | Chip text |
| `on_surface` | `#191c1e` | All body text (never pure black) |
| `on_surface_variant` | `#40484f` | Metadata, secondary labels |
| `outline_variant` | `#bec8d2` | Ghost border base (used at 20% opacity only) |
| `error` | `#ba1a1a` | Vulnerability alerts, destructive actions |

### 2.2 Dark Theme — "The Observability Monolith"

A deep-blue slate environment that feels carved from a single block of precision material. Built for high-density data: SHA hashes, image layers, version trees. The UI is a precision instrument, not a dashboard.

| Token | Value | Role |
|---|---|---|
| `surface` | `#0b1326` | The bedrock — deepest layer |
| `surface_container_low` | `#131b2e` | Main workspace background |
| `surface_container` | `#1a2236` | Secondary navigation, panel grouping |
| `surface_container_high` | `#222c42` | Interactive modules, hover states |
| `surface_container_highest` | `#2d3449` | Critical active data, selected states |
| `surface_container_lowest` | `#0d1520` | Recessed cards, terminal blocks |
| `surface_variant` | `#1e2a40` | Glassmorphism base (used at 60% opacity) |
| `surface_bright` | `#2a3552` | Tertiary hover lift |
| `primary` | `#7bd0ff` | Primary brand, CTAs, active accents |
| `primary_fixed` | `#4db8ff` | Row indicator lines on hover |
| `on_primary_container` | `#0086b5` | Gradient endpoint for CTA fills |
| `on_primary` | `#003751` | Text on primary backgrounds |
| `secondary_container` | `#1e3a50` | Chip backgrounds |
| `on_secondary_container` | `#7bd0ff` | Chip text |
| `tertiary` | `#3cddc7` | Healthy/Running status, terminal text, focus borders |
| `on_surface` | `#e2e8f0` | All body text |
| `on_surface_variant` | `#8899b0` | Metadata, secondary labels |
| `outline_variant` | `#2d3a50` | Ghost border base (used at 15% opacity only) |
| `error` | `#ffb4ab` | Vulnerability alerts, destructive actions |

---

## 3. The "No-Line" Rule (Universal)

**1px solid borders are prohibited for structural sectioning in both themes.**

Traditional divider lines create visual noise and break the "Monolith" feel. Boundaries must be defined exclusively through **background color shifts** between adjacent surface tiers. When navigating from a sidebar to a content panel, it is the tonal step between `surface_container_low` and `surface_container_highest` that communicates the transition — not a line.

**Ghost Border Fallback:** Where containment is strictly required for accessibility or high-density data clarity, use `outline_variant` at **20% opacity (light)** or **15% opacity (dark)**. It should be felt, not seen.

---

## 4. Typography

### 4.1 Typefaces

| Context | Font | Rationale |
|---|---|---|
| Display, Headlines, Repo names | **Space Grotesk** | Geometric, monospaced-adjacent — architectural and futuristic |
| Body copy, dense metadata, SHA hashes | **Inter** | Optimized for high-density legibility at small sizes |

### 4.2 Type Scale

| Scale | Size | Usage |
|---|---|---|
| `display-lg` | 3.5rem | High-impact editorial moments; `-0.04em` letter-spacing for a "machined" look |
| `title-lg` | 1.5rem | Section headers; paired with `primary` color accent |
| `body-md` | 0.875rem | Standard reading text; `line-height: 1.5–1.6` for technical legibility |
| `label-sm` | 0.6875rem | Metadata, micro-copy; all-caps, `+0.05em` letter-spacing |

### 4.3 Metadata & Terminal Text

For SHA hashes, image IDs, Docker pull commands, and terminal outputs: use **Inter** with `letter-spacing: -0.01em`. This tightens the character rhythm for high-density lists without sacrificing legibility.

---

## 5. Elevation & Depth

### 5.1 Tonal Layering (Primary Method)

Depth is achieved through surface tier stacking, not drop shadows.

- **Light theme:** A `surface_container_lowest` (`#ffffff`) card on `surface` (`#f7f9fb`) creates a soft, natural lift through a ~2% brightness delta.
- **Dark theme:** A `surface_container_lowest` card on `surface_container` creates a **recessed** look — the card sinks into the dark environment, producing an expensive, inlaid feel.

Always ask: *"Can I define this area with a background color shift instead of a line?"*

### 5.2 Ambient Shadows (Floating Elements Only)

Use only for modals, command palettes, and dropdowns. Shadows must be an ambient whisper — if you can clearly see it, it is too dark.

| Theme | Shadow recipe |
|---|---|
| **Light** | `0 8px 32px rgba(25, 28, 30, 0.05)` — tinted with `on_surface` |
| **Dark** | `0 20px 40px rgba(6, 14, 32, 0.4)` — tinted with `surface_container_lowest` |

Blur: 32px–64px. Opacity: 4–6% (light) / 30–45% (dark).

### 5.3 Glassmorphism (Overlays & Command Palettes)

- **Light:** `surface_container_lowest` at 80% opacity + `backdrop-blur: 12px–20px`. Primary accent colors bleed through from the background.
- **Dark:** `surface_variant` at 60% opacity + `backdrop-blur: 20px`. Allows the deep-blue atmosphere to remain present beneath floating UI.

---

## 6. Components

### 6.1 Buttons

| Variant | Light Theme | Dark Theme |
|---|---|---|
| **Primary** | `primary` (`#006591`) fill → `on_primary` text; `border-radius: 0.375rem` | Gradient fill: `primary` → `on_primary_container` at 135°; `on_primary` text; no border |
| **Secondary** | `secondary_container` fill; `on_secondary_container` text; no border | `surface_container_highest` fill; `primary` ghost border at 10% opacity |
| **Tertiary** | Transparent; `primary` text; no background | Transparent; `primary` text; `surface_bright` on hover |

### 6.2 Input Fields (Command Line Style)

Inputs mimic a terminal prompt — no boxing.

- **Fill:** `surface_container_low` (light) / `surface_container_lowest` (dark)
- **Border:** `none` by default. **Bottom-only** `outline_variant` at 20% opacity.
- **Focus state (light):** 2px bottom border in `primary` (`#006591`), mimicking a terminal cursor.
- **Focus state (dark):** Bottom border transitions to `tertiary` (`#3cddc7`).

### 6.3 Chips / Tags / Version Labels

| Property | Light | Dark |
|---|---|---|
| Background | `surface_container_high` | `secondary_container` |
| Text | `on_surface_variant` | `on_secondary_container` |
| Active background | `primary_container` (`#0ea5e9`) | `primary` (`#7bd0ff`) at 15% opacity |
| Active text | `on_primary_container` (`#003751`) | `primary` |
| Corner radius | `full` — approachable, badge-like | `sm` (0.125rem) — sharp, technical |

### 6.4 Cards & Technical Lists

**Divider lines are forbidden in both themes.**

- Separate list items with **8px (light)** or **12px (dark)** vertical gap.
- Alternate background between adjacent surface tiers where needed for density.
- **Hover state (dark):** Row shifts to `surface_container_high`; a 2px `primary_fixed` indicator line appears on the far left edge.
- **Hover state (light):** Row background shifts to `surface_container_high`; left edge accent in `primary`.

### 6.5 Code Blocks & Monospace Metadata

| Property | Light | Dark |
|---|---|---|
| Background | `surface_container_highest` (`#e0e3e5`) | `surface_container_lowest` (`#0d1520`) |
| Text color | `on_surface` | `tertiary` (`#3cddc7`) — terminal vibes |
| Border radius | `none` or `sm` — structured feel | `md` (4px) |
| Font | Inter, `letter-spacing: -0.01em` | Inter, `letter-spacing: -0.01em` |

### 6.6 Status Indicators

| Status | Light | Dark |
|---|---|---|
| Healthy / Running | `primary_container` accent | `tertiary` (`#3cddc7`) — "Crisp Teal" |
| Vulnerability / Error | `error` (`#ba1a1a`) | `error` (`#ffb4ab`) |
| Neutral / Unknown | `on_surface_variant` | `on_surface_variant` |

---

## 7. Layout Principles (Universal)

- **Intentional Asymmetry:** Give the left side of the layout more breathing room than the right to create an editorial, off-center flow. White space is structural.
- **Tonal Nesting:** Define hierarchy through nested surface layers, not through lines or heavy borders.
- **High-Density by Default:** Developers prefer seeing 20 images at once. In dark theme especially, lean into density. Light theme may use more generous spacing for reading-heavy views.
- **Asymmetric Padding:** Left padding should generally exceed right padding in list views to create a visual anchor axis.

---

## 8. Signature Textures & Gradients

### Light Theme — Gradient CTA
Hero actions and primary buttons may use a linear gradient from `primary` (`#006591`) to `primary_container` (`#0ea5e9`). This provides a "glow" that is professional and intentional.

### Dark Theme — CTA Soul
Main CTAs use a linear gradient from `primary` (`#7bd0ff`) to `on_primary_container` (`#0086b5`) at **135°**. Status indicators and active chips may echo this gradient at reduced opacity.

---

## 9. Do's and Don'ts

### Do
- **Layer surfaces.** Always ask whether a background color shift can replace a line.
- **Use asymmetric padding.** The left axis anchors the layout.
- **Embrace the blue.** `primary_container` (light) and `tertiary` (dark) are the "highlighters" for keywords, statuses, and live indicators.
- **Use `error` sparingly.** Reserve it for genuine urgency — vulnerability alerts, destructive confirmations.
- **Meet contrast minimums.** All `on_surface_variant` metadata must achieve 4.5:1 contrast against its `surface_container` tier in both themes.

### Don't
- **Don't use pure black (`#000000`) or pure white (`#ffffff`) for text.** Always use theme-scoped `on_surface` tokens.
- **Don't use heavy shadows.** If the shadow is clearly visible, it is too dark.
- **Don't use large border-radii in the dark theme.** `xl` and `full` are too consumer-soft for a technical registry. Keep to `sm`–`md`.
- **Don't crowd the type.** Space Grotesk needs room. Body `line-height` must be 1.5 minimum.
- **Don't use 100% opaque borders for structural separation.** This violates the No-Line rule in both themes.

---

## 10. Accessibility

- All text must meet WCAG AA contrast (4.5:1 for body, 3:1 for large text) against its container in both themes.
- Ghost borders (`outline_variant` at reduced opacity) are supplemental only — never the sole means of communicating state or structure.
- Focus states must be clearly visible: 2px bottom border in light, teal bottom border in dark.
- Do not rely on color alone to communicate status — pair `error` and `tertiary` indicators with iconography or text labels.

---

**Director's Final Note:** ContainerHive's design language is about the *invisible hand of the engineer-designer*. In light mode, the user should feel organization through hierarchy of light and space. In dark mode, they should feel the precision of a calibrated instrument. In both: keep it sharp, keep it intentional, keep it precise.
