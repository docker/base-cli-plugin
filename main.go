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
	"fmt"

	"github.com/docker/base-cli-plugin/commands"
	"github.com/docker/base-cli-plugin/internal"
	"github.com/docker/base-cli-plugin/query"
	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func main() {
	plugin.Run(func(dockerCli command.Cli) *cobra.Command {
		var (
			workspace, apiKey string
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
			Use:   "login",
			Short: "Authenticate with an Atomist workspace",
			RunE: func(cmd *cobra.Command, _ []string) error {
				if !query.CheckAuth(workspace, apiKey) {
					return errors.New("login failed")
				} else {
					dockerCli.ConfigFile().SetPluginConfig("base", "workspace", workspace)
					dockerCli.ConfigFile().SetPluginConfig("base", "api-key", apiKey)
					return dockerCli.ConfigFile().Save()
				}
			},
		}
		loginFlags := login.Flags()
		loginFlags.StringVar(&workspace, "workspace", "", "Atomist workspace")
		loginFlags.StringVar(&apiKey, "api-key", "", "Atomist API key")
		login.MarkFlagRequired("workspace")
		login.MarkFlagRequired("api-key")

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
				if workspace == "" {
					workspace, _ = dockerCli.ConfigFile().PluginConfig("base", "workspace")
				}
				if apiKey == "" {
					apiKey, _ = dockerCli.ConfigFile().PluginConfig("base", "api-key")
				}

				return commands.Detect(dockerCli, args[0], workspace, apiKey)
			},
		}
		baseFlags := base.Flags()
		baseFlags.StringVar(&workspace, "workspace", "", "Atomist workspace")
		baseFlags.StringVar(&apiKey, "api-key", "", "Atomist API key")
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
