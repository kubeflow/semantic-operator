# Contributing to Semantic Operator

Welcome to Semantic Operator. We appreciate patches, documentation updates, bug reports,
and design discussions from the community.

Read the [Kubeflow contribution guide](https://www.kubeflow.org/docs/about/contributing/)
before contributing. All contributors must follow the Kubeflow community standards and
sign their commits under the Developer Certificate of Origin.

## Developer workflow

The [developer guide](https://kubeflow.github.io/semantic-operator/guides/developing/)
describes the repository layout, required tools, local build, test, and Kubernetes workflows.
Run the following checks before submitting a pull request:

```bash
make lint
make test
make generate
make helm-lint
make helm-unittest
make docs-check
```

If a change modifies anything under `api/`, commit the regenerated deepcopy code and CRD
manifests produced by `make generate`.

## Developer Certificate of Origin

Every commit must include a `Signed-off-by` line that certifies agreement with the
[Developer Certificate of Origin](https://developercertificate.org/). Create it with:

```bash
git commit -s
```

If a commit is missing the sign-off, amend it and update the branch:

```bash
git commit --amend --signoff --no-edit
git push --force-with-lease
```

## Issues and pull requests

Open an issue before starting a large feature or compatibility change. Small fixes can go
directly to a pull request. Keep each pull request focused and explain user-visible behavior,
tests, compatibility considerations, and documentation changes.

All submissions, including those from maintainers, require review. Approval and review
permissions are defined in `OWNERS` and `.github/CODEOWNERS`.

Do not include credentials, production data, or security vulnerability details in public
issues or pull requests. Follow `SECURITY.md` for private vulnerability reporting.
