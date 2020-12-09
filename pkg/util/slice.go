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

package util

func StringsContains(list []string, target string) bool {
	for _, elem := range list {
		if elem == target {
			return true
		}
	}
	return false
}

func StringsRemove(list []string, target string) []string {
	newList := make([]string, 0, len(list)-1)
	for _, elem := range list {
		if target != elem {
			newList = append(newList, elem)
		}
	}
	return newList
}
