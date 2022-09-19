/*
 * Copyright © 2022 Docker, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/base-cli-plugin/commands"
	"github.com/docker/base-cli-plugin/internal"
	"github.com/docker/base-cli-plugin/query"
	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/moby/term"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func main() {
	plugin.Run(func(dockerCli command.Cli) *cobra.Command {
		var (
			apiKeyStdin bool
		)

		logout := &cobra.Command{
			Use:   "logout",
			Short: "Remove Atomist workspace authentication",
			RunE: func(cmd *cobra.Command, _ []string) error {
				dockerCli.ConfigFile().SetPluginConfig("base", "workspace", "")
				dockerCli.ConfigFile().SetPluginConfig("base", "api-key", "")
				return dockerCli.ConfigFile().Save()
			},
		}

		login := &cobra.Command{
			Use:   "login WORKSPACE",
			Short: "Authenticate with Atomist workspace",
			RunE: func(cmd *cobra.Command, args []string) error {
				workspace, err := readWorkspace(args, dockerCli)
				if err != nil {
					return err
				}
				apiKey, err := readApiKey(apiKeyStdin, dockerCli)
				if err != nil {
					return err
				}
				if query.CheckAuth(workspace, apiKey) {
					fmt.Println("Login successful")
					dockerCli.ConfigFile().SetPluginConfig("base", "workspace", workspace)
					dockerCli.ConfigFile().SetPluginConfig("base", "api-key", apiKey)
					return dockerCli.ConfigFile().Save()
				} else {
					return errors.New("Login failed")
				}
			},
		}
		loginFlags := login.Flags()
		loginFlags.BoolVar(&apiKeyStdin, "api-key-stdin", false, "Atomist API key")

		base := &cobra.Command{
			Use:   "detect [OPTIONS] IMAGE",
			Short: "Detect base images for a given image",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) != 1 {
					if err := cmd.Usage(); err != nil {
						return err
					}
					return fmt.Errorf(`"docker base detect" requires exactly 1 argument`)
				}
				workspace, _ := dockerCli.ConfigFile().PluginConfig("base", "workspace")
				apiKey, _ := dockerCli.ConfigFile().PluginConfig("base", "api-key")

				return commands.Detect(dockerCli, args[0], workspace, apiKey)
			},
		}
		base.MarkFlagRequired("image")

		cmd := &cobra.Command{
			Use:   "base",
			Short: "Identify base image",
		}

		cmd.AddCommand(login, logout, base)
		return cmd
	},
		manager.Metadata{
			SchemaVersion: "0.1.0",
			Vendor:        "Docker Inc.",
			Version:       internal.FromBuild().Version,
		})
}

func readWorkspace(args []string, cli command.Cli) (string, error) {
	var workspace string
	if len(args) == 1 {
		workspace = args[0]
	} else {
		fmt.Fprintf(cli.Out(), "Workspace: ")

		workspace = readInput(cli.In(), cli.Out())
		if workspace == "" {
			return "", errors.Errorf("Error: Workspace required")
		}
	}
	return workspace, nil
}

func readApiKey(apiKeyStdin bool, cli command.Cli) (string, error) {
	var apiKey string

	if apiKeyStdin {
		contents, err := io.ReadAll(cli.In())
		if err != nil {
			return "", err
		}

		apiKey = strings.TrimSuffix(string(contents), "\n")
		apiKey = strings.TrimSuffix(apiKey, "\r")
	} else if v, ok := os.LookupEnv("ATOMIST_API_KEY"); v != "" && ok {
		apiKey = v
	} else {
		oldState, err := term.SaveState(cli.In().FD())
		if err != nil {
			return "", err
		}
		fmt.Fprintf(cli.Out(), "API key: ")
		term.DisableEcho(cli.In().FD(), oldState)

		apiKey = readInput(cli.In(), cli.Out())
		fmt.Fprint(cli.Out(), "\n")
		term.RestoreTerminal(cli.In().FD(), oldState)
		if apiKey == "" {
			return "", errors.Errorf("Error: API key required")
		}
	}
	return apiKey, nil
}

func readInput(in io.Reader, out io.Writer) string {
	reader := bufio.NewReader(in)
	line, _, err := reader.ReadLine()
	if err != nil {
		fmt.Fprintln(out, err.Error())
		os.Exit(1)
	}
	return string(line)
}
