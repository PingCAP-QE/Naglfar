/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dockerutil

import (
	"bytes"
	"io"

	dockerTypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

func (client *Client) PullImageByPolicy(image string, policy naglfarv1.PullImagePolicy) error {
	_, _, err := client.ImageInspectWithRaw(client.Ctx, image)
	if err == nil && policy != naglfarv1.PullPolicyAlways {
		return nil
	}
	if err != nil && !docker.IsErrImageNotFound(err) {
		return err
	}
	reader, err := client.ImagePull(client.Ctx, image, dockerTypes.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	var b bytes.Buffer
	_, err = io.Copy(&b, reader)
	return err
}
