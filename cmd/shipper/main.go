package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/neosperience/shipper/targets"
	bitbucket_target "github.com/neosperience/shipper/targets/bitbucket"
	github_target "github.com/neosperience/shipper/targets/github"
	gitlab_target "github.com/neosperience/shipper/targets/gitlab"
	helm_templater "github.com/neosperience/shipper/templater/helm"
	kustomize_templater "github.com/neosperience/shipper/templater/kustomize"
)

func app(c *cli.Context) error {
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
		assert(apikey != "", "GitHub API key must be specified when using GitHub")

		repository = github_target.NewAPIClient(uri, project, apikey)
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
	switch templater {
	case "helm":
		valuesFile := c.String("helm-values-file")
		assert(valuesFile != "", "values.yaml path must be specified when using Helm")

		newfiles, err := helm_templater.UpdateHelmChart(repository, helm_templater.HelmProviderOptions{
			ValuesFile: valuesFile,
			Ref:        c.String("repo-branch"),
			Image:      c.String("container-image"),
			ImagePath:  c.String("helm-image-path"),
			Tag:        c.String("container-tag"),
			TagPath:    c.String("helm-tag-path"),
		})
		if err != nil {
			return err
		}
		payload.Files.Add(newfiles)
	case "kustomize":
		kustomizationFile := c.String("kustomize-file")
		assert(kustomizationFile != "", "kustomization.yaml path must be specified when using Kustomize")

		newfiles, err := kustomize_templater.UpdateKustomization(repository, kustomize_templater.KustomizeProviderOptions{
			KustomizationFile: kustomizationFile,
			Ref:               c.String("repo-branch"),
			Image:             c.String("container-image"),
			NewTag:            c.String("container-tag"),
		})
		if err != nil {
			return err
		}
		payload.Files.Add(newfiles)
	default:
		return fmt.Errorf("templater option not supported: %s", templater)
	}

	if len(payload.Files) < 1 {
		log.Println("no changes to commit, exiting")
		return nil
	}

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
				Usage:    "Repository type (available: \"gitlab\", \"github\", \"bitbucket-cloud\")",
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
			&cli.StringFlag{
				Name:     "container-image",
				Aliases:  []string{"ci"},
				Usage:    "Container image",
				EnvVars:  []string{"SHIPPER_CONTAINER_IMAGE"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "container-tag",
				Aliases:  []string{"ct"},
				Usage:    "Container tag",
				EnvVars:  []string{"SHIPPER_CONTAINER_TAG"},
				Required: true,
			},
			// Helm options
			&cli.StringFlag{
				Name:    "helm-values-file",
				Aliases: []string{"hpath"},
				Usage:   "[helm] Path to values.yaml file",
				EnvVars: []string{"SHIPPER_HELM_VALUES_FILE"},
			},
			&cli.StringFlag{
				Name:    "helm-image-path",
				Aliases: []string{"himg"},
				Usage:   "[helm] Container image path",
				EnvVars: []string{"SHIPPER_HELM_IMAGE_PATH"},
				Value:   "image.repository",
			},
			&cli.StringFlag{
				Name:    "helm-tag-path",
				Aliases: []string{"htag"},
				Usage:   "[helm] Container tag path",
				EnvVars: []string{"SHIPPER_HELM_TAG_PATH"},
				Value:   "image.tag",
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
