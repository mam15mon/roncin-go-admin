# CLAUDE.md

## Project

React admin SPA on Vite + React Router v8 + React Query v5, antd v6, ProComponents v3, Vitest.

## Commands

Run from `web/` (or via `pnpm --dir web <script>` from the repo root; this repo uses **pnpm only** for dependency management — no npm/Yarn installs):

`pnpm dev` (Vite dev server on :8001; `pnpm start` is an alias), `pnpm build`, `pnpm lint` (Biome+tsc), `pnpm test` (Vitest), `pnpm test:e2e` (Playwright), `npx antd lint ./src` (antd-specific checks).

Other: `pnpm openapi` (regenerate `src/services/` from the server OpenAPI document; root shortcut `pnpm run generate:web-client`), `pnpm biome` (auto-fix), `pnpm tsc` (type-check only).

## Critical Rules

- **Never edit `src/services/roncin/`** — auto-generated, regenerate with `pnpm openapi`
- **Biome only** — no ESLint, no Prettier. `pnpm lint` must pass before commit; run `npx antd lint ./src` for antd-specific checks
- **Always `npx antd info <Component>` before writing antd code** — don't guess APIs from memory
- **Conventional commits** required (commitlint enforced)
- **TypeScript strict** · **Node ≥ 24.14.1** · **pnpm** with the repo-root `pnpm-lock.yaml` (not npm/Yarn)

## Architecture Essentials

**Config**: `vite.config.ts` (dev server, proxy, build), `config/routes.ts` (route table). `src/router/adaptRoutes.tsx` maps it onto React Router (`import.meta.glob` lazy pages) and ProLayout menu data; the `access` field gates visibility via `AccessGuard`.

**Entry & key files** (`src/`): `main.tsx` (Provider chain + global styles), `app/AppProvider.tsx` (initial state), `access.ts` (permissions), `requestErrorConfig.ts` (shared request-error handling), `typings.d.ts`.

**Auth**: `app/AppProvider.tsx` calls `authServiceMe()` (`/api/v1/auth/me`, proxied to the Go server in dev); 401 → redirect login. `access.ts` derives every gate from `currentUser.permissions` + `roleScopes`; permission key names are generated into `src/permissions.generated.ts` from the backend manifest — never hardcode a second source of truth.

**State**: `useInitialState()` (from `src/app/AppProvider.tsx`) for currentUser/settings. ProTable `request` prop for most data loading. `@tanstack/react-query` for server state.

**Styling priority**: Tailwind CSS v4 (layout) → antd-style v4 / `createStyles` (theme tokens) → CSS Modules → Less (legacy only).

**Request**: OpenAPI-generated clients in `src/services/roncin/` (regenerate via `pnpm openapi`); shared error handling in `src/requestErrorConfig.ts`. Page-specific requests/types/styles live next to the page — no ad-hoc backend host strings.

**Dev mock**: there is no global `mock/` directory. For offline local development, `src/utils/devMockUser.ts` provides a full-permission placeholder user, enabled only in dev builds via the localStorage flag `roncin_dev_mock_mode = 'true'`.

## AI Skills

This project ships with two built-in Claude Code Skills (`.claude/skills/`). If you already have these skills in your project, no installation is needed — just run them directly. To update to the latest skill definitions, run `npx skills add ant-design/ant-design-pro`.

### `/pro-upgrade` — Project Upgrade

Run `/pro-upgrade` in Claude Code to auto-upgrade the project to the latest Ant Design Pro version. It diffs the latest template against this project and merges framework changes while preserving business code. Works for any version gap (v5→v6, v6.x→latest, etc.).

### `/antd` — Ant Design CLI

Run `/antd` in Claude Code for any antd-related work. It provides access to `@ant-design/cli` with offline metadata for antd v3/v4/v5/v6. Key commands:

- `npx antd info <Component>` — look up props/API before writing code (mandatory)
- `npx antd lint ./src` — check for deprecated or problematic usage (must pass before commit)
- `npx antd demo <Component> <demo>` — get working code examples
- `npx antd migrate <from> <to>` — migration checklist between major versions

## Page Co-location

Each page dir keeps page-specific code next to the page: `index.tsx` / `detail.tsx` entry points, a `components/` subfolder, co-located `*.test.ts` files, and plain TS helpers/constants (`common.ts`, `*-constants.ts`, `list-query.ts`, …). Generated API calls come from `src/services/roncin/`.

# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

---