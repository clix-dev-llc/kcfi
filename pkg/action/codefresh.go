/*
Copyright The Codefresh Authors.

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

package action

import (
    "fmt"
	"path"
	"path/filepath"
	"os"
	"io/ioutil"

	"github.com/pkg/errors"
    "github.com/stretchr/objx"
    
	helm "helm.sh/helm/v3/pkg/action"
//	"helm.sh/helm/v3/pkg/postrender"
	"helm.sh/helm/v3/pkg/storage/driver"

	"k8s.io/cli-runtime/pkg/resource"
)

// GetDockerRegistryVars - calculater docker registry vals
func (o *CfApply) GetDockerRegistryVars () (map[string]interface{}, error) {
	
	var registryAddress, registryUsername, registryPassword string
	var err error
	valsX := objx.New(o.vals)
	usePrivateRegistry := valsX.Get(keyDockerUsePrivateRegistry).Bool(false)
	if !usePrivateRegistry {
		// using Codefresh Enterprise registry
		registryAddress = "gcr.io"
		registryUsername = "_json_key"
		cfRegistrySaVal := valsX.Get(keyDockerCodefreshRegistrySa).Str("sa.json")
		cfRegistrySaPath := path.Join(filepath.Dir(o.ConfigFile), cfRegistrySaVal)
    registryPasswordB, err := ioutil.ReadFile(cfRegistrySaPath)
    if err != nil {
        return nil, errors.Wrap(err, fmt.Sprintf("cannot read %s", cfRegistrySaPath))
		}
		registryPassword = string(registryPasswordB)
	} else {
		registryAddress = valsX.Get(keyDockerprivateRegistryAddress).String()
		registryUsername = valsX.Get(keyDockerprivateRegistryUsername).String()
		registryPassword = valsX.Get(keyDockerprivateRegistryPassword).String()
		if len(registryAddress) == 0 || len(registryUsername) == 0 || len(registryPassword) == 0 {
			err = fmt.Errorf("missing private registry data: ")
			if len(registryAddress) == 0 {
				err = errors.Wrapf(err, "missing %s", keyDockerprivateRegistryAddress)
			}
			if len(registryUsername) == 0 {
				err = errors.Wrapf(err, "missing %s", keyDockerprivateRegistryUsername)
			}
			if len(registryPassword) == 0 {
				err = errors.Wrapf(err, "missing %s", keyDockerprivateRegistryPassword)
			}
			return nil, err
		}
	}
	// Creating 
	registryTplData := map[string]interface{}{
		"RegistryAddress": registryAddress,
		"RegistryUsername": registryUsername,
		"RegistryPassword": registryPassword,
	} 
	registryValues, err := ExecuteTemplateToValues(RegistryValuesTpl, registryTplData)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error to parse docker registry values"))
  }

	return registryValues, nil
}

func (o *CfApply) ApplyCodefresh() error {

	registryValues, err := o.GetDockerRegistryVars()
	//_, err := o.AddDockerRegistryVars(vals)
	if err != nil {
		return errors.Wrapf(err, "Failed to parse docker registry values")
	}
	base := map[string]interface{}{}
	base = MergeMaps(base, o.vals)
	base = MergeMaps(base, registryValues)

	// If a release does not exist add seeded jobs
	histClient := helm.NewHistory(o.cfg)
	histClient.Max = 1
	if _, err := histClient.Run(CodefreshReleaseName); err == driver.ErrReleaseNotFound {
		seedJobsValues := map[string]interface{}{
			"global": map[string]interface{}{
				"seedJobs": true,
				"certsJobs": true,
			},
		}
		base = MergeMaps(base, seedJobsValues)
	}

	valuesTplResult, err := ExecuteTemplate(ValuesTpl, base)
	if err != nil {
		return errors.Wrapf(err, "Failed to generate values.yaml")
	}

	valuesYamlPath := path.Join(GetAssetsDir(o.ConfigFile), "values.yaml")
	err = ioutil.WriteFile(valuesYamlPath, []byte(valuesTplResult), 0644)
	if err != nil {
		return errors.Wrapf(err, "Failed to write %s ", valuesYamlPath)
	}
	fmt.Printf("values.yaml has been generated in %s\n", valuesYamlPath)

	cfResourceTplResult, err := ExecuteTemplate(CfResourceTpl, base)
	if err != nil {
		return errors.Wrapf(err, "Failed to generate codefresh-resource.yaml")
	}
	cfResourceYamlPath := path.Join(GetAssetsDir(o.ConfigFile), "codefresh-resource.yaml")
	err = ioutil.WriteFile(cfResourceYamlPath, []byte(cfResourceTplResult), 0644)
	if err != nil {
		return errors.Wrapf(err, "Failed to write %s ", cfResourceYamlPath)
	}
	fmt.Printf("codefresh-resource.yaml is generated in %s\n", cfResourceYamlPath)

    valsX := objx.New(o.vals)
    installerType := valsX.Get(keyInstallerType).String()
    if installerType == installerTypeOperator {
        cfResourceYamlReader, err := os.Open(cfResourceYamlPath)
        if err != nil {
            return errors.Wrapf(err, "Failed to read %s ", cfResourceYamlPath)
        }
        cfResources, err := o.cfg.KubeClient.Build(cfResourceYamlReader, true)
        if err != nil {
            return errors.Wrapf(err, "Failed to write %s ", cfResourceYamlPath)
        }
        fmt.Printf("applying %s\n %v", cfResourceYamlPath, cfResources)
        err = cfResources.Visit(func(info *resource.Info, err error) error {
            if err != nil {
                return err
            }

            helper := resource.NewHelper(info.Client, info.Mapping)
            _, err = helper.Replace(info.Namespace, info.Name, true, info.Object)
            return err
        })
        return err     
    }

    return nil
}