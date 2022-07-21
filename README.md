# Shipper <img align="right" width=300 src="logo.webp">

[![CI](https://github.com/Neosperience/shipper/actions/workflows/main.yaml/badge.svg)](https://github.com/Neosperience/shipper/actions/workflows/main.yaml)
[![codecov](https://codecov.io/gh/Neosperience/shipper/branch/main/graph/badge.svg?token=DZMN03DYDR)](https://codecov.io/gh/Neosperience/shipper)
[![GitHub release](https://img.shields.io/github/v/release/neosperience/shipper?include_prereleases)](https://github.com/Neosperience/shipper/releases)

**Ship your apps the GitOps way!**

Shipper is a CLI tool that automates deploying new versions of containerized applications to GitOps repository managed by tools such as [ArgoCD]. Deploying such applications usually means updating a configuration file in a Git repository and writing the newly built container image tag. This file is then read by the CD tool, processed using a templater and then syncronized with a remote infrastructure environment where the application is actually deployed.

A practical example in [Neosperience]: applications deployed on instances of the [Karavel Container Platform], which uses ArgoCD to monitor deployment configurations. To deploy a new image tag, a commit must be performed against either a [Helm] `values.yaml` file, or a [Kustomize] `kustomization.yaml` file setting a YAML key to the new value.

Shipper automates this by leveraging the native Git provider API (e.g. GitLab, GitHub, BitBucket, etc...) to atomically update the configuration file.

## Supported tools

### Git Providers

- [Azure DevOps]
- [BitBucket] (only BitBucket cloud ie. bitbucket.org)
- [Gitea]
- [GitHub] (both GitHub.com and GitHub Enteprise Server)
- [GitLab] (both self-managed and gitlab.com)

### Templaters

- Flat JSON (cdk.json)
- [Helm] (values.yaml)
- [Kustomize] (kustomization.yaml)

## Usage

```
NAME:
   shipper - A new cli application

USAGE:
   shipper [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --templater value, -p value                Template system (available: "helm", "kustomize") [$SHIPPER_PROVIDER]
   --repo-kind value, -t value                Repository type (available: "gitlab", "github", "bitbucket-cloud") (default: "gitlab") [$SHIPPER_REPO_KIND]
   --repo-branch value, -b value              Repository branch [$SHIPPER_REPO_BRANCH]
   --commit-author value, -a value            Commit author in "name <email>" format (default: "Shipper agent <shipper@example.com>") [$SHIPPER_COMMIT_AUTHOR]
   --commit-message value, -m value           Commit message (default: "Deploy") [$SHIPPER_COMMIT_MESSAGE]
   --container-image value, --ci value        Container image [$SHIPPER_CONTAINER_IMAGE]
   --container-tag value, --ct value          Container tag [$SHIPPER_CONTAINER_TAG]
   --helm-values-file value, --hpath value    [helm] Path to values.yaml file [$SHIPPER_HELM_VALUES_FILE]
   --helm-image-path value, --himg value      [helm] Container image path (default: "image.repository") [$SHIPPER_HELM_IMAGE_PATH]
   --helm-tag-path value, --htag value        [helm] Container tag path (default: "image.tag") [$SHIPPER_HELM_TAG_PATH]
   --kustomize-file value, --kfile value      [kustomize] Path to kustomization.yaml file [$SHIPPER_KUSTOMIZE_FILE]
   --gitlab-endpoint value, --gl-uri value    [gitlab] Gitlab API endpoint, including "/api/v4" (default: "https://gitlab.com/api/v4") [$SHIPPER_GITLAB_ENDPOINT]
   --gitlab-key value, --gl-key value         [gitlab] A valid API key with commit access [$SHIPPER_GITLAB_KEY]
   --gitlab-project value, --gl-pid value     [gitlab] Project ID in "org/project" format [$SHIPPER_GITLAB_PROJECT]
   --github-endpoint value, --gh-uri value    [github] GitHub API endpoint (include "/api/v3" if using Enterprise Server) (default: "https://api.github.com") [$SHIPPER_GITHUB_ENDPOINT]
   --github-key value, --gh-key value         [github] Username/password pair in "username:password" format (use a personal access token!) [$SHIPPER_GITHUB_KEY]
   --github-project value, --gh-pid value     [github] Project ID in "org/project" format [$SHIPPER_GITHUB_PROJECT]
   --bitbucket-key value, --bb-key value      [bitbucket-cloud] Username/password pair in "username:password" format (use app passwords!) [$SHIPPER_GITLAB_KEY]
   --bitbucket-project value, --bb-pid value  [bitbucket-cloud] Project path in "org/project" format [$SHIPPER_GITLAB_PROJECT]
   --help, -h                                 show help (default: false)
```

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

### Updating multiple images at once

Command line arguments `--container-image`, `--container-tag`, `--helm-values-file`, `--helm-image-path`, `--helm-tag-path` and `--kustomize-file` can be specified multiple times to update multiple images at once. If using the environment variables, these values are comma-separated (eg. `CONTAINER_IMAGES=image1,image2`).

This is an example of updating multiple images at once:

```bash
shipper -p helm --helm-values-file helm/values.yml \
  --helm-image-path image.repository --helm-tag-path image.tag --container-image img1 --container-tag tag1 \
  --helm-image-path image2.repository --helm-tag-path image2.tag --container-image img2 --container-tag tag2 \
  ... \
  --repo-branch master --commit-author "Test" ...
```

You can mix & match which to specify once and which to specify many times, but every tag *must* be specified either 1 or N times, with some exceptions where closely-related tags must always be both specified the same amount of times, for example:

- ✔️ one `--helm-values-file` but many `--helm-image-path` (all changes will be applied to the same YAML file)
- ✔️ many `--helm-values-file` and `--helm-image-path` (every change will be applied to a specific file, the same file can be specified multiple times)
- ❌ 3 instances of `--helm-image-path` but 2 instances of `--helm-values-file`
- ❌ non-equal amount of `--container-image`, `--container-tag`, `--helm-image-path`, `--helm-tag-path`

## Available templaters

### Helm

The Helm templater is meant to be used on Helm values.yaml files. For each image to update, you must specify a value for each of the following:

 - `--helm-values-file` / `SHIPPER_HELM_VALUES_FILES`: Path to the Helm values.yaml file to modify
 - `--helm-image-path` / `SHIPPER_HELM_IMAGE_PATHS`: YAML field in the Helm values.yaml file to modify with the specified image repository, using dot notation for nested fields (eg. `image.repository`)
 - `--helm-tag-path` / `SHIPPER_HELM_TAG_PATHS`: YAML field in the Helm values.yaml file to modify with the specified image tag, using dot notation for nested fields (eg. `image.tag`)

### Kustomize

The Kustomize templater is meant to be used on kustomization.yaml files using the `images` list format like in [this example](https://github.com/kubernetes-sigs/kustomize/blob/master/examples/image.md).

To use it, you must specify the following, either once or for each image/tag pair:

 - `--kustomize-file` / `SHIPPER_KUSTOMIZE_FILES`: Path to the kustomization.yaml file to modify

### JSON

The JSON templater is meant for flat JSON files such as CDK context files (`cdk.json`).

When using JSON, the `--container-image` argument will be used as the key to update in the JSON file, while the `--container-tag` argument will be used as the value. You will also need to specify the following value, either once or for each image/tag pair:

 - `--json-file` / `SHIPPER_JSON_FILES`: Path to the JSON file to modify

## Provider notes

### Azure DevOps

- Only Git repositories are supported, not Team Foundation (TFVC).
- When creating a [personal access token](https://docs.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate) for shipper, only the "Code (Read & write)" permission is needed.
- The author string MUST be in the `John Doe <john.doe@example.com>` format or the commit will ignore it and use the default (i.e. the credentials' owner).
- You will need both a Project ID (in `org/project` format) and a Repository ID, if you don't know what your Repository ID is, it's probably the Project ID (without the organization). E.g. If your Project ID is `my-org/my-project` and you have only one repository, your Repository ID is `my-project`.

### Bitbucket cloud

- When creating an [app password](https://support.atlassian.com/bitbucket-cloud/docs/app-passwords/) for shipper, only the permissions `repository:read` and `repository:write` are needed.
- The author string MUST be in the `John Doe <john.doe@example.com>` format or the commit will fail.
- Bitbucket cloud integration only works with Bitbucket cloud, Bitbucket server has completely different APIs.

### Gitea

- Due to how the Commit API is implemented, calling shipper with multiple files will result in a multiple commits, one per modified file.

### GitHub

- When creating a [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) for shipper, only the permissions `repo` is needed.
- The author string MUST be in the `John Doe <john.doe@example.com>` format or the commit will fail.
- The GitHub Cloud API endpoint is `https://api.github.com`, however GitHub Enterprise Server will have something more akin to `https://HOSTNAME/api/v3`
- Due to how the Commit API is implemented, calling shipper with multiple files will result in a multiple commits, one per modified file.

### GitLab

- When creating a [project access token](https://docs.gitlab.com/ee/user/project/settings/project_access_tokens.html) for shipper, only the permission `api` is needed. (Role depends on your branch permissions, eg. protected branches)

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

[neosperience]: https://neosperience.com
[argocd]: https://argoproj.github.io/cd
[karavel container platform]: https://platform.karavel.io
[helm]: https://helm.sh
[kustomize]: https://kustomize.io
[azure devops]: https://azure.microsoft.com/en-us/services/devops/repos/
[gitlab]: https://gitlab.com
[github]: https://github.com
[gitea]: https://gitea.com
[bitbucket]: https://bitbucket.com
[golang]: https://go.dev
[semver]: https://semver.org/
