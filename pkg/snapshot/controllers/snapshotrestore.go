package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	snap_v1 "github.com/kubernetes-incubator/external-storage/snapshot/pkg/apis/crd/v1"
	"github.com/libopenstorage/stork/drivers/volume"
	"github.com/libopenstorage/stork/pkg/apis/stork"
	stork_api "github.com/libopenstorage/stork/pkg/apis/stork/v1alpha1"
	"github.com/libopenstorage/stork/pkg/controller"
	"github.com/libopenstorage/stork/pkg/log"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/portworx/sched-ops/k8s/apiextensions"
	"github.com/portworx/sched-ops/k8s/core"
	k8sextops "github.com/portworx/sched-ops/k8s/externalstorage"
	storkops "github.com/portworx/sched-ops/k8s/stork"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
)

const (
	annotationPrefix   = "stork.libopenstorage.org/"
	storkSchedulerName = "stork"
	// RestoreAnnotation for pvc which has in-place resotre in progress
	RestoreAnnotation            = annotationPrefix + "restore-in-progress"
	validateSnapshotTimeout      = 1 * time.Minute
	validateSnapshotRetryTimeout = 5 * time.Second
)

// SnapshotRestoreController controller to watch over In-Place snap restore CRD's
type SnapshotRestoreController struct {
	Driver   volume.Driver
	Recorder record.EventRecorder
}

// Init initialize the cluster pair controller
func (c *SnapshotRestoreController) Init() error {
	err := c.createCRD()
	if err != nil {
		return err
	}

	return controller.Register(
		&schema.GroupVersionKind{
			Group:   stork.GroupName,
			Version: stork_api.SchemeGroupVersion.Version,
			Kind:    reflect.TypeOf(stork_api.VolumeSnapshotRestore{}).Name(),
		},
		"",
		1*time.Minute,
		c)
}

// Handle updates for SnapshotRestore objects
func (c *SnapshotRestoreController) Handle(ctx context.Context, event sdk.Event) error {
	var (
		snapRestore *stork_api.VolumeSnapshotRestore
		err         error
	)

	switch o := event.Object.(type) {
	case *stork_api.VolumeSnapshotRestore:
		snapRestore = o
		if snapRestore.Spec.SourceName == "" {
			c.Recorder.Event(snapRestore,
				v1.EventTypeWarning,
				string(snapRestore.Spec.SourceName),
				"Empty Snapshot Name")
			return fmt.Errorf("empty snapshot name")
		}

		if event.Deleted {
			return c.handleDelete(snapRestore)
		}

		switch snapRestore.Status.Status {
		case stork_api.VolumeSnapshotRestoreStatusInitial:
			err = c.handleInitial(snapRestore)
		case stork_api.VolumeSnapshotRestoreStatusPending,
			stork_api.VolumeSnapshotRestoreStatusInProgress:
			err = c.handleStartRestore(snapRestore)
		case stork_api.VolumeSnapshotRestoreStatusStaged:
			err = c.handleFinal(snapRestore)
			if err == nil {
				c.Recorder.Event(snapRestore,
					v1.EventTypeNormal,
					string(snapRestore.Status.Status),
					"Snapshot in-Place  Restore completed")
			}
		case stork_api.VolumeSnapshotRestoreStatusFailed:
			err = c.Driver.CleanupSnapshotRestoreObjects(snapRestore)
		case stork_api.VolumeSnapshotRestoreStatusSuccessful:
			return nil
		default:
			err = fmt.Errorf("invalid stage for volume snapshot restore: %v", snapRestore.Status.Status)
		}
	}

	if err != nil {
		log.VolumeSnapshotRestoreLog(snapRestore).Errorf("Error handling event: %v err: %v", event, err.Error())
		c.Recorder.Event(snapRestore,
			v1.EventTypeWarning,
			string(stork_api.VolumeSnapshotRestoreStatusFailed),
			err.Error())
	}

	err = sdk.Update(snapRestore)
	if err != nil {
		return err
	}

	return nil
}

func (c *SnapshotRestoreController) handleStartRestore(snapRestore *stork_api.VolumeSnapshotRestore) error {
	log.VolumeSnapshotRestoreLog(snapRestore).Infof("Preparing volumes for snapshot restore %v", snapRestore.Spec.SourceName)
	inProgress, err := c.waitForRestoreToReady(snapRestore)
	if err != nil {
		return err
	}
	if inProgress {
		snapRestore.Status.Status = stork_api.VolumeSnapshotRestoreStatusInProgress
		return nil
	}

	// start in-place restore
	snapRestore.Status.Status = stork_api.VolumeSnapshotRestoreStatusStaged
	return nil
}

func (c *SnapshotRestoreController) handleInitial(snapRestore *stork_api.VolumeSnapshotRestore) error {
	// snapshot is list of snapshots
	snapshotList := []*snap_v1.VolumeSnapshot{}
	var err error

	snapName := snapRestore.Spec.SourceName
	snapNamespace := snapRestore.Spec.SourceNamespace
	log.VolumeSnapshotRestoreLog(snapRestore).Infof("Starting in place restore for snapshot %v", snapName)
	if snapRestore.Spec.GroupSnapshot {
		log.VolumeSnapshotRestoreLog(snapRestore).Infof("GroupVolumeSnapshot In-place restore request for %v", snapName)
		snapshotList, err = storkops.Instance().GetSnapshotsForGroupSnapshot(snapName, snapNamespace)
		if err != nil {
			log.VolumeSnapshotRestoreLog(snapRestore).Errorf("unable to get group snapshot details %v", err)
			return err
		}
	} else {
		// GetSnapshot Details
		snapshot, err := k8sextops.Instance().GetSnapshot(snapName, snapNamespace)
		if err != nil {
			return fmt.Errorf("unable to get get snapshot  details %s: %v",
				snapName, err)
		}
		if err := k8sextops.Instance().ValidateSnapshot(snapName,
			snapNamespace, false,
			validateSnapshotRetryTimeout,
			validateSnapshotTimeout); err != nil {
			return fmt.Errorf("snapshot is not complete %v", err)
		}
		snapshotList = append(snapshotList, snapshot)
	}

	// get map of snapID and pvcs
	err = initRestoreVolumesInfo(snapshotList, snapRestore)
	if err != nil {
		return err
	}

	snapRestore.Status.Status = stork_api.VolumeSnapshotRestoreStatusPending
	return nil
}

func (c *SnapshotRestoreController) handleFinal(snapRestore *stork_api.VolumeSnapshotRestore) error {
	var err error

	// annotate and delete pods using pvcs
	err = markPVCForRestore(snapRestore.Status.Volumes)
	if err != nil {
		log.VolumeSnapshotRestoreLog(snapRestore).Errorf("unable to mark pvc for restore %v", err)
		return err
	}
	// Do driver volume snapshot restore here
	err = c.Driver.CompleteVolumeSnapshotRestore(snapRestore)
	if err != nil {
		if err := unmarkPVCForRestore(snapRestore.Status.Volumes); err != nil {
			log.VolumeSnapshotRestoreLog(snapRestore).Errorf("unable to umark pvc for restore %v", err)
			return err
		}
		snapRestore.Status.Status = stork_api.VolumeSnapshotRestoreStatusFailed
		return fmt.Errorf("failed to restore pvc %v", err)
	}
	err = unmarkPVCForRestore(snapRestore.Status.Volumes)
	if err != nil {
		log.VolumeSnapshotRestoreLog(snapRestore).Errorf("unable to unmark pvc for restore %v", err)
		return err
	}

	snapRestore.Status.Status = stork_api.VolumeSnapshotRestoreStatusSuccessful
	return nil
}

func markPVCForRestore(volumes []*stork_api.RestoreVolumeInfo) error {
	for _, vol := range volumes {
		pvc, err := core.Instance().GetPersistentVolumeClaim(vol.PVC, vol.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get pvc details %v", err)
		}
		if pvc.Annotations == nil {
			pvc.Annotations = make(map[string]string)
		}
		pvc.Annotations[RestoreAnnotation] = "true"
		newPvc, err := core.Instance().UpdatePersistentVolumeClaim(pvc)
		if err != nil {
			return err
		}
		pods, err := core.Instance().GetPodsUsingPVC(newPvc.Name, newPvc.Namespace)
		if err != nil {
			return err
		}
		for _, pod := range pods {
			if pod.Spec.SchedulerName != storkSchedulerName {
				return fmt.Errorf("application not scheduled by stork scheduler")
			}
			log.PodLog(&pod).Infof("Deleting pod %v", pod.Name)
			if err := core.Instance().DeletePod(pod.Name, pod.Namespace, true); err != nil {
				log.PodLog(&pod).Errorf("Error deleting pod %v: %v", pod.Name, err)
				return err
			}
			if err := core.Instance().WaitForPodDeletion(pod.UID, pod.Namespace, 120*time.Second); err != nil {
				log.PodLog(&pod).Errorf("Pod is not deleted %v:%v", pod.Name, err)
				return err
			}
			log.PodLog(&pod).Debugf("Deleted pod %v", pod.Name)
		}
	}
	return nil
}

func unmarkPVCForRestore(volumes []*stork_api.RestoreVolumeInfo) error {
	// remove annotation from pvc's
	for _, vol := range volumes {
		pvc, err := core.Instance().GetPersistentVolumeClaim(vol.PVC, vol.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get pvc details %v", err)
		}
		logrus.Infof("Removing annotation for %v", pvc.Name)
		if pvc.Annotations == nil {
			// somehow annotation got deleted but since restore is done,
			// we shouldn't care
			log.PVCLog(pvc).Warnf("No annotation found for %v", pvc.Name)
			continue
		}
		if _, ok := pvc.Annotations[RestoreAnnotation]; !ok {
			log.PVCLog(pvc).Warnf("Restore annotation not found for %v", pvc.Name)
			continue
		}
		delete(pvc.Annotations, RestoreAnnotation)
		_, err = core.Instance().UpdatePersistentVolumeClaim(pvc)
		if err != nil {
			log.PVCLog(pvc).Warnf("failed to update pvc %v", err)
			return err
		}
	}

	return nil
}

func initRestoreVolumesInfo(snapshotList []*snap_v1.VolumeSnapshot, snapRestore *stork_api.VolumeSnapshotRestore) error {
	for _, snap := range snapshotList {
		snapData := string(snap.Spec.SnapshotDataName)
		logrus.Debugf("Getting volume ID for pvc %v", snap.Spec.PersistentVolumeClaimName)
		pvc, err := core.Instance().GetPersistentVolumeClaim(snap.Spec.PersistentVolumeClaimName, snap.Metadata.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get pvc details for snapshot %v", err)
		}

		volInfo := &stork_api.RestoreVolumeInfo{}
		// check whether we have volInfo already processed for given
		// pvc. If so update existing vol info
		for _, vol := range snapRestore.Status.Volumes {
			if pvc.Spec.VolumeName == vol.PVC {
				volInfo = vol
				break
			}
		}
		volInfo.Volume = pvc.Spec.VolumeName
		volInfo.PVC = pvc.Name
		volInfo.Namespace = pvc.Namespace
		volInfo.Snapshot = snapData
		volInfo.RestoreStatus = stork_api.VolumeSnapshotRestoreStatusInitial
		snapRestore.Status.Volumes = append(snapRestore.Status.Volumes, volInfo)
	}

	return nil
}

func (c *SnapshotRestoreController) createCRD() error {
	resource := apiextensions.CustomResource{
		Name:    stork_api.SnapshotRestoreResourceName,
		Plural:  stork_api.SnapshotRestoreResourcePlural,
		Group:   stork.GroupName,
		Version: stork_api.SchemeGroupVersion.Version,
		Scope:   apiextensionsv1beta1.NamespaceScoped,
		Kind:    reflect.TypeOf(stork_api.VolumeSnapshotRestore{}).Name(),
	}
	err := apiextensions.Instance().CreateCRD(resource)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return apiextensions.Instance().ValidateCRD(resource, validateCRDTimeout, validateCRDInterval)
}

func (c *SnapshotRestoreController) handleDelete(snapRestore *stork_api.VolumeSnapshotRestore) error {
	return c.Driver.CleanupSnapshotRestoreObjects(snapRestore)
}

func (c *SnapshotRestoreController) waitForRestoreToReady(
	snapRestore *stork_api.VolumeSnapshotRestore,
) (bool, error) {
	if snapRestore.Status.Status == stork_api.VolumeSnapshotRestoreStatusPending {
		err := c.Driver.StartVolumeSnapshotRestore(snapRestore)
		if err != nil {
			message := fmt.Sprintf("Error starting snapshot restore for volumes: %v", err)
			log.VolumeSnapshotRestoreLog(snapRestore).Errorf(message)
			c.Recorder.Event(snapRestore,
				v1.EventTypeWarning,
				string(stork_api.VolumeSnapshotRestoreStatusFailed),
				message)
			return false, err
		}

		snapRestore.Status.Status = stork_api.VolumeSnapshotRestoreStatusInProgress
		err = sdk.Update(snapRestore)
		if err != nil {
			return false, err
		}
	}

	// Volume Snapshot restore is already initiated , check for status
	continueProcessing := false
	// Skip checking status if no volumes are being restored
	if len(snapRestore.Status.Volumes) != 0 {
		err := c.Driver.GetVolumeSnapshotRestoreStatus(snapRestore)
		if err != nil {
			return continueProcessing, err
		}

		// Now check if there is any failure or success
		for _, vInfo := range snapRestore.Status.Volumes {
			if vInfo.RestoreStatus == stork_api.VolumeSnapshotRestoreStatusInProgress {
				log.VolumeSnapshotRestoreLog(snapRestore).Infof("Volume restore for volume %v is in %v state", vInfo.PVC, vInfo.RestoreStatus)
				continueProcessing = true
			} else if vInfo.RestoreStatus == stork_api.VolumeSnapshotRestoreStatusFailed {
				c.Recorder.Event(snapRestore,
					v1.EventTypeWarning,
					string(vInfo.RestoreStatus),
					fmt.Sprintf("Error restoring volume %v: %v", vInfo.PVC, vInfo.Reason))
				return false, fmt.Errorf("restore failed for volume: %v", vInfo.PVC)
			} else if vInfo.RestoreStatus == stork_api.VolumeSnapshotRestoreStatusSuccessful {
				c.Recorder.Event(snapRestore,
					v1.EventTypeNormal,
					string(vInfo.RestoreStatus),
					fmt.Sprintf("Volume %v restored successfully", vInfo.PVC))
			}
		}
	}

	return continueProcessing, nil
}
