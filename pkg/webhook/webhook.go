// Copyright (c) 2018 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/intel/multus-cni/types"
	"github.com/pkg/errors"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type jsonPatchOperation struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value,omitempty"`
}

const (
	networksAnnotationKey  = "k8s.v1.cni.cncf.io/networks"
	networkResourceNameKey = "k8s.v1.cni.cncf.io/resourceName"
)

var (
	clientset kubernetes.Interface
)

func prepareAdmissionReviewResponse(allowed bool, message string, ar *v1beta1.AdmissionReview) error {
	if ar.Request != nil {
		ar.Response = &v1beta1.AdmissionResponse{
			UID:     ar.Request.UID,
			Allowed: allowed,
		}
		if message != "" {
			ar.Response.Result = &metav1.Status{
				Message: message,
			}
		}
		return nil
	}
	return errors.New("received empty AdmissionReview request")
}

func readAdmissionReview(req *http.Request) (*v1beta1.AdmissionReview, int, error) {
	var body []byte

	if req.Body != nil {
		if data, err := ioutil.ReadAll(req.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		err := errors.New("Error reading HTTP request: empty body")
		glog.Errorf("%s", err)
		return nil, http.StatusBadRequest, err
	}

	/* validate HTTP request headers */
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		err := errors.Errorf("Invalid Content-Type='%s', expected 'application/json'", contentType)
		glog.Errorf("%v", err)
		return nil, http.StatusUnsupportedMediaType, err
	}

	/* read AdmissionReview from the request body */
	ar, err := deserializeAdmissionReview(body)
	if err != nil {
		err := errors.Wrap(err, "error deserializing AdmissionReview")
		glog.Errorf("%v", err)
		return nil, http.StatusBadRequest, err
	}

	return ar, http.StatusOK, nil
}

func deserializeAdmissionReview(body []byte) (*v1beta1.AdmissionReview, error) {
	ar := &v1beta1.AdmissionReview{}
	runtimeScheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(runtimeScheme)
	deserializer := codecs.UniversalDeserializer()
	_, _, err := deserializer.Decode(body, nil, ar)

	/* Decode() won't return an error if the data wasn't actual AdmissionReview */
	if err == nil && ar.TypeMeta.Kind != "AdmissionReview" {
		err = errors.New("received object is not an AdmissionReview")
	}

	return ar, err
}

func deserializeNetworkAttachmentDefinition(ar *v1beta1.AdmissionReview) (types.NetworkAttachmentDefinition, error) {
	/* unmarshal NetworkAttachmentDefinition from AdmissionReview request */
	netAttachDef := types.NetworkAttachmentDefinition{}
	err := json.Unmarshal(ar.Request.Object.Raw, &netAttachDef)
	return netAttachDef, err
}

func deserializePod(ar *v1beta1.AdmissionReview) (corev1.Pod, error) {
	/* unmarshal Pod from AdmissionReview request */
	pod := corev1.Pod{}
	err := json.Unmarshal(ar.Request.Object.Raw, &pod)
	/* fix for missing "default" namespace */
	if pod.ObjectMeta.Namespace == "" {
		pod.ObjectMeta.Namespace = "default"
	}
	return pod, err
}
func parsePodNetworkSelections(podNetworks, defaultNamespace string) ([]*types.NetworkSelectionElement, error) {
	var networkSelections []*types.NetworkSelectionElement

	if len(podNetworks) == 0 {
		err := errors.New("empty string passed as network selection elements list")
		glog.Error(err)
		return nil, err
	}

	/* try to parse as JSON array */
	err := json.Unmarshal([]byte(podNetworks), &networkSelections)

	/* if failed, try to parse as comma separated */
	if err != nil {
		glog.Infof("'%s' is not in JSON format: %s... trying to parse as comma separated network selections list", podNetworks, err)
		for _, networkSelection := range strings.Split(podNetworks, ",") {
			networkSelection = strings.TrimSpace(networkSelection)
			networkSelectionElement, err := parsePodNetworkSelectionElement(networkSelection, defaultNamespace)
			if err != nil {
				err := errors.Wrap(err, "error parsing network selection element")
				glog.Error(err)
				return nil, err
			}
			networkSelections = append(networkSelections, networkSelectionElement)
		}
	}

	/* fill missing namespaces with default value */
	for _, networkSelection := range networkSelections {
		if networkSelection.Namespace == "" {
			networkSelection.Namespace = defaultNamespace
		}
	}

	return networkSelections, nil
}

func parsePodNetworkSelectionElement(selection, defaultNamespace string) (*types.NetworkSelectionElement, error) {
	var namespace, name, netInterface string
	var networkSelectionElement *types.NetworkSelectionElement

	units := strings.Split(selection, "/")
	switch len(units) {
	case 1:
		namespace = defaultNamespace
		name = units[0]
	case 2:
		namespace = units[0]
		name = units[1]
	default:
		err := errors.Errorf("invalid network selection element - more than one '/' rune in: '%s'", selection)
		glog.Info(err)
		return networkSelectionElement, err
	}

	units = strings.Split(name, "@")
	switch len(units) {
	case 1:
		name = units[0]
		netInterface = ""
	case 2:
		name = units[0]
		netInterface = units[1]
	default:
		err := errors.Errorf("invalid network selection element - more than one '@' rune in: '%s'", selection)
		glog.Info(err)
		return networkSelectionElement, err
	}

	validNameRegex, _ := regexp.Compile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	for _, unit := range []string{namespace, name, netInterface} {
		ok := validNameRegex.MatchString(unit)
		if !ok && len(unit) > 0 {
			err := errors.Errorf("at least one of the network selection units is invalid: error found at '%s'", unit)
			glog.Info(err)
			return networkSelectionElement, err
		}
	}

	networkSelectionElement = &types.NetworkSelectionElement{
		Namespace:        namespace,
		Name:             name,
		InterfaceRequest: netInterface,
	}

	return networkSelectionElement, nil
}

func getNetworkAttachmentDefinition(namespace, name string) (*types.NetworkAttachmentDefinition, error) {
	path := fmt.Sprintf("/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s", namespace, name)
	rawNetworkAttachmentDefinition, err := clientset.ExtensionsV1beta1().RESTClient().Get().AbsPath(path).DoRaw()
	if err != nil {
		err := errors.Wrapf(err, "could not get Network Attachment Definition %s/%s", namespace, name)
		glog.Error(err)
		return nil, err
	}

	networkAttachmentDefinition := types.NetworkAttachmentDefinition{}
	json.Unmarshal(rawNetworkAttachmentDefinition, &networkAttachmentDefinition)

	return &networkAttachmentDefinition, nil
}

func handleValidationError(w http.ResponseWriter, ar *v1beta1.AdmissionReview, orgErr error) {
	err := prepareAdmissionReviewResponse(false, orgErr.Error(), ar)
	if err != nil {
		err := errors.Wrap(err, "error preparing AdmissionResponse")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeResponse(w, ar)
}

func writeResponse(w http.ResponseWriter, ar *v1beta1.AdmissionReview) {
	glog.Infof("sending response to the Kubernetes API server")
	resp, _ := json.Marshal(ar)
	w.Write(resp)
}

func MutateHandler(w http.ResponseWriter, req *http.Request) {
	glog.Infof("Received mutation request")

	/* read AdmissionReview from the HTTP request */
	ar, httpStatus, err := readAdmissionReview(req)
	if err != nil {
		http.Error(w, err.Error(), httpStatus)
		return
	}

	/* read pod annotations */
	/* if networks missing skip everything */
	pod, err := deserializePod(ar)
	if err != nil {
		handleValidationError(w, ar, err)
		return
	}
	if netSelections, exists := pod.ObjectMeta.Annotations[networksAnnotationKey]; exists && netSelections != "" {
		/* map of resources request needed by a pod and a number of them */
		resourceRequests := make(map[string]int64)

		/* unmarshal list of network selection objects */
		networks, _ := parsePodNetworkSelections(netSelections, pod.ObjectMeta.Namespace)

		for _, n := range networks {
			/* for each network in annotation ask API server for network-attachment-definition */
			networkAttachmentDefinition, err := getNetworkAttachmentDefinition(n.Namespace, n.Name)
			if err != nil {
				/* if doesn't exist: deny pod */
				reason := errors.Wrapf(err, "could not find network attachment definition '%s/%s'", n.Namespace, n.Name)
				glog.Info(reason)
				err = prepareAdmissionReviewResponse(false, reason.Error(), ar)
				if err != nil {
					glog.Errorf("error preparing AdmissionReview response: %s", err)
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				writeResponse(w, ar)
				return
			}
			glog.Infof("network attachment definition '%s/%s' found", n.Namespace, n.Name)

			/* network object exists, so check if it contains resourceName annotation */
			if resourceName, exists := networkAttachmentDefinition.Metadata.Annotations[networkResourceNameKey]; exists {
				/* add resource to map/increment if it was already there */
				resourceRequests[resourceName]++
				glog.Infof("resource '%s' needs to be requested for network '%s/%s'", resourceName, n.Namespace, n.Name)
			} else {
				glog.Infof("network '%s/%s' doesn't use custom resources, skipping...", n.Namespace, n.Name)
			}
		}

		/* patch with custom resources requests and limits */
		err = prepareAdmissionReviewResponse(true, "allowed", ar)
		if err != nil {
			glog.Errorf("error preparing AdmissionReview response: %s", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(resourceRequests) == 0 {
			glog.Infof("pod doesn't need any custom network resources")
		} else {
			var patch []jsonPatchOperation

			resourceList := corev1.ResourceList{}
			for name, number := range resourceRequests {
				resourceList[corev1.ResourceName(name)] = *resource.NewQuantity(number, resource.DecimalSI)
			}

			patch = append(patch, jsonPatchOperation{
				Operation: "add",
				Path:      "/spec/containers/0/resources/requests", // NOTE: in future we may want to patch specific container (not always the first one)
				Value:     resourceList,
			})
			patch = append(patch, jsonPatchOperation{
				Operation: "add",
				Path:      "/spec/containers/0/resources/limits",
				Value:     resourceList,
			})
			patchBytes, _ := json.Marshal(patch)
			ar.Response.Patch = patchBytes
		}
	} else {
		/* network annoation not provided or empty */
		glog.Infof("pod spec doesn't have network annotations. Skipping...")
		err = prepareAdmissionReviewResponse(true, "Pod spec doesn't have network annotations. Skipping...", ar)
		if err != nil {
			glog.Infof("error preparing AdmissionReview response: %s", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	writeResponse(w, ar)
	return

}

func SetupInClusterClient() {
	/* setup Kubernetes API client */
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatal(err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}
}
