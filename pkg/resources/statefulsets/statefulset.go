package statefulsets

import (
	pvc "github.com/rh-messaging/activemq-artemis-operator/pkg/resources/persistentvolumeclaims"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"context"
	brokerv1alpha1 "github.com/rh-messaging/activemq-artemis-operator/pkg/apis/broker/v1alpha1"
	"github.com/rh-messaging/activemq-artemis-operator/pkg/utils/selectors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("package statefulsets")

// newPodForCR returns an activemqartemis pod with the same name/namespace as the cr
//func newPodForCR(cr *brokerv1alpha1.ActiveMQArtemis) *corev1.Pod {
//
//	// Log where we are and what we're doing
//	reqLogger:= log.WithName(cr.Name)
//	reqLogger.Info("Creating new pod for custom resource")
//
//	dataPath := "/opt/" + cr.Name + "/data"
//	userEnvVar := corev1.EnvVar{"AMQ_USER", "admin", nil}
//	passwordEnvVar := corev1.EnvVar{"AMQ_PASSWORD", "admin", nil}
//	dataPathEnvVar := corev1.EnvVar{ "AMQ_DATA_DIR", dataPath, nil}
//
//	pod := &corev1.Pod{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      cr.Name,
//			Namespace: cr.Namespace,
//			Labels:    cr.Labels,
//		},
//		Spec: corev1.PodSpec{
//			Containers: []corev1.Container{
//				{
//					Name:    cr.Name + "-container",
//					Image:	 cr.Spec.Image,
//					Command: []string{"/opt/amq/bin/launch.sh", "start"},
//					VolumeMounts:	[]corev1.VolumeMount{
//						corev1.VolumeMount{
//							Name:		cr.Name + "-data-volume",
//							MountPath:	dataPath,
//							ReadOnly:	false,
//						},
//					},
//					Env:     []corev1.EnvVar{userEnvVar, passwordEnvVar, dataPathEnvVar},
//				},
//			},
//			Volumes: []corev1.Volume{
//				corev1.Volume{
//					Name: cr.Name + "-data-volume",
//					VolumeSource: corev1.VolumeSource{
//						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
//							ClaimName: cr.Name + "-pvc",
//							ReadOnly:  false,
//						},
//					},
//				},
//			},
//		},
//	}
//
//	return pod
//}

func makeDataPathForCR(cr *brokerv1alpha1.ActiveMQArtemis) string {
	return "/opt/" + cr.Name + "/data"
}

func makeVlMntCR(cr *brokerv1alpha1.ActiveMQArtemis) []corev1.VolumeMount {

	volumeMounts := []corev1.VolumeMount{}
	basicCRVlMnt := makeBasicCRVlMnt(cr)
	volumeMounts = append(volumeMounts, basicCRVlMnt...)

	if cr.Spec.SSLEnabled && len(cr.Spec.SSLConfig.SecretName) > 0 {
		sslCRVlMnt := makeSSLCRVlMnt(cr)
		volumeMounts = append(volumeMounts, sslCRVlMnt...)

	}
	return volumeMounts

}

func makeVolumesForCR(cr *brokerv1alpha1.ActiveMQArtemis) []corev1.Volume {

	volume := []corev1.Volume{}
	basicCRVolume := makeBasicCRVolumes(cr)
	volume = append(volume, basicCRVolume...)

	if cr.Spec.SSLEnabled && len(cr.Spec.SSLConfig.SecretName) > 0 {
		sslCRVolume := makeSSLCRVolumes(cr)
		volume = append(volume, sslCRVolume...)

	}
	return volume

}

func makeBasicCRVolumes(cr *brokerv1alpha1.ActiveMQArtemis) []corev1.Volume {
	volume := []corev1.Volume{
		{
			Name: cr.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: cr.Name,
					ReadOnly:  false,
				},
			},
		},
	}

	return volume
}

func makeSSLCRVolumes(cr *brokerv1alpha1.ActiveMQArtemis) []corev1.Volume {
	volume := []corev1.Volume{
		{
			Name: "broker-secret-volume",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cr.Spec.SSLConfig.SecretName,
				},
			},
		},
	}

	return volume
}

func makeBasicCRVlMnt(cr *brokerv1alpha1.ActiveMQArtemis) []corev1.VolumeMount {
	dataPath := makeDataPathForCR(cr)
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      cr.Name,
			MountPath: dataPath,
			ReadOnly:  false,
		},
	}
	return volumeMounts

}

func makeSSLCRVlMnt(cr *brokerv1alpha1.ActiveMQArtemis) []corev1.VolumeMount {

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "broker-secret-volume",
			MountPath: "/etc/amq-secret-volume",
			ReadOnly:  true,
		},
	}
	return volumeMounts

}

func makeEnvVarArrayForCR(cr *brokerv1alpha1.ActiveMQArtemis) []corev1.EnvVar {
	return makeEnvVarArrayForBasicCR(cr)
}

func getPropertyForCR (propName string, cr *brokerv1alpha1.ActiveMQArtemis, defaultValue string) string {
	result := defaultValue;
	switch propName {
		case "AMQ_USER":
			if len(cr.Spec.CredentialConfig.AMQUserName) > 0 {
				result = cr.Spec.CredentialConfig.AMQUserName
			}
		case "AMQ_PASSWORD":
			if len(cr.Spec.CredentialConfig.AMQPassword) > 0 {
				result = cr.Spec.CredentialConfig.AMQPassword
			}
		case "AMQ_CLUSTER_USER":
			if len(cr.Spec.CredentialConfig.AMQClusterUser) > 0 {
				result = cr.Spec.CredentialConfig.AMQClusterUser
			}
		case "AMQ_CLUSTER_PASSWORD":
			if len(cr.Spec.CredentialConfig.AMQClusterPassword) > 0 {
				result = cr.Spec.CredentialConfig.AMQClusterPassword
			}
		case "AMQ_KEYSTORE_TRUSTSTORE_DIR":
			if cr.Spec.SSLEnabled && len(cr.Spec.SSLConfig.SecretName) > 0 {
			result = "/etc/amq-secret-volume"
		    }
		case "AMQ_TRUSTSTORE":
			if cr.Spec.SSLEnabled && len(cr.Spec.SSLConfig.SecretName) > 0 {
				result = cr.Spec.SSLConfig.TrustStoreFilename
			}
		case "AMQ_TRUSTSTORE_PASSWORD":
			if cr.Spec.SSLEnabled && len(cr.Spec.SSLConfig.SecretName) > 0 {
				result = cr.Spec.SSLConfig.TrustStorePassword
			}
		case "AMQ_KEYSTORE":
			if cr.Spec.SSLEnabled && len(cr.Spec.SSLConfig.SecretName) > 0 {
				result = cr.Spec.SSLConfig.KeystoreFilename
			}
		case "AMQ_KEYSTORE_PASSWORD":
			if cr.Spec.SSLEnabled && len(cr.Spec.SSLConfig.SecretName) > 0 {
				result = cr.Spec.SSLConfig.KeyStorePassword
			}
	}
	return result;
}

func makeEnvVarArrayForBasicCR(cr *brokerv1alpha1.ActiveMQArtemis) []corev1.EnvVar {

	envVarArray := []corev1.EnvVar{
		{
			"AMQ_USER",
			getPropertyForCR("AMQ_USER", cr, "admin"),
			nil,
		},
		{
			"AMQ_PASSWORD",
			getPropertyForCR("AMQ_PASSWORD", cr, "admin"),
			nil,
		},
		{
			"AMQ_DATA_DIR",
			getPropertyForCR("AMQ_DATA_DIR", cr, "/opt/" + cr.Name + "/data"),
			nil,
		},
		{
			"AMQ_ROLE",
			getPropertyForCR("AMQ_ROLE", cr, "admin"),
			nil,
		},
		{
			"AMQ_NAME",
			getPropertyForCR("AMQ_NAME", cr, "amq-broker"),
			nil,
		},
		{
			"AMQ_TRANSPORTS",
			getPropertyForCR("AMQ_TRANSPORTS", cr, "openwire,amqp,stomp,mqtt,hornetq"),
			nil,
		},
		{
			"AMQ_QUEUES",
			//"q0,q1,q2,q3",
			getPropertyForCR("AMQ_QUEUES", cr, ""),
			nil,
		},
		{
			"AMQ_ADDRESSES",
			//"a0,a1,a2,a3",
			getPropertyForCR("AMQ_ADDRESSES", cr, ""),
			nil,
		},
		{
			"AMQ_KEYSTORE_TRUSTSTORE_DIR",
			getPropertyForCR("AMQ_KEYSTORE_TRUSTSTORE_DIR", cr, ""),
			nil,
		},
		{
			"AMQ_TRUSTSTORE",
			getPropertyForCR("AMQ_TRUSTSTORE", cr, ""),
			nil,
		},
		{
			"AMQ_TRUSTSTORE_PASSWORD",
			getPropertyForCR("AMQ_TRUSTSTORE_PASSWORD", cr, ""),
			nil,
		},
		{
			"AMQ_KEYSTORE",
			getPropertyForCR("AMQ_KEYSTORE", cr, ""),
			nil,
		},
		{
			"AMQ_KEYSTORE_PASSWORD",
			getPropertyForCR("AMQ_KEYSTORE_PASSWORD", cr, ""),
			nil,
		},
		{
			"AMQ_GLOBAL_MAX_SIZE",
			getPropertyForCR("AMQ_GLOBAL_MAX_SIZE", cr, "100 mb"),
			nil,
		},
		{
			"AMQ_REQUIRE_LOGIN",
			getPropertyForCR("AMQ_REQUIRE_LOGIN", cr, ""),
			nil,
		},
		{
			"AMQ_EXTRA_ARGS",
			getPropertyForCR("AMQ_EXTRA_ARGS", cr, "--no-autotune"),
			nil,
		},
		{
			"AMQ_ANYCAST_PREFIX",
			getPropertyForCR("AMQ_ANYCAST_PREFIX", cr, ""),
			nil,
		},
		{
			"AMQ_MULTICAST_PREFIX",
			getPropertyForCR("AMQ_MULTICAST_PREFIX", cr, ""),
			nil,
		},
		// included from p-c-ssl template
		{
			"AMQ_DATA_DIR_LOGGING",
			getPropertyForCR("AMQ_DATA_DIR_LOGGING", cr, "true"),
			nil,
		},
		{
			"AMQ_CLUSTERED",
			getPropertyForCR("AMQ_CLUSTERED", cr, "true"),
			nil,
		},
		{
			"AMQ_REPLICAS",
			getPropertyForCR("AMQ_REPLICAS", cr, "0"),
			nil,
		},
		{
			"AMQ_CLUSTER_USER",
			getPropertyForCR("AMQ_CLUSTER_USER", cr, "clusteruser"),
			nil,
		},
		{
			"AMQ_CLUSTER_PASSWORD",
			getPropertyForCR("AMQ_CLUSTER_PASSWORD", cr, "clusterpass"),
			nil,
		},
		{
			"POD_NAMESPACE",
			"", // Set to the field metadata.namespace in current object
			nil,
		},
	}

	return envVarArray
}

func newEnvVarArrayForCR(cr *brokerv1alpha1.ActiveMQArtemis) *[]corev1.EnvVar {

	envVarArray := makeEnvVarArrayForCR(cr)

	return &envVarArray
}

func newPodTemplateSpecForCR(cr *brokerv1alpha1.ActiveMQArtemis) corev1.PodTemplateSpec {

	// Log where we are and what we're doing
	reqLogger := log.WithName(cr.Name)
	reqLogger.Info("Creating new pod template spec for custom resource")

	//var pts corev1.PodTemplateSpec

	pts := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    cr.Labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    cr.Name + "-container",
					Image:   cr.Spec.Image,
					Command: []string{"/opt/amq/bin/launch.sh", "start"},

					VolumeMounts: makeVlMntCR(cr),
					Env:          makeEnvVarArrayForCR(cr),
				},
			},

			Volumes: makeVolumesForCR(cr),
		},
	}

	return pts
}

func newStatefulSetForCR(cr *brokerv1alpha1.ActiveMQArtemis) *appsv1.StatefulSet {

	// Log where we are and what we're doing
	reqLogger := log.WithName(cr.Name)
	reqLogger.Info("Creating new statefulset for custom resource")

	var replicas int32 = 1

	labels := selectors.LabelsForActiveMQArtemis(cr.Name)

	ss := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name + "-ss", //"-statefulset",
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: nil,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "hs", //cr.Name + "-headless" + "-service",
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template:             newPodTemplateSpecForCR(cr),
			VolumeClaimTemplates: *pvc.NewPersistentVolumeClaimArrayForCR(cr, 1),
		},
	}

	return ss
}
func CreateStatefulSet(cr *brokerv1alpha1.ActiveMQArtemis, client client.Client, scheme *runtime.Scheme) (*appsv1.StatefulSet, error) {

	// Log where we are and what we're doing
	reqLogger := log.WithValues("ActiveMQArtemis Name", cr.Name)
	reqLogger.Info("Creating new statefulset")

	var err error = nil

	// Define the StatefulSet
	ss := newStatefulSetForCR(cr)
	// Set ActiveMQArtemis instance as the owner and controller
	if err = controllerutil.SetControllerReference(cr, ss, scheme); err != nil {
		// Add error detail for use later
		reqLogger.Info("Failed to set controller reference for new " + "statefulset")
	}
	reqLogger.Info("Set controller reference for new " + "statefulset")

	// Call k8s create for statefulset
	if err = client.Create(context.TODO(), ss); err != nil {
		// Add error detail for use later
		reqLogger.Info("Failed to creating new " + "statefulset")
	}
	reqLogger.Info("Created new " + "statefulset")

	return ss, err
}
func RetrieveStatefulSet(statefulsetName string, namespacedName types.NamespacedName, client client.Client) (*appsv1.StatefulSet, error) {

	// Log where we are and what we're doing
	reqLogger := log.WithValues("ActiveMQArtemis Name", namespacedName.Name)
	reqLogger.Info("Retrieving " + "statefulset")

	var err error = nil
	//ss := &appsv1.StatefulSet{
	//	TypeMeta: metav1.TypeMeta{},
	//	ObjectMeta: metav1.ObjectMeta {
	//		Name: statefulsetName,
	//	},
	//}

	labels := selectors.LabelsForActiveMQArtemis("example-activemqartemis")

	ss := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        statefulsetName,
			Namespace:   namespacedName.Namespace,
			Labels:      labels,
			Annotations: nil,
		},
	}
	// Check if the headless service already exists
	if err = client.Get(context.TODO(), namespacedName, ss); err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Error(err, "Statefulset claim IsNotFound", "Namespace", namespacedName.Namespace, "Name", namespacedName.Name)
		} else {
			reqLogger.Error(err, "Statefulset claim found", "Namespace", namespacedName.Namespace, "Name", namespacedName.Name)
		}
	}

	return ss, err
}
