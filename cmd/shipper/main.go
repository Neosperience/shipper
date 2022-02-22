package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"gitlab.neosperience.com/tools/shipper/targets"
	gitlab_target "gitlab.neosperience.com/tools/shipper/targets/gitlab"
	helm_templater "gitlab.neosperience.com/tools/shipper/templater/helm"
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
	}

	// Get provider to use
	templater := c.String("provider")
	switch templater {
	case "helm":
		valuesFile := c.String("helm-values-file")
		assert(valuesFile != "", "values.yaml path must be specified when using Helm")

		newfiles, err := helm_templater.UpdateHelmChart(repository, helm_templater.HelmProviderOptions{
			ValuesFile: valuesFile,
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
		return fmt.Errorf("not implemented")
	}

	return repository.Commit(payload)
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
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
				Usage:    "Repository type (available: \"gitlab\")",
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
			&cli.StringFlag{
				Name:    "gitlab-endpoint",
				Aliases: []string{"gl-uri"},
				Usage:   "[gitlab] Gitlab API endpoint, including \"/api/v4/\"",
				EnvVars: []string{"SHIPPER_GITLAB_ENDPOINT"},
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
		},
		Action: app,
	}

	check(app.Run(os.Args), "Fatal error")
}

func check(err error, format string, args ...interface{}) {
	if err != nil {
		log.Fatalf(format+": "+err.Error(), args...)
	}
}

func assert(cond bool, format string, args ...interface{}) {
	if !cond {
		log.Fatalf(format, args...)
	}
}
