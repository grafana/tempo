package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ory/viper"
	"github.com/pborman/uuid"
	"github.com/spf13/cobra"
)

func check(err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "go-acc <flags> <packages...>",
	Short: "Receive accurate code coverage reports for Golang (Go)",
	Args:  cobra.MinimumNArgs(1),
	Example: `$ go-acc github.com/some/package
$ go-acc -o my-coverfile.txt github.com/some/package
$ go-acc ./...
$ go-acc $(glide novendor)
$ go-acc  --ignore pkga,pkgb .

You can pass all flags defined by "go test" after "--":
$ go-acc . -- -short -v -failfast

You can pick an alternative go test binary using:

GO_TEST_BINARY="go test"
GO_TEST_BINARY="gotest"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, err := cmd.Flags().GetString("covermode")
		if err != nil {
			return err
		}

		if verbose, err := cmd.Flags().GetBool("verbose"); err != nil {
			return err
		} else if verbose {
			fmt.Println("Flag -v has been deprecated, use `go-acc -- -v` instead!")
		}

		ignores, err := cmd.Flags().GetStringSlice("ignore")
		if err != nil {
			return err
		}

		payload := "mode: " + mode + "\n"

		var packages []string
		var passthrough []string
		for _, a := range args {
			if len(a) == 0 {
				continue
			}

			// The first tag indicates that we're now passing through all tags
			if a[0] == '-' || len(passthrough) > 0 {
				passthrough = append(passthrough, a)
				continue
			}

			if len(a) > 4 && a[len(a)-4:] == "/..." {
				var buf bytes.Buffer
				c := exec.Command("go", "list", a)
				c.Stdout = &buf
				c.Stderr = &buf
				if err := c.Run(); err != nil {
					check(fmt.Errorf("unable to run go list: %w", err))
				}

				var add []string
				for _, s := range strings.Split(buf.String(), "\n") {
					// Remove go system messages, e.g. messages from go mod	like
					//   go: finding ...
					//   go: downloading ...
					//   go: extracting ...
					if len(s) > 0 && !strings.HasPrefix(s, "go: ") {
						// Test if package name contains ignore string
						ignore := false
						for _, ignoreStr := range ignores {
							if strings.Contains(s, ignoreStr) {
								ignore = true
								break
							}
						}

						if !ignore {
							add = append(add, s)
						}
					}
				}

				packages = append(packages, add...)
			} else {
				packages = append(packages, a)
			}
		}

		files := make([]string, len(packages))
		for k, pkg := range packages {
			files[k] = filepath.Join(os.TempDir(), uuid.New()) + ".cc.tmp"

			gotest := os.Getenv("GO_TEST_BINARY")
			if gotest == "" {
				gotest = "go test"
			}

			gt := strings.Split(gotest, " ")
			if len(gt) != 2 {
				gt = append(gt, "")
			}

			var c *exec.Cmd
			ca := append(append(
				[]string{
					gt[1],
					"-covermode=" + mode,
					"-coverprofile=" + files[k],
					"-coverpkg=" + strings.Join(packages, ","),
				},
				passthrough...),
				pkg)
			c = exec.Command(gt[0], ca...)

			stderr, err := c.StderrPipe()
			check(err)

			stdout, err := c.StdoutPipe()
			check(err)

			check(c.Start())

			var wg sync.WaitGroup
			wg.Add(2)
			go scan(&wg, stderr)
			go scan(&wg, stdout)

			check(c.Wait())

			wg.Wait()
		}

		for _, file := range files {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				continue
			}

			p, err := ioutil.ReadFile(file)
			check(err)

			ps := strings.Split(string(p), "\n")
			payload += strings.Join(ps[1:], "\n")
		}

		output, err := cmd.Flags().GetString("output")
		check(err)

		check(ioutil.WriteFile(output, []byte(payload), 0644))
		return nil
	},
}

func scan(wg *sync.WaitGroup, r io.ReadCloser) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "warning: no packages being tested depend on matches for pattern") {
			continue
		}
		fmt.Println(strings.Split(line, "% of statements in")[0])
	}
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	RootCmd.Flags().BoolP("verbose", "v", false, "Does nothing, there for compatibility")
	RootCmd.Flags().StringP("output", "o", "coverage.txt", "Location for the output file")
	RootCmd.Flags().String("covermode", "atomic", "Which code coverage mode to use")
	RootCmd.Flags().StringSlice("ignore", []string{}, "Will ignore packages that contains any of these strings")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

}

type filter struct {
	dtl io.Writer
}

func (f *filter) Write(p []byte) (n int, err error) {
	for _, ppp := range strings.Split(string(p), "\n") {
		if strings.Contains(ppp, "warning: no packages being tested depend on matches for pattern") {
			continue
		} else {
			nn, err := f.dtl.Write(p)
			n = n + nn
			if err != nil {
				return n, err
			}
		}
	}
	return len(p), nil
}
