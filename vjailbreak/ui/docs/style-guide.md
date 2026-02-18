# Platform9 UI Style Guide

This document captures the shared design language for the vJailbreak UI. Every new component should consume tokens and building blocks from the `src/design-system` package.

## Design Principles

1. **Consistency over customization** – prefer tokens + shared components before writing bespoke styles.
2. **Light/Dark Parity** – tokens must work in both palette modes surfaced by `ThemeContext`.
3. **Composable Primitives** – expose layout primitives (FormGrid, FieldLabel, ToggleField) for predictable form UX.

## Foundations

| Token Group | File                                          | Notes                                                              |
| ----------- | --------------------------------------------- | ------------------------------------------------------------------ |
| Color       | `src/design-system/foundations/palette.ts`    | Light/dark surfaces, secondary palette per mode, scrollbar colors. |
| Spacing     | `src/design-system/foundations/spacing.ts`    | Canonical spacing scale (`xxs`-`xxxl`) + helpers.                  |
| Radii       | `src/design-system/foundations/radii.ts`      | Tokenized shape radii for cards, pills, circles.                   |
| Shadows     | `src/design-system/foundations/shadows.ts`    | Layered elevation scale from hairline to xl.                       |
| Typography  | `src/design-system/foundations/typography.ts` | Re-exported `customTypography` with variant metadata.              |

### Usage Guidelines

```ts
import { spacingPx, gapStyle, radii, shadows } from 'src/design-system'

const Card = styled(Box)(({ theme }) => ({
  padding: spacingPx('lg'),
  borderRadius: radii.md,
  boxShadow: shadows.sm,
  ...gapStyle('md')
}))
```

## Components

| Component                                  | Purpose                                        | Key Props / Notes                                                                 |
| ------------------------------------------ | ---------------------------------------------- | --------------------------------------------------------------------------------- |
| `FieldLabel`                               | Standardized label + tooltip treatment.        | `label`, `tooltip`, `required`, `helperText`.                                     |
| `FormGrid`                                 | Responsive auto-fit grid for forms.            | `minWidth`, `gap`, accepts `BoxProps`.                                            |
| `ToggleField`                              | Outlined switch card with description.         | `label`, `tooltip`, `description`, passthrough `SwitchProps`.                     |
| `SurfaceCard`                              | Tokenized Paper variant for content groupings. | `title`, `subtitle`, `actions`, `footer`; exposes `data-testid="surface-card"`.   |
| `ActionButton`                             | CTA helper with tone + loading treatments.     | `tone` (`primary`/`secondary`/`danger`), `loading`, `data-testid` for automation. |
| `NavTabs` / `NavTab`                       | Dashboard/tab navigation with descriptions.    | `description`, `count`, `data-testid="dashboard-tabs"` for entire control.        |
| `DrawerShell` + `DrawerHeader/Body/Footer` | Shared drawer chrome + layout.                 | Always provide `onClose`; body/footer expose `data-testid`s for test targeting.   |

### Form Patterns

1. Wrap grouped inputs in `FormGrid` with `minWidth={320}` for dashboard parity.
2. Pair every input with `FieldLabel` (either standalone or via `ToggleField`).
3. Reserve inline helper copy for validation messaging; reuse `FieldLabel.helperText` for secondary hints.

## Migration Plan

1. Refactor dashboard forms (e.g., Rolling Migration drawer, Credentials forms) to consume `FormGrid` + `FieldLabel`.
2. Replace bespoke switch cards with `ToggleField` for automation/feature flags.
3. Centralize future primitives (Buttons, Drawers, Chips, Status indicators) under `src/design-system/components`.

## Storybook Usage

Use Storybook to review and document design-system primitives before integrating them into product flows.

```bash
yarn storybook          # Starts Storybook locally on http://localhost:6006
yarn storybook:build    # Generates the static Storybook bundle for CI/deploy
```

The Storybook setup automatically wraps stories with `ThemeProvider`, `CssBaseline`, and a centered container so light/dark mode parity is visible by default. Place new component stories in `src/design-system/components/*.stories.tsx` and include meaningful `data-testid`s in component code whenever they assist unit/automation tests (e.g., `drawer-shell`, `nav-tabs`, `*-submit`).

Adhering to these guidelines ensures the UI scales with minimal drift and accelerates future refinements.
