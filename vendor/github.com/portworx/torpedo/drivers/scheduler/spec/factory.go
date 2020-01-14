package spec

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/portworx/torpedo/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Factory is an application spec factory
type Factory struct {
	specDir    string
	specParser Parser
}

// CloudVolDriver Name of the cloud provider's volume driver to be used
type CloudVolDriver string

const (
	gcepd CloudVolDriver = "gcepd"
)

var appSpecFactory = make(map[string]*AppSpec)
var supportedCloudProviders = map[CloudVolDriver]bool{
	gcepd: true,
}

// register registers a new spec with the factory
func (f *Factory) register(id string, app *AppSpec) {
	if _, ok := appSpecFactory[id]; !ok {
		logrus.Infof("Registering app: %v", id)
		appSpecFactory[id] = app
	}
}

// check if cloud provider drivers are being used for provisioning
func (f *Factory) getCloudProviderIfPresent(specDir, appDir, storageProvisioner string) string {
	cloudProviderDirList, err := ioutil.ReadDir(path.Join(specDir, appDir))
	if err != nil {
		return ""
	}

	for _, cloudFile := range cloudProviderDirList {
		logrus.Infof("Looking for cloud dir: %v for storage provisioner %s", cloudFile.Name(), storageProvisioner)
		// Check if cloud provider vol driver is being used and if specs are present for that vol driver
		if cloudFile.IsDir() && cloudFile.Name() == storageProvisioner {
			logrus.Infof("RK=> Found  Cloud dir: %v", cloudFile.Name())
			return cloudFile.Name()
		}
	}
	return ""
}

// Get returns a registered application
func (f *Factory) Get(id string) (*AppSpec, error) {
	if d, ok := appSpecFactory[id]; ok && d.Enabled {
		if copy := d.DeepCopy(); copy != nil {
			return d.DeepCopy(), nil
		}
		return nil, fmt.Errorf("error creating copy of app: %v", d)
	}

	return nil, &errors.ErrNotFound{
		ID:   id,
		Type: "AppSpec",
	}
}

// GetAll returns all registered enabled applications
func (f *Factory) GetAll() []*AppSpec {
	var specs []*AppSpec
	for _, val := range appSpecFactory {
		if val.Enabled {
			valCopy := val.DeepCopy()
			if valCopy != nil {
				specs = append(specs, valCopy)
			}
		}
	}

	return specs
}

// NewFactory creates a new spec factory
func NewFactory(specDir, volDriverName string, parser Parser) (*Factory, error) {
	f := &Factory{
		specDir:    specDir,
		specParser: parser,
	}

	appDirList, err := ioutil.ReadDir(f.specDir)
	if err != nil {
		return nil, err
	}

	for _, file := range appDirList {
		if file.IsDir() {
			specID := file.Name()

			var cloudProviderSpecID string
			cloudProviderSpecID = f.getCloudProviderIfPresent(f.specDir, specID, volDriverName)

			var specs []interface{}
			if cloudProviderSpecID != "" {
				logrus.Infof("Parsing: %v...", path.Join(f.specDir, specID+"/"+cloudProviderSpecID))
				specs, err = f.specParser.ParseSpecs(path.Join(f.specDir, specID+"/"+cloudProviderSpecID))
			} else {
				logrus.Infof("Parsing: %v...", path.Join(f.specDir, specID))
				specs, err = f.specParser.ParseSpecs(path.Join(f.specDir, specID))
			}
			if err != nil {
				return nil, err
			}

			if len(specs) == 0 {
				continue
			}

			// Register the spec
			f.register(specID, &AppSpec{
				Key:      specID,
				SpecList: specs,
				Enabled:  true,
			})
		}
	}

	if apps := f.GetAll(); len(apps) == 0 {
		return nil, fmt.Errorf("found 0 supported applications in given specDir: %v", specDir)
	}

	return f, nil
}
