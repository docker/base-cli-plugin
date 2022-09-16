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

package registry

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

// DigestForImage retrieves the layer digests from either a local or remote image
func DigestForImage(dockerCli command.Cli, image string) ([]digest.Digest, error) {
	digests := make([]digest.Digest, 0)

	limg, _, err := dockerCli.Client().ImageInspectWithRaw(context.Background(), image)
	if err == nil {
		for _, l := range limg.RootFS.Layers {
			parsed, _ := digest.Parse(l)
			digests = append(digests, parsed)
		}
		return digests, nil
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse reference: %s", image)
	}

	// check local daemon first
	img, err := daemon.Image(ref)
	if err != nil {
		// image doesn't exist in daemon; try remote
		index, _ := remote.Index(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if index != nil {
			m, _ := index.IndexManifest()
			for _, manifest := range m.Manifests {
				ref, _ = name.ParseReference(fmt.Sprintf("%s@%s", ref.Context(), manifest.Digest))
				if manifest.Platform.OS == "linux" && manifest.Platform.Architecture == "amd64" {
					break
				}
			}
		}

		desc, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to pull image: %s", image)
		} else if desc != nil {
			img, _ = desc.Image()
		}
	}

	if img != nil {
		layers, _ := img.Layers()
		for _, layer := range layers {
			d, _ := layer.DiffID()
			parsed, _ := digest.Parse(d.String())
			digests = append(digests, parsed)
		}
	}

	return digests, nil
}
