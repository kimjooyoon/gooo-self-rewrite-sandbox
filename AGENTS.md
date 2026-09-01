# Development boundary

- Keep this repository independent from every sibling repository under
  `/Users/alice/meta-go`.
- Treat `meta/self-rewrite-boundary.gooo` as the semantic authority.
- Never write to the immutable input root; generated files belong only in the
  caller-owned temporary/output workspace.
- Never add automatic commit, push, merge, or promotion behavior.
- Validation runs in GitHub Actions. Do not run local test, build, vet, or
  conformance commands during development.
