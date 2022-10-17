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

package query

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"olympos.io/encoding/edn"
)

type ManifestList struct {
	Digest string `edn:"docker.manifest-list/digest"`
	Tags   []struct {
		Name string `edn:"docker.tag/name"`
	} `edn:"docker.manifest-list/tag"`
}

type Report struct {
	Critical    int64 `edn:"vulnerability.report/critical"`
	High        int64 `edn:"vulnerability.report/high"`
	Medium      int64 `edn:"vulnerability.report/medium"`
	Low         int64 `edn:"vulnerability.report/low"`
	Unspecified int64 `edn:"vulnerability.report/unspecified"`
}

type Repository struct {
	Badge         string   `edn:"docker.repository/badge"`
	Host          string   `edn:"docker.repository/host"`
	Name          string   `edn:"docker.repository/name"`
	SupportedTags []string `edn:"docker.repository/supported-tags"`
}

type Image struct {
	TeamId    string    `edn:"atomist/team-id"`
	Digest    string    `edn:"docker.image/digest"`
	CreatedAt time.Time `edn:"docker.image/created-at"`
	Tags      []string  `edn:"docker.image/tags"`
	Tag       []struct {
		Name string `edn:"docker.tag/name"`
	} `edn:"docker.image/tag"`
	ManifestList []ManifestList `edn:"docker.image/manifest-list"`
	Repository   Repository     `edn:"docker.image/repository"`
	File         struct {
		Path string `edn:"git.file/path"`
	} `edn:"docker.image/file"`
	Commit struct {
		Sha  string `edn:"git.commit/sha"`
		Repo struct {
			Name string `edn:"git.repo/name"`
			Org  struct {
				Name string `edn:"git.org/name"`
			} `edn:"git.repo/org"`
		} `edn:"git.commit/repo"`
	} `edn:"docker.image/commit"`
	Report []Report `edn:"vulnerability.report/report"`
}

type ImageQueryResult struct {
	Query struct {
		Data [][]Image `edn:"data"`
	} `edn:"query"`
}

type RepositoryQueryResult struct {
	Query struct {
		Data [][]Repository `edn:"data"`
	} `edn:"query"`
}

//go:embed base_image_query.edn
var baseImageQuery string

//go:embed repository_query.edn
var repositoryQuery string

//go:embed enabled_skills_query.edn
var enabledSkillsQuery string

func CheckAuth(workspace string, apiKey string) bool {
	resp, err := query(enabledSkillsQuery, workspace, apiKey)
	if resp.StatusCode != 200 || err != nil {
		return false
	}
	return true
}

// ForBaseImageInDb returns images with matching digest in :docker.image/blob-digest or :docker.image/diff-chain-id
func ForBaseImageInDb(digest digest.Digest, workspace string, apiKey string) (*[]Image, error) {
	resp, err := query(fmt.Sprintf(baseImageQuery, digest), workspace, apiKey)

	if workspace == "" || apiKey == "" {
		var result ImageQueryResult
		err = edn.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal response")
		}
		if len(result.Query.Data) > 0 {
			return &result.Query.Data[0], nil
		} else {
			return nil, nil
		}
	} else {
		var images [][]Image
		err = edn.NewDecoder(resp.Body).Decode(&images)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal response")
		}
		if len(images) > 0 {
			image := make([]Image, 0)

			for _, img := range images {
				tba := true
				for j := range image {
					if image[j].Digest == img[0].Digest && img[0].TeamId == "A11PU8L1C" {
						image[j] = img[0]
						tba = false
						break
					}
				}
				if tba {
					image = append(image, img[0])
				}
			}

			return &image, nil
		} else {
			return nil, nil
		}
	}
}

func ForRepositoryInDb(repo string, workspace string, apiKey string) (*Repository, error) {
	resp, err := query(fmt.Sprintf(repositoryQuery, repo), workspace, apiKey)

	if workspace == "" || apiKey == "" {
		var result RepositoryQueryResult
		err = edn.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal response")
		}
		if len(result.Query.Data) > 0 {
			return &result.Query.Data[0][0], nil
		} else {
			return nil, nil
		}
	} else {
		var repositories [][]Repository
		err = edn.NewDecoder(resp.Body).Decode(&repositories)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal response")
		}
		if len(repositories) > 0 {
			return &repositories[0][0], nil
		} else {
			return nil, nil
		}
	}
}

func query(query string, workspace string, apiKey string) (*http.Response, error) {
	url := "https://api.dso.docker.com/datalog/team/" + workspace
	if workspace == "" || apiKey == "" {
		url = "https://api.dso.docker.com/datalog/shared-vulnerability/queries"
		query = fmt.Sprintf(`{:queries [{:name "query" :query %s}]}`, query)
	} else {
		query = fmt.Sprintf(`{:query %s}`, query)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(query))
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Content-Type", "application/edn")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create http client")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run query")
	}
	return resp, nil
}
