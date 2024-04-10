/*
Copyright 2023.

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

package v1alpha1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Phase is a string representation of a HostConfig Phase.
type Phase string

// Phases for HostConfig.
const (
	PhaseInitializing = Phase("Initializing")
	PhaseProcessing   = Phase("Processing")
	PhaseSucceeded    = Phase("Succeeded")
	PhaseFailed       = Phase("Failed")
)

type Ansible struct {
	// RemotePlaybook 在远端的 playbook，单个 .tar.gz 压缩包，内容可以是单个 yaml 文件，也可以符合 ansible 要求的目录
	RemotePlaybook *RemotePlaybook `json:"remotePlaybook,omitempty"`
	// LocalPlaybookText 本地的 playbook，单个 yaml 文件， secret 引用或者 yaml 字符串
	LocalPlaybookText *YAMLText `json:"localPlaybookText,omitempty"`
	// Values 执行 playbook 的参数，yaml 格式，可以是 secret 引用或者 yaml 字符串
	Values *YAMLText `json:"values,omitempty"`
}

type RemotePlaybook struct {
	// URL playbook 在远端的地址，支持 https
	URL string `json:"url"`
	// Name 要执行的 playbook 文件名，相对于压缩包顶层的位置
	Name string `json:"name"`
	// MD5sum 压缩包的 MD5，填写了会进行校验，已经下载过的 playbook 校验通过后跳过重复下载
	MD5sum string `json:"md5sum,omitempty"`
}

type YAMLText struct {
	// SecretRef specifies the secret which stores yaml text.
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`
	// Inline is the inline yaml text.
	//+kubebuilder:validation:Format=yaml
	Inline string `json:"inline,omitempty"`
}

const (
	valuesYamlKey = "values.yaml"
)

func (in *YAMLText) IsEmpty() bool {
	if in == nil {
		return true
	}
	if in.SecretRef != nil && in.SecretRef.Name == "" && in.Inline == "" {
		return true
	}
	return in.SecretRef == nil && in.Inline == ""
}

func (in *YAMLText) getSecretNamespacedName(defaultNamespace string) (apitypes.NamespacedName, bool) {
	if in.SecretRef != nil && in.SecretRef.Name != "" {
		result := apitypes.NamespacedName{
			Namespace: in.SecretRef.Namespace,
			Name:      in.SecretRef.Name,
		}
		if result.Namespace == "" {
			result.Namespace = defaultNamespace
		}
		return result, true
	}
	return apitypes.NamespacedName{}, false
}

// GetContent 优先使用secret，如果不存在尝试使用inline yaml.
func (in *YAMLText) GetContent(ctx context.Context, c client.Client, defaultNamespace string) (string, error) {
	if in.IsEmpty() {
		return "", nil
	}
	secretKey, ok := in.getSecretNamespacedName(defaultNamespace)
	if ok {
		var err error
		secret := &corev1.Secret{}
		if err = c.Get(ctx, secretKey, secret); err != nil {
			return "", err
		}
		return string(secret.Data[valuesYamlKey]), nil
	} else {
		return in.Inline, nil
	}
}
