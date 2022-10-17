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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type IndexImage struct {
	Digest    string    `json:"digest"`
	CreatedAt time.Time `json:"createdAt"`
	Platform  struct {
		Os      string `json:"os"`
		Arch    string `json:"arch"`
		Variant string `json:"variant"`
	} `json:"platform"`
	Layers []struct {
		Digest       string    `json:"digest"`
		Size         int       `json:"size"`
		LastModified time.Time `json:"lastModified"`
	} `json:"layers"`
	DigestChainId string `json:"digestChainId"`
	DiffIdChainId string `json:"diffIdChainId"`
}

type IndexManifestList struct {
	Name   string       `json:"name"`
	Tags   []string     `json:"tags"`
	Digest string       `json:"digest"`
	Images []IndexImage `json:"images"`
}

func ForBaseImageInIndex(digest digest.Digest, workspace string, apiKey string) (*[]Image, error) {
	url := fmt.Sprintf("https://api.dso.docker.com/docker-images/chain-ids/%s.json", digest.String())

	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query index")
	}

	if resp.StatusCode == 200 {
		var manifestList []IndexManifestList
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read response body")
		}
		err = json.Unmarshal(body, &manifestList)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal response body")
		}
		var ii IndexImage
		for _, i := range manifestList[0].Images {
			if i.DigestChainId == digest.String() || i.DiffIdChainId == digest.String() {
				ii = i
				break
			}
		}
		repository, err := ForRepositoryInDb(manifestList[0].Name, workspace, apiKey)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to query for respository")
		}
		image := Image{
			Digest:     ii.Digest,
			CreatedAt:  ii.CreatedAt,
			Tags:       manifestList[0].Tags,
			Repository: *repository,
		}
		return &[]Image{image}, nil
	}

	return nil, nil
}
