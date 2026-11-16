# Dashboard Design Assets

- Current source of truth: `revision-v2/design-spec.md`
- Current confirmed prototype: `revision-v2/prototype.html`
- Historical baseline: `index.html` and `DESIGN.md` are marked `DEPRECATED`; retain them only for traceability.
- Production routing is strictly one model account to one upstream model. Observability is read-only and must not contain traffic switching, rollback, environment switching, or capacity writes.
- Visual evidence belongs in `revision-v2/screenshots/`; production release evidence belongs in `../mvp-list/evidence/screenshots/`.
