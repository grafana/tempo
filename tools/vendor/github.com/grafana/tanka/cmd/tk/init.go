package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/go-clix/cli"

	"github.com/grafana/tanka/pkg/spec/v1alpha1"
)

const defaultK8sVersion = "1.29"

// initCmd creates a new application
func initCmd(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Use:   "init",
		Short: "Create the directory structure",
		Args:  cli.ArgsNone(),
	}

	force := cmd.Flags().BoolP("force", "f", false, "ignore the working directory not being empty")
	installK8s := cmd.Flags().String("k8s", defaultK8sVersion, "choose the version of k8s-libsonnet, set to false to skip")
	inline := cmd.Flags().BoolP("inline", "i", false, "create an inline environment")

	cmd.Run = func(_ *cli.Command, _ []string) error {
		_, span := tracer.Start(ctx, "initCmd")
		defer span.End()
		failed := false

		files, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("error listing files: %s", err)
		}
		if len(files) > 0 && !*force {
			return fmt.Errorf("error: directory not empty. Use `-f` to force")
		}

		if err := writeNewFile("jsonnetfile.json", "{}"); err != nil {
			return fmt.Errorf("error creating `jsonnetfile.json`: %s", err)
		}

		if err := os.Mkdir("vendor", os.ModePerm); err != nil {
			return fmt.Errorf("error creating `vendor/` folder: %s", err)
		}

		if err := os.Mkdir("lib", os.ModePerm); err != nil {
			return fmt.Errorf("error creating `lib/` folder: %s", err)
		}

		cfg := v1alpha1.New()
		if err := addEnv("environments/default", cfg, *inline); err != nil {
			return err
		}

		version := *installK8s
		doInstall, err := strconv.ParseBool(*installK8s)
		if err != nil {
			// --k8s=<non-boolean>
			doInstall = true
		} else {
			// --k8s=<boolean>, fallback to default version
			version = defaultK8sVersion
		}

		if doInstall {
			if err := installK8sLib(version); err != nil {
				// This is not fatal, as most of Tanka will work anyways
				fmt.Println("Installing k.libsonnet:", err)
				failed = true
			}
		}

		if *inline {
			fmt.Println("Directory structure set up! Remember to configure the API endpoint in environments/default/main.jsonnet")
		} else {
			fmt.Println("Directory structure set up! Remember to configure the API endpoint:\n`tk env set environments/default --server=https://127.0.0.1:6443`")
		}
		if failed {
			fmt.Println("Errors occurred while initializing the project. Check the above logs for details.")
		}

		return nil
	}
	return cmd
}

func installK8sLib(version string) error {
	jbBinary := "jb"
	if env := os.Getenv("TANKA_JB_PATH"); env != "" {
		jbBinary = env
	}

	if _, err := exec.LookPath(jbBinary); err != nil {
		return errors.New("jsonnet-bundler not found in $PATH. Follow https://tanka.dev/install#jsonnet-bundler for installation instructions")
	}

	var initialPackages = []string{
		"github.com/jsonnet-libs/k8s-libsonnet/" + version + "@main",
		"github.com/grafana/jsonnet-libs/ksonnet-util",
		"github.com/jsonnet-libs/docsonnet/doc-util", // install docsonnet to make `tk lint` work
	}

	if err := writeNewFile("lib/k.libsonnet", "import 'github.com/jsonnet-libs/k8s-libsonnet/"+version+"/main.libsonnet'\n"); err != nil {
		return err
	}

	cmd := exec.Command(jbBinary, append([]string{"install"}, initialPackages...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// writeNewFile writes the content to a file if it does not exist
func writeNewFile(name, content string) error {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return os.WriteFile(name, []byte(content), 0644)
	}
	return nil
}
