package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/neosperience/shipper/targets"
	bitbucket_target "github.com/neosperience/shipper/targets/bitbucket"
	gitea_target "github.com/neosperience/shipper/targets/gitea"
	github_target "github.com/neosperience/shipper/targets/github"
	gitlab_target "github.com/neosperience/shipper/targets/gitlab"
	helm_templater "github.com/neosperience/shipper/templater/helm"
	kustomize_templater "github.com/neosperience/shipper/templater/kustomize"
	"github.com/urfave/cli/v2"
)

func oneOrMany[T any](arr []T, index int) T {
	if len(arr) == 1 {
		return arr[0]
	}
	return arr[index]
}

func app(c *cli.Context) error {
	// Modify default HTTP client transport to not check for certificates if asked to do so
	insecureCert := c.Bool("no-verify-tls")
	if insecureCert {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Create payload
	payload := targets.NewPayload(c.String("repo-branch"), c.String("commit-author"), c.String("commit-message"))

	// Get target repository interface
	var repository targets.Repository
	target := c.String("repo-kind")
	switch target {
	case "gitlab":
		uri := c.String("gitlab-endpoint")
		assert(uri != "", "Gitlab endpoint must be specified when using Gitlab")

		project := c.String("gitlab-project")
		assert(project != "", "Gitlab project ID must be specified when using Gitlab")

		apikey := c.String("gitlab-key")
		assert(apikey != "", "Gitlab API key must be specified when using Gitlab")

		repository = gitlab_target.NewAPIClient(uri, project, apikey)
	case "github":
		uri := c.String("github-endpoint")
		assert(uri != "", "GitHub endpoint must be specified when using GitHub")

		project := c.String("github-project")
		assert(project != "", "GitHub project ID must be specified when using GitHub")

		apikey := c.String("github-key")
		assert(apikey != "", "GitHub credentials must be specified when using GitHub")

		repository = github_target.NewAPIClient(uri, project, apikey)
	case "gitea":
		uri := c.String("gitea-endpoint")
		assert(uri != "", "Gitea endpoint must be specified when using Gitea")

		project := c.String("gitea-project")
		assert(project != "", "Gitea project ID must be specified when using Gitea")

		apikey := c.String("gitea-key")
		assert(apikey != "", "Gitea credentials must be specified when using Gitea")

		repository = gitea_target.NewAPIClient(uri, project, apikey)
	case "bitbucket-cloud":
		creds := c.String("bitbucket-key")
		assert(creds != "", "Bitbucket cloud credentials must be specified when using Bitbucket cloud")

		project := c.String("bitbucket-project")
		assert(project != "", "Bitbucket project path must be specified when using Bitbucket cloud")

		repository = bitbucket_target.NewCloudAPIClient(project, creds)
	default:
		return fmt.Errorf("repository option not supported: %s", target)
	}

	// Get provider to use
	templater := c.String("templater")
	branch := c.String("repo-branch")

	images := c.StringSlice("container-image")
	tags := c.StringSlice("container-tag")
	assert(len(images) == len(tags), "An equal number of --container-image and --container-tag must be specified")

	switch templater {
	case "helm":
		valuesFile := c.StringSlice("helm-values-file")
		assert(valuesFile != nil && len(valuesFile) > 0, "values.yaml path must be specified when using Helm")
		imagePaths := c.StringSlice("helm-image-path")
		tagPaths := c.StringSlice("helm-tag-path")

		assert(len(tagPaths) == len(imagePaths), "An equal number of --helm-image-path and --helm-tag-path must be specified")
		assert(len(imagePaths) == 1 || len(imagePaths) == len(images), "There can on be either one global --helm-image-path or one per each --container-image")

		updates := make([]helm_templater.HelmUpdate, len(images))
		for index := 0; index < len(images); index += 1 {
			updates[index] = helm_templater.HelmUpdate{
				ValuesFile: oneOrMany(valuesFile, index),
				Image:      oneOrMany(images, index),
				Tag:        oneOrMany(tags, index),
				ImagePath:  oneOrMany(imagePaths, index),
				TagPath:    oneOrMany(tagPaths, index),
			}
		}

		newFiles, err := helm_templater.UpdateHelmChart(repository, helm_templater.HelmProviderOptions{
			Ref:     branch,
			Updates: updates,
		})
		if err != nil {
			return err
		}
		_ = payload.Files.Add(newFiles)
	case "kustomize":
		kustomizationFiles := c.StringSlice("kustomize-file")
		assert(kustomizationFiles != nil && len(kustomizationFiles) > 0, "kustomization.yaml path must be specified when using Kustomize")

		updates := make([]kustomize_templater.KustomizeUpdate, len(images))
		for index := 0; index < len(images); index += 1 {
			updates[index] = kustomize_templater.KustomizeUpdate{
				KustomizationFile: oneOrMany(kustomizationFiles, index),
				Image:             oneOrMany(images, index),
				NewTag:            oneOrMany(tags, index),
			}
		}

		newFiles, err := kustomize_templater.UpdateKustomization(repository, kustomize_templater.KustomizeProviderOptions{
			Ref:     branch,
			Updates: updates,
		})
		if err != nil {
			return err
		}
		_ = payload.Files.Add(newFiles)
	default:
		return fmt.Errorf("templater option not supported: %s", templater)
	}

	if len(payload.Files) < 1 {
		log.Println("no changes to commit, exiting")
		return nil
	}

	log.Printf("Pushing changes\n%s", payload)

	return repository.Commit(payload)
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			// Global options
			&cli.StringFlag{
				Name:     "templater",
				Aliases:  []string{"p"},
				Usage:    "Template system (available: \"helm\", \"kustomize\")",
				EnvVars:  []string{"SHIPPER_PROVIDER"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "repo-kind",
				Aliases:  []string{"t"},
				Value:    "gitlab",
				Usage:    "Repository type (available: \"gitlab\", \"github\", \"gitea\", \"bitbucket-cloud\")",
				EnvVars:  []string{"SHIPPER_REPO_KIND"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "repo-branch",
				Aliases:  []string{"b"},
				Usage:    "Repository branch",
				EnvVars:  []string{"SHIPPER_REPO_BRANCH"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "commit-author",
				Aliases: []string{"a"},
				Usage:   "Commit author in \"name <email>\" format",
				EnvVars: []string{"SHIPPER_COMMIT_AUTHOR"},
				Value:   "Shipper agent <shipper@example.com>",
			},
			&cli.StringFlag{
				Name:    "commit-message",
				Aliases: []string{"m"},
				Usage:   "Commit message",
				EnvVars: []string{"SHIPPER_COMMIT_MESSAGE"},
				Value:   "Deploy",
			},
			&cli.StringSliceFlag{
				Name:     "container-image",
				Aliases:  []string{"ci"},
				Usage:    "Container image",
				EnvVars:  []string{"SHIPPER_CONTAINER_IMAGE"},
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:     "container-tag",
				Aliases:  []string{"ct"},
				Usage:    "Container tag",
				EnvVars:  []string{"SHIPPER_CONTAINER_TAG"},
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "no-verify-tls",
				Usage:   "If provided, skip X.509 certificate validation on HTTPS requests",
				EnvVars: []string{"SHIPPER_NO_VERIFY_TLS"},
				Value:   false,
			},
			// Helm options
			&cli.StringSliceFlag{
				Name:    "helm-values-file",
				Aliases: []string{"hpath"},
				Usage:   "[helm] Path to values.yaml file",
				EnvVars: []string{"SHIPPER_HELM_VALUES_FILE"},
			},
			&cli.StringSliceFlag{
				Name:    "helm-image-path",
				Aliases: []string{"himg"},
				Usage:   "[helm] Container image path",
				EnvVars: []string{"SHIPPER_HELM_IMAGE_PATH"},
				Value:   cli.NewStringSlice("image.repository"),
			},
			&cli.StringSliceFlag{
				Name:    "helm-tag-path",
				Aliases: []string{"htag"},
				Usage:   "[helm] Container tag path",
				EnvVars: []string{"SHIPPER_HELM_TAG_PATH"},
				Value:   cli.NewStringSlice("image.tag"),
			},
			// Kustomize options
			&cli.StringFlag{
				Name:    "kustomize-file",
				Aliases: []string{"kfile"},
				Usage:   "[kustomize] Path to kustomization.yaml file",
				EnvVars: []string{"SHIPPER_KUSTOMIZE_FILE"},
			},
			// Gitlab options
			&cli.StringFlag{
				Name:    "gitlab-endpoint",
				Aliases: []string{"gl-uri"},
				Usage:   "[gitlab] Gitlab API endpoint, including \"/api/v4\"",
				EnvVars: []string{"SHIPPER_GITLAB_ENDPOINT"},
				Value:   "https://gitlab.com/api/v4",
			},
			&cli.StringFlag{
				Name:    "gitlab-key",
				Aliases: []string{"gl-key"},
				Usage:   "[gitlab] A valid API key with commit access",
				EnvVars: []string{"SHIPPER_GITLAB_KEY"},
			},
			&cli.StringFlag{
				Name:    "gitlab-project",
				Aliases: []string{"gl-pid"},
				Usage:   "[gitlab] Project ID in \"org/project\" format",
				EnvVars: []string{"SHIPPER_GITLAB_PROJECT"},
			},
			// GitHub options
			&cli.StringFlag{
				Name:    "github-endpoint",
				Aliases: []string{"gh-uri"},
				Usage:   "[github] GitHub API endpoint (include \"/api/v3\" if using Enterprise Server)",
				EnvVars: []string{"SHIPPER_GITHUB_ENDPOINT"},
				Value:   "https://api.github.com",
			},
			&cli.StringFlag{
				Name:    "github-key",
				Aliases: []string{"gh-key"},
				Usage:   "[github] Username/password pair in \"username:password\" format (use a personal access token!)",
				EnvVars: []string{"SHIPPER_GITHUB_KEY"},
			},
			&cli.StringFlag{
				Name:    "github-project",
				Aliases: []string{"gh-pid"},
				Usage:   "[github] Project ID in \"org/project\" format",
				EnvVars: []string{"SHIPPER_GITHUB_PROJECT"},
			},
			// Gitea options
			&cli.StringFlag{
				Name:    "gitea-endpoint",
				Aliases: []string{"ge-uri"},
				Usage:   "[gitea] GitHub API endpoint (include \"/api/v1\")",
				EnvVars: []string{"SHIPPER_GITEA_ENDPOINT"},
				Value:   "https://api.github.com",
			},
			&cli.StringFlag{
				Name:    "gitea-key",
				Aliases: []string{"ge-key"},
				Usage:   "[gitea] Username/application token pair in \"username:token\" format",
				EnvVars: []string{"SHIPPER_GITEA_KEY"},
			},
			&cli.StringFlag{
				Name:    "gitea-project",
				Aliases: []string{"ge-pid"},
				Usage:   "[gitea] Project ID in \"org/project\" format",
				EnvVars: []string{"SHIPPER_GITEA_PROJECT"},
			},
			// Bitbucket options
			&cli.StringFlag{
				Name:    "bitbucket-key",
				Aliases: []string{"bb-key"},
				Usage:   "[bitbucket-cloud] Username/password pair in \"username:password\" format (use app passwords!)",
				EnvVars: []string{"SHIPPER_GITLAB_KEY"},
			},
			&cli.StringFlag{
				Name:    "bitbucket-project",
				Aliases: []string{"bb-pid"},
				Usage:   "[bitbucket-cloud] Project path in \"org/project\" format",
				EnvVars: []string{"SHIPPER_GITLAB_PROJECT"},
			},
		},
		Action: app,
	}

	check(app.Run(os.Args), "Fatal error")
}

func check(err error, format string, args ...interface{}) {
	if err != nil {
		args = append(args, err.Error())
		log.Fatalf(format+": %s", args...)
	}
}

func assert(cond bool, format string, args ...interface{}) {
	if !cond {
		log.Fatalf(format, args...)
	}
}
