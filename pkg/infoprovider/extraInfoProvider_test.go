/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package infoprovider_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

var _ = Describe("ExtraInfoProvider", func() {
	Describe("creating new extraInfoProvider", func() {
		It("should return valid rdmaInfoProvider object", func() {
			dip := infoprovider.NewExtraInfoProvider("fake01", map[string]types.AdditionalInfo{})
			Expect(dip).NotTo(Equal(nil))
		})
	})
	Describe("GetEnvVal", func() {
		It("should return an empty list if there are no environment variables", func() {
			dip := infoprovider.NewExtraInfoProvider("fake", nil)
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(0))
		})
		It("should return an object with environment variables", func() {
			dip := infoprovider.NewExtraInfoProvider("fake", map[string]types.AdditionalInfo{"*": map[string]string{"test": "test"}})
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(1))
			value, exist := envs["test"]
			Expect(exist).To(BeTrue())
			Expect(value).To(Equal("test"))
		})
		It("should return an object with specific selector for environment variable", func() {
			dip := infoprovider.NewExtraInfoProvider("0000:00:00.1", map[string]types.AdditionalInfo{"*": map[string]string{"test": "test"}, "0000:00:00.1": map[string]string{"test": "test1"}})
			envs := dip.GetEnvVal()
			value, exist := envs["test"]
			Expect(exist).To(BeTrue())
			Expect(value).To(Equal("test1"))
		})
		It("should return an object with specific selector for multiple environment variable", func() {
			dip := infoprovider.NewExtraInfoProvider("0000:00:00.1", map[string]types.AdditionalInfo{"*": map[string]string{"test": "test", "bla": "bla"}, "0000:00:00.1": map[string]string{"test": "test1"}})
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(2))
			value, exist := envs["test"]
			Expect(exist).To(BeTrue())
			Expect(value).To(Equal("test1"))
			value, exist = envs["bla"]
			Expect(exist).To(BeTrue())
			Expect(value).To(Equal("bla"))
		})
		It("should return an object with multiple specific selector for environment variable", func() {
			dip := infoprovider.NewExtraInfoProvider("0000:00:00.1", map[string]types.AdditionalInfo{"*": map[string]string{"test": "test"}, "0000:00:00.1": map[string]string{"test": "test1", "bla": "bla"}})
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(2))
			value, exist := envs["test"]
			Expect(exist).To(BeTrue())
			Expect(value).To(Equal("test1"))
			value, exist = envs["bla"]
			Expect(exist).To(BeTrue())
			Expect(value).To(Equal("bla"))
		})
	})
})
