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

package infoprovider

import (
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

/*
extraInfoProvider implements DeviceInfoProvider
*/
type extraInfoProvider struct {
	pciAddr   string
	extraInfo map[string]types.AdditionalInfo
}

// NewExtraInfoProvider create instance of Environment DeviceInfoProvider
func NewExtraInfoProvider(pciAddr string, extraInfo map[string]types.AdditionalInfo) types.DeviceInfoProvider {
	return &extraInfoProvider{
		pciAddr:   pciAddr,
		extraInfo: extraInfo,
	}
}

// *****************************************************************
/* DeviceInfoProvider Interface */

func (rp *extraInfoProvider) GetName() string {
	return "extra"
}

func (rp *extraInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	devSpecs := make([]*pluginapi.DeviceSpec, 0)
	return devSpecs
}

func (rp *extraInfoProvider) GetEnvVal() types.AdditionalInfo {
	extraInfos := make(map[string]string, 0)

	// first we search for global configuration with the * then we check for specific one to override
	for _, value := range []string{"*", rp.pciAddr} {
		extraInfoDict, ok := rp.extraInfo[value]
		if ok {
			for k, v := range extraInfoDict {
				extraInfos[k] = v
			}
		}
	}
	return extraInfos
}

func (rp *extraInfoProvider) GetMounts() []*pluginapi.Mount {
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

// *****************************************************************
