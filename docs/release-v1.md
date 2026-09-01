# Release protocol v1

After the pull request's conformance and semantic audit are closed, dispatch
the immutable release workflow on `main` with the exact merge SHA and a new
unused `0.x.y` version.

The workflow re-runs the full CI evidence, creates six assets from that exact
commit, creates one annotated tag, and verifies the tag object resolves to the
merge SHA. It refuses an existing tag or release and has no delete, overwrite,
merge, or promotion path. The final REST check requires a non-draft,
non-prerelease `immutable: true` release and exact asset name, size, and digest
tuples. If a release attempt leaves a tag behind, that version is permanently
burned and the next unused version must be used.
