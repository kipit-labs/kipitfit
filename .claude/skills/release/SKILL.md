---
name: release
description: Cut a kipitfit release — bump `VERSION` in a `chore(release): X.Y.Z` PR and merge it (the release workflow tags and publishes the image). Deploying is not set up yet. Use when the user says "release", "cut a release", "deploy", or invokes /release [version|major|minor|patch].
argument-hint: "[patch|minor|major|X.Y.Z]"
disable-model-invocation: true
---

# Release

A release is a chore PR that bumps the root `VERSION` file. Merging it into `main` runs
`.github/workflows/release.yml`: it tags the merged commit `X.Y.Z`, publishes `ghcr.io/kipit-labs/kipitfit:X.Y.Z` +
`:latest` (amd64, arm64) and the GitHub release with generated notes. It refuses a `VERSION` that isn't above the last
tag. Tags are bare semver, no `v` prefix.

## Steps

1. **Preflight** — stop and report if any fails:
   - on `main`, working tree clean (`git status --porcelain` empty)
   - `git fetch --tags origin`, then local `main` == `origin/main`
   - there are commits since the last tag (`git describe --tags --abbrev=0`; the first release has no tag yet)
   - `make cover`, `make test`, `make e2e` and `make lint` green

2. **Pick the version.** `$ARGUMENTS` may be `X.Y.Z` or `major|minor|patch`. Without one, infer from the
   conventional commits since the last tag (`git log <last>..HEAD --format=%s`): any `!:` or `BREAKING CHANGE` →
   major, any `feat` → minor, otherwise patch. The first release is the `VERSION` already in the repo: tag it by
   merging any change to `VERSION`, or ask the user.

3. **Confirm with the user**: last tag → new version and the commit subjects included. Wait for a yes.

4. **Release PR, auto-merged:**
   ```sh
   git switch -c chore/release-X.Y.Z
   printf 'X.Y.Z\n' > VERSION
   git commit -am "chore(release): X.Y.Z"
   git push -u origin chore/release-X.Y.Z
   gh pr create --title "chore(release): X.Y.Z" --body "Release X.Y.Z."
   gh pr merge --rebase --auto
   ```
   No AI attribution on the commit or the PR. Never push a tag by hand: the workflow makes it.

5. **Wait for the image.** Once the PR is `MERGED`: `git switch main && git pull --ff-only && git branch -d
   chore/release-X.Y.Z`, then watch the run (`gh run list --workflow release.yml --limit 1`, `gh run watch <id>
   --exit-status`; on failure stop and show `gh run view <id> --log-failed`). Check the image is pullable:
   `docker manifest inspect ghcr.io/kipit-labs/kipitfit:X.Y.Z`.

6. **Deploy: not set up yet.** kipitfit will run on the homelab k3s cluster, deployed by ArgoCD from
   `~/work/homelab/ops` (origin `git.mathieuhoyer.fr/homelab/ops`), like kipitrek — but **the ops manifests for
   kipitfit don't exist yet** (namespace, Deployment, Service, Ingress, longhorn PVC, ProtonPassSecret, ArgoCD
   Application, Velero `daily-backup` entry; see docs/design.md "Build order"). Stop after step 5 and tell the user.
   Once they exist, the deploy pins `ghcr.io/kipit-labs/kipitfit:X.Y.Z@<digest>` (from `docker buildx imagetools
   inspect ... --format '{{.Manifest.Digest}}'`, never `latest`) in the kipitfit Deployment and pushes the ops repo;
   update this step then.

7. **Report**: version, GitHub release URL, image tag, and that deploying is still manual / not set up.
