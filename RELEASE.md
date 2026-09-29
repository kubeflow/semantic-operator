# Releasing Semantic Operator

Semantic Operator follows [Semantic Versioning](https://semver.org/). Release tags use
`vX.Y.Z`. Release candidates use `vX.Y.Z-rc.N`.

Only maintainers with repository and artifact-publishing access may perform a release. The
long-term container registry and publishing credentials must be confirmed by the maintainers
before publishing a release.

## Prepare the release

1. Start from an up-to-date `main` branch and create a release preparation branch.
2. Set `version` and `appVersion` in `charts/semantic-operator/Chart.yaml` to the release
   version without the leading `v`.
3. Confirm that the Helm chart's manager and server image repositories are the approved
   release locations.
4. Update release notes with user-visible changes, compatibility notes, and upgrade steps.
5. Run the complete validation suite:

   ```bash
   make lint
   make test
   make generate
   make helm-lint
   make helm-unittest
   make docs-check
   make docker-build TAG=vX.Y.Z
   ```

6. Confirm that `make generate` leaves no uncommitted generated changes.
7. Submit the release preparation pull request and obtain approval under `OWNERS`.

## Publish the release

After the preparation pull request is merged:

1. Create the `vX.Y.Z` tag from the approved commit on `main`.
2. Build and publish both Dockerfile targets, `manager` and `server`, with the same immutable
   version tag. Publish multi-architecture images when the release registry supports them.
3. Package the Helm chart and verify that its image defaults reference published images.
4. Create the GitHub release from the tag and attach the packaged chart and checksums when
   they are part of the approved distribution process.
5. Verify installation on a clean Kubernetes cluster. Reconcile a `SemanticModel`, query it
   through MCP and REST, and verify both StarRocks and Trino before announcing the release.

Never overwrite an existing release tag or published image tag. Publish a new patch release
for corrections.
