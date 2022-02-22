# Shipper <img align="right" width=300 src="logo.webp">

**Ship your apps the GitOps way!**

Shipper is a CLI tool that automates deploying new versions of containerized applications to GitOps repository managed by tools such as [ArgoCD]. Deploying such applications usually means updating a configuration file in a Git repository and writing the newly built container image tag. This file is then read by the CD tool, processed using a templater and then syncronized with a remote infrastructure environment where the application is actually deployed.

A practical example in Neosperience: applications deployed on instances of the [Karavel Container Platform], which uses ArgoCD to monitor deployment configurations. To deploy a new image tag, a commit must be performed against either a [Helm] `values.yaml` file, or a [Kustomize] `kustomization.yaml` file setting a YAML key to the new value.

Shipper automates this by leveraging the native Git provider API (e.g. GitLab, GitHub, BitBucket, etc...) to atomically update the configuration file.

## Usage

The main use-case for Shipper is to be used as a CI pipeline step. The Tools Team provides a preconfigured [GitLab CI template](https://gitlab.neosperience.com/tools/templates/) that you can just import in your `.gitlab-ci.yml`. The Templates repository README contains more information about its usage.

```yaml
include:
  - project: tools/templates
    file: /ci/build-buildctl.yml

  - project: tools/templates
    file: /ci/deploy-helm.yml
  # or
  - project: tools/templates
    file: /ci/deploy-kustomize.yml

build-and-push:
  extends: .build-with-buildctl
  stage: build

deploy-helm:
  extends: .deploy-with-helm-v2
  stage: deploy
  needs:
    - build-and-push
  environment: staging
  variables:
    DEPLOY_REPO: my-group/my-project-deployments
    DEPLOY_VALUES_FILE: staging/values.yaml

#or

deploy-kustomize:
  extends: .deploy-with-kustomize-v2
  stage: deploy
  needs:
    - build-and-push
  environment: staging
  variables:
    DEPLOY_REPO: my-group/my-project-deployments
    DEPLOY_KUSTOMIZATION_FILE: staging/kustomization.yaml
```

## Contributing

Found a bug or need a new feature? [Open an issue and discuss it with the team!](https://gitlab.neosperience.com/tools/shipper/-/issues/new).

Shipper is written in [Golang], so you'll need to install the SDK to work with the codebase.

## License

Copyright 2022 Neosperience S.P.A. All rights reserved

[ArgoCD]: https://argoproj.github.io/cd
[Karavel Container Platform]: https://platform.karavel.io
[Helm]: https://helm.sh
[Kustomize]: https://kustomize.io
[Golang]: https://go.dev
