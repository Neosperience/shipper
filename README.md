# Shipper <img align="right" width=300 src="logo.webp">

[![codecov](https://codecov.io/gh/Neosperience/shipper/branch/main/graph/badge.svg?token=DZMN03DYDR)](https://codecov.io/gh/Neosperience/shipper)

**Ship your apps the GitOps way!**

Shipper is a CLI tool that automates deploying new versions of containerized applications to GitOps repository managed by tools such as [ArgoCD]. Deploying such applications usually means updating a configuration file in a Git repository and writing the newly built container image tag. This file is then read by the CD tool, processed using a templater and then syncronized with a remote infrastructure environment where the application is actually deployed.

A practical example in [Neosperience]: applications deployed on instances of the [Karavel Container Platform], which uses ArgoCD to monitor deployment configurations. To deploy a new image tag, a commit must be performed against either a [Helm] `values.yaml` file, or a [Kustomize] `kustomization.yaml` file setting a YAML key to the new value.

Shipper automates this by leveraging the native Git provider API (e.g. GitLab, GitHub, BitBucket, etc...) to atomically update the configuration file.

## Supported tools

### Git Providers

- [GitLab] (both self-managed and gitlab.com)
- [GitHub] (planned)
- [BitBucket] (planned)

### Templaters

- [Helm]
- [Kustomize]

## Usage

The main use-case for Shipper is to be used as a CI pipeline step. In container-based CI systems like GitLab CI, GitHub Actions and alike, you can run the [official container image](https://github.com/Neosperience/shipper/pkgs/container/shipper) in a step and invoke shipper with the appropriate flags.

### GitLab CI

```yaml
deploy-prod:
  image: ghcr.io/neosperience/shipper:main
  environment: prod
  script:
  - |
    shipper --templater helm \
      --repo-kind gitlab \
      --repo-branch main \
      --commit-author "$GITLAB_USER_NAME <$GITLAB_USER_EMAIL>" \
      --commit-message "Deploy new version" \
      --container-image $CI_REGISTRY_IMAGE \
      --container-tag  $CI_COMMIT_SHA \
      --helm-values-file prod/values.yaml \
      --helm-image-path image.repository \
      --helm-tag-path image.tag \
      --gitlab-endpoint $CI_API_V4_URL \
      --gitlab-key $CI_JOB_TOKEN" \
      --gitlab-project my-app/deployments
```

## Contributing

Found a bug or need a new feature? [Open an issue and discuss it with the team!](https://github.com/Neosperience/shipper/issues/new). Don't forget to check out our [contribution guide](CONTRIBUTING.md) for more information!

### Releasing

See our [release guide](RELEASE.md)

## License

Copyright 2022 Neosperience S.P.A.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

<a href="https://www.vecteezy.com/free-vector/ship">Logo by joezhuang on Vecteezy</a>

[Neosperience]: https://neosperience.com
[ArgoCD]: https://argoproj.github.io/cd
[Karavel Container Platform]: https://platform.karavel.io
[Helm]: https://helm.sh
[Kustomize]: https://kustomize.io
[GitLab]: https://gitlab.com
[GitHub]: https://github.com
[BitBucket]: https://bitbucket.com
[Golang]: https://go.dev
[SemVer]: https://semver.org/