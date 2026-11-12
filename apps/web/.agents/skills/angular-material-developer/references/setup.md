# Setup

## Install via schematics (recommended)

```
ng add @angular/material
```

The schematic asks these questions and wires everything for you:

1. **Custom theme or prebuilt?** Pick *Custom theme*, the schematic creates a `styles.scss` with `mat.theme()` and a default violet palette. The prebuilt themes use M2 internals and aren't recommended for new projects.
2. **Set up global typography?** Yes, this installs the Roboto font and wires `mat-typography` on `<body>`.

> **Animations (v21.2+ / v22):** Modern Material animates via native CSS and **dropped the `@angular/animations` peer dependency**. On 21.2+/22 the schematic no longer asks the "Include browser animations?" question, does not add `provideAnimationsAsync()`, and does not need `@angular/animations`, and a `--animations=enabled` flag is ignored. Only **v18-v21.1** projects need the provider (kept below for those). Verify by checking whether `@angular/material`'s `package.json` declares an `@angular/animations` peer, on 22.x it does not.

After it finishes (on 22.x):
- `@angular/material` and `@angular/cdk` are in `dependencies` (no `@angular/animations`).
- `src/styles.scss` contains a `mat.theme()` block with a default palette.
- `app.config.ts` has **no** `provideAnimationsAsync()` (native CSS animations).
- `angular.json` `styles` array points at `src/styles.scss`.
- `<body class="mat-typography">` is added to `index.html` (if you opted in to typography).

## Manual install

```
npm install @angular/material @angular/cdk
```

In `src/styles.scss`:

```scss
@use '@angular/material' as mat;

html {
  color-scheme: light dark;
  @include mat.theme((
    color: mat.$violet-palette,
    typography: Roboto,
    density: 0,
  ));
}
```

In `angular.json`, set `"styles": ["src/styles.scss"]`.

In `app.config.ts` (Material **22.x** / 21.2+, native CSS animations, no animations provider):

```ts
import { ApplicationConfig, provideZonelessChangeDetection } from '@angular/core';
import { provideRouter } from '@angular/router';

import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZonelessChangeDetection(),    // optional, Material is zoneless-compatible
    provideRouter(routes),
  ],
};
```

Only on **v18-v21.1** do you also need `provideAnimationsAsync()` from `@angular/platform-browser/animations/async` (and the `@angular/animations` package installed). On 22.x, adding it is unnecessary noise.

## Importing components

Material v15+ exports every component as a standalone class. Prefer the standalone import:

```ts
import { MatButton } from '@angular/material/button';

@Component({
  selector: 'app-save',
  imports: [MatButton],
  template: `<button mat-flat-button>Save</button>`,
})
export class SaveButton {}
```

The legacy `MatButtonModule` still re-exports the same classes, do not introduce new module imports in v15+ code.

## Common pitfalls

1. **No animations (v18-v21.1 only).** On those versions, symptom: ripples don't show, menu opens with no transition, dialog backdrop pops in. Fix: ensure `provideAnimationsAsync()` is in `app.config.ts`. **On 21.2+/22 this does not apply**, Material animates via native CSS and needs no provider; adding `provideAnimationsAsync()` there is harmless but pointless.
2. **Missing `@angular/animations` peer dep (v18-v21.1 only).** On those versions, `ng add @angular/material --animations=enabled` does **not** always install `@angular/animations` as a top-level dep, even though `provideAnimationsAsync()` dynamically imports `@angular/animations/browser` at runtime, and the build fails with `Could not resolve "@angular/animations/browser"`. Fix: `npm install @angular/animations` matching your Angular minor. On **22.x Material has no `@angular/animations` peer at all**, so this failure mode is gone; do not install the package chasing it.
3. **Missing `@angular/cdk`.** Material cascades into CDK for overlay, portal, a11y. If `@angular/cdk` isn't in dependencies, dialogs and dropdowns throw on construction.
4. **`styles.css` instead of `styles.scss`.** The theming engine requires Sass. Rename and update `angular.json`.
5. **Color-scheme not set.** Without `color-scheme: light dark`, native form controls render in light mode even when your app is dark. Add it on `html` in `styles.scss`.
6. **Using `mat.define-light-theme()` in v19+.** That API is M2-only. Switch to `mat.theme()`, see [theming.md](./theming.md).
7. **Browser-default `<button>` border bleeds through "borderless" designs.** If you build custom controls (calendar cells, kanban cards, nav rows) on `<button>` elements, the user-agent stylesheet draws a `buttonborder` line that's especially visible in light mode. Explicitly set `border: none` on the host. This isn't Material-specific, but Material's heavy use of `<button>` for icon and list-item triggers makes it show up frequently.

8. **Using `@HostListener` / `@HostBinding` in new code.** Symptom: works, but the host expressions get no compile-time type check, and the decorators don't compose cleanly with signal-based APIs. Fix: declare host bindings via the `host: {}` metadata in `@Component` / `@Directive`, then enable `typeCheckHostBindings: true` under `angularCompilerOptions` in `tsconfig.json`. Pattern:
   ```ts
   @Component({
     // ...
     host: {
       '(window:keydown)': 'onKey($event)',
       '[class.dark]': "mode() === 'dark'",
     },
   })
   ```
   Event targets (`window:`, `document:`, `body:`) live inside the parentheses of the key. This is an Angular fundamentals topic, not Material-specific, but it shows up in every component you'll write.

9. **`MatCalendar`'s `[startAt]` is init-only.** Binding `[startAt]="someSignal()"` sets the displayed month once, on first render, and does **not** follow later changes to the bound value. Symptom: a mini-calendar that ignores your toolbar's prev/next navigation. Fix: force a re-instantiation when the month changes, e.g. wrap it in `@for (key of [monthKey()]; track key) { <mat-calendar [startAt]="viewDate()" .../> }` so a changed `track` key rebuilds the component. `(selectedChange)` for the other direction works normally.

10. **Some M3 component-shape tokens aren't consumed.** In Material 21.2/22, `mat-nav-list` / `mat-list-item` ignore `--mat-list-list-item-container-shape` and `--mat-list-list-item-one-line-container-height` (the CSS vars exist but nothing reads them). Symptom: nav items keep their default 9999px pill radius and height no matter what you set. Fix: override `border-radius` and `height` **directly** on `a.mat-mdc-list-item` / `.mat-mdc-list-item`. Likewise the `mat-sidenav`'s own rounded corners need `border-radius: 0` on the `mat-sidenav` host, not a token. When a documented token has no visible effect, fall back to a plain CSS override on the rendered class rather than assuming the token is wrong.
