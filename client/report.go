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

package client

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

func (c *Client) SetTestWorkloadResult(ctx context.Context, text string) error {
	var testWorkload naglfarv1.TestWorkload
	testWorkload.Namespace, testWorkload.Name = namespace, testWorkloadName
	key, err := client.ObjectKeyFromObject(&testWorkload)
	if err != nil {
		return err
	}
	if err := c.Get(ctx, key, &testWorkload); err != nil {
		return err
	}
	return retry(3, 1*time.Second, func() error {
		testWorkload.Status.Results[workloadItemName] = naglfarv1.TestWorkloadResult{
			PlainText: text,
		}
		if err := c.Update(ctx, &testWorkload); err != nil {
			return err
		}
		return nil
	})
}
