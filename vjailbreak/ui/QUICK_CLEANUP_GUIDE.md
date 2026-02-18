# Quick Cleanup Guide - Safe Deletions

## ✅ ZERO RISK - Delete These Now

### 1. Delete Duplicate API Directory

```bash
rm -rf src/shared/api/
```

**Why**: Complete duplicate of `src/api/`, not being used anywhere.

### 2. Delete Duplicate RouteCompatibility

```bash
rm src/components/providers/RouteCompatibility.tsx
```

**Why**: Duplicate of `src/app/layout/RouteCompatibility.tsx` which is actively used.

Then update the index file:

```bash
# Edit src/components/providers/index.ts
# Remove: export { default as RouteCompatibility } from './RouteCompatibility'
```

### 3. Remove .gitkeep Files (Optional)

```bash
rm src/app/.gitkeep
rm src/app/layout/.gitkeep
rm src/shared/utils/.gitkeep
rm src/shared/types/.gitkeep
rm src/shared/hooks/.gitkeep
rm src/shared/components/.gitkeep
```

**Why**: These directories either have content or aren't needed.

---

## ⚠️ LOW RISK - Consolidate Layout Components

### 4. Consolidate AppBar

The files `src/app/layout/AppBar.tsx` and `src/components/layout/AppBar.tsx` are **99% identical**.

**Only difference**: Import path for ThemeToggle

- `src/app/layout/AppBar.tsx` uses: `import { ThemeToggle } from 'src/components/layout'`
- `src/components/layout/AppBar.tsx` uses: `import ThemeToggle from './ThemeToggle'`

**Action**:

1. Keep `src/components/layout/AppBar.tsx` (better import)
2. Update `src/App.tsx` to import from components:
   ```typescript
   // Change from:
   import AppBar from './app/layout/AppBar'
   // To:
   import { AppBar } from './components/layout'
   ```
3. Delete `src/app/layout/AppBar.tsx`

### 5. Consolidate Sidenav

The files are **nearly identical** with only import path differences.

**Action**:

1. Keep `src/components/layout/Sidenav/Sidenav.tsx` (uses absolute imports - better)
2. Update `src/app/layout/DashboardLayout.tsx`:
   ```typescript
   // Change from:
   import Sidenav from './Sidenav'
   // To:
   import { Sidenav } from 'src/components/layout'
   ```
3. Delete `src/app/layout/Sidenav.tsx`

### 6. Move DashboardLayout

```bash
mv src/app/layout/DashboardLayout.tsx src/components/layout/
```

Then update `src/App.tsx`:

```typescript
// Change from:
import DashboardLayout from './app/layout/DashboardLayout'
// To:
import { DashboardLayout } from './components/layout'
```

### 7. Move RouteCompatibility

```bash
mv src/app/layout/RouteCompatibility.tsx src/components/providers/
```

Then update `src/App.tsx`:

```typescript
// Change from:
import RouteCompatibility from './app/layout/RouteCompatibility'
// To:
import { RouteCompatibility } from './components/providers'
```

### 8. Delete src/app Directory

After moving all files:

```bash
rm -rf src/app/
```

---

## 📋 Checklist

- [ ] Delete `src/shared/api/`
- [ ] Delete `src/components/providers/RouteCompatibility.tsx`
- [ ] Update `src/components/providers/index.ts`
- [ ] Remove `.gitkeep` files
- [ ] Update AppBar import in App.tsx
- [ ] Delete `src/app/layout/AppBar.tsx`
- [ ] Update Sidenav import in DashboardLayout.tsx
- [ ] Delete `src/app/layout/Sidenav.tsx`
- [ ] Move DashboardLayout to components/layout
- [ ] Move RouteCompatibility to components/providers
- [ ] Update App.tsx imports
- [ ] Delete `src/app/` directory
- [ ] Run `npm run build` to verify
- [ ] Run `yarn dev` to test

---

## 🎯 Expected Result

**Before:**

```
src/
├── app/
│   └── layout/
│       ├── AppBar.tsx (duplicate)
│       ├── Sidenav.tsx (duplicate)
│       ├── DashboardLayout.tsx
│       └── RouteCompatibility.tsx (duplicate)
├── components/
│   ├── layout/
│   │   ├── AppBar.tsx
│   │   └── Sidenav/
│   └── providers/
│       └── RouteCompatibility.tsx (duplicate)
└── shared/
    └── api/ (entire directory is duplicate)
```

**After:**

```
src/
├── components/
│   ├── layout/
│   │   ├── AppBar.tsx
│   │   ├── Sidenav/
│   │   ├── DashboardLayout.tsx (moved)
│   │   ├── Platform9Logo.tsx
│   │   └── ThemeToggle.tsx
│   └── providers/
│       ├── AnalyticsProvider.tsx
│       ├── ErrorBoundary.tsx
│       ├── RouteCompatibility.tsx (moved)
│       └── index.ts
└── shared/
    └── components/ (api/ deleted)
```

**Files Removed:** 8-10 files
**Directories Removed:** 2 (src/app/, src/shared/api/)
**Estimated Time:** 15-20 minutes
