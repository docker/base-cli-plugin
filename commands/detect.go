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

package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/docker/base-cli-plugin/query"
	"github.com/docker/base-cli-plugin/registry"
	"github.com/docker/cli/cli/command"
	"github.com/fatih/color"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/identity"
	"github.com/xeonx/timeago"
)

func Detect(dockerCli command.Cli, image string, workspace string, apiKey string) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Retrieving layer information for image %s", image)
	s.Color("blue")
	s.Start()
	defer s.Stop()

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	blue := color.New(color.BgBlue, color.FgHiWhite).SprintFunc()
	red := color.New(color.BgRed, color.FgHiWhite).SprintFunc()

	digests, err := registry.DigestForImage(dockerCli, image)
	if err != nil {
		return err
	}
	chainIds := make([]digest.Digest, 0)
	for i := range digests {
		label := ""
		if i == 0 {
			label = "layer 0"
		} else {
			label = fmt.Sprintf("layers 0-%d", i)
		}
		chainIds = append(chainIds, digests[i])
		chainId := identity.ChainID(chainIds)
		s.Suffix = fmt.Sprintf(" Finding matching base images for %s", label)
		s.Restart()
		images, err := query.ForBaseImage(chainId, workspace, apiKey)

		if err != nil {
			return err
		}
		if images != nil {
			bi := make([]string, len(*images))
			for ix := range *images {
				image := (*images)[ix]
				e := "  "
				if image.Repository.Host != "hub.docker.com" {
					e += image.Repository.Host + "/"
				}
				e += image.Repository.Name
				e = green(e)
				if len(image.Tags) > 0 {
					tags := make([]string, len(image.Tags))
					for i := range image.Tags {
						tags[i] = cyan(image.Tags[i])
					}
					e += ":" + strings.Join(tags, ", ")

				}
				if oc := officialContent(image); oc != "" {
					e += " " + blue(oc)
					if st := supportedTag(image); st != "" {
						e += " " + red(st)
					}
				}
				if ct := currentTag(image); ct != "" {
					e += " " + red(ct)
				}
				e += "\n  " + image.Digest
				if cve := renderVulnerabilities(image); cve != "" {
					e += " " + red(cve)
				}
				e += " " + timeago.NoMax(timeago.English).Format(image.CreatedAt)
				if url := renderCommit(image); url != "" {
					e += "\n  " + url
				}
				bi[ix] = e
			}
			s.Stop()
			fmt.Printf("Base image for %s\n%s\n\n", label, strings.Join(bi, "\n\n"))
		}
	}
	return nil
}

func officialContent(image query.Image) string {
	switch image.Repository.Badge {
	case "open_source":
		return " Sponsored OSS "
	case "verified_publisher":
		return " Verified Publisher "
	default:
		if image.Repository.Host == "hub.docker.com" && !strings.Contains(image.Repository.Name, "/") {
			return " Docker Official Image "
		}
	}
	return ""
}

func supportedTag(image query.Image) string {
	if tagCount := len(image.Repository.SupportedTags); tagCount > 0 {
		unsupportedTags := make([]string, 0)
		for _, tag := range image.Tags {
			if !contains(image.Repository.SupportedTags, tag) {
				unsupportedTags = append(unsupportedTags, tag)
			}
		}
		if len(unsupportedTags) == len(image.Tags) {
			return " unsupported tag "
		}
	}
	return ""
}

func currentTag(image query.Image) string {
	currentTags := make([]string, 0)
	for _, tag := range image.Tag {
		currentTags = append(currentTags, tag.Name)
	}
	for _, manifestList := range image.ManifestList {
		for _, tag := range manifestList.Tags {
			currentTags = append(currentTags, tag.Name)
		}
	}
	if len(currentTags) > 0 {
		for _, tag := range image.Tags {
			if contains(currentTags, tag) {
				return ""
			}
		}
	}
	return " tag moved "
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func renderCommit(image query.Image) string {
	if image.Commit.Sha != "" {
		url := fmt.Sprintf("https://github.com/%s/%s", image.Commit.Repo.Org.Name, image.Commit.Repo.Name)
		if image.File.Path != "" {
			url = fmt.Sprintf("%s/blob/%s/%s", url, image.Commit.Sha, image.File.Path)
		}
		return url
	}
	return ""
}

func renderVulnerabilities(image query.Image) string {
	if len(image.Report) > 0 {
		report := image.Report[0]
		parts := make([]string, 0)
		if report.Critical > 0 {
			parts = append(parts, " C"+strconv.FormatInt(report.Critical, 10))
		}
		if report.High > 0 {
			parts = append(parts, " H"+strconv.FormatInt(report.High, 10))
		}
		if report.Medium > 0 {
			parts = append(parts, " M"+strconv.FormatInt(report.Medium, 10))
		}
		if report.Low > 0 {
			parts = append(parts, " L"+strconv.FormatInt(report.Low, 10))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ") + " "
		}
	}
	return ""
}
