package applicationmanager

import (
	"os"
	"reflect"
	"time"

	"github.com/libopenstorage/stork/drivers/volume"
	"github.com/libopenstorage/stork/pkg/apis/stork"
	stork_api "github.com/libopenstorage/stork/pkg/apis/stork/v1alpha1"
	"github.com/libopenstorage/stork/pkg/applicationmanager/controllers"
	"github.com/libopenstorage/stork/pkg/resourcecollector"
	"github.com/portworx/sched-ops/k8s"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
)

const (
	validateCRDInterval time.Duration = 5 * time.Second
	validateCRDTimeout  time.Duration = 1 * time.Minute
)

// ApplicationManager maintains all controllers for application level operations
type ApplicationManager struct {
	Driver            volume.Driver
	Recorder          record.EventRecorder
	ResourceCollector resourcecollector.ResourceCollector
}

// Init Initializes the ApplicationManager and any children controller
func (a *ApplicationManager) Init(adminNamespace string, stopChannel chan os.Signal) error {
	if err := a.createCRD(); err != nil {
		return err
	}
	backupController := &controllers.ApplicationBackupController{
		Recorder:          a.Recorder,
		ResourceCollector: a.ResourceCollector,
	}
	if err := backupController.Init(adminNamespace); err != nil {
		return err
	}

	restoreController := &controllers.ApplicationRestoreController{
		Recorder:          a.Recorder,
		ResourceCollector: a.ResourceCollector,
	}
	if err := restoreController.Init(adminNamespace); err != nil {
		return err
	}

	cloneController := &controllers.ApplicationCloneController{
		Driver:            a.Driver,
		Recorder:          a.Recorder,
		ResourceCollector: a.ResourceCollector,
	}
	if err := cloneController.Init(adminNamespace); err != nil {
		return err
	}

	scheduleController := &controllers.ApplicationBackupScheduleController{
		Recorder: a.Recorder,
	}
	if err := scheduleController.Init(); err != nil {
		return err
	}

	syncController := &controllers.BackupSyncController{
		Recorder:     a.Recorder,
		SyncInterval: 1 * time.Minute,
	}
	if err := syncController.Init(stopChannel); err != nil {
		return err
	}
	return nil
}

func (a *ApplicationManager) createCRD() error {
	resource := k8s.CustomResource{
		Name:    stork_api.BackupLocationResourceName,
		Plural:  stork_api.BackupLocationResourcePlural,
		Group:   stork.GroupName,
		Version: stork_api.SchemeGroupVersion.Version,
		Scope:   apiextensionsv1beta1.NamespaceScoped,
		Kind:    reflect.TypeOf(stork_api.BackupLocation{}).Name(),
	}
	err := k8s.Instance().CreateCRD(resource)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return k8s.Instance().ValidateCRD(resource, validateCRDTimeout, validateCRDInterval)
}
