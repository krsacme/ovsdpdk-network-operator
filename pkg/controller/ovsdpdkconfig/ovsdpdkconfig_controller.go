package ovsdpdkconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	ovsdpdkv1 "github.com/krsacme/ovsdpdk-network-operator/pkg/apis/ovsdpdknetwork/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Masterminds/sprig"
)

const (
	OVSDPDK_NETWORK_PREPARE_DS = "./bindata/manifests/prepare/daemonset.yaml"
	CONFIG_MAP_KEY_INTERFACE   = "interface"
	CONFIG_MAP_KEY_NODE        = "node"
)

type RenderData struct {
	Funcs template.FuncMap
	Data  map[string]interface{}
}

func MakeRenderData() RenderData {
	return RenderData{
		Funcs: template.FuncMap{},
		Data:  map[string]interface{}{},
	}
}

var log = logf.Log.WithName("controller_ovsdpdkconfig")

// Add creates a new OvsDpdkConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileOvsDpdkConfig{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("ovsdpdkconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OvsDpdkConfig
	err = c.Watch(&source.Kind{Type: &ovsdpdkv1.OvsDpdkConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSet and requeue the owner OvsDpdkConfig
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovsdpdkv1.OvsDpdkConfig{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSet and requeue the owner OvsDpdkConfig
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovsdpdkv1.OvsDpdkConfig{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileOvsDpdkConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOvsDpdkConfig{}

// ReconcileOvsDpdkConfig reconciles a OvsDpdkConfig object
type ReconcileOvsDpdkConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a OvsDpdkConfig object and makes changes based on the state read
// and what is in the OvsDpdkConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOvsDpdkConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling OvsDpdkConfig")

	// Fetch the OvsDpdkConfig instance
	instance := &ovsdpdkv1.OvsDpdkConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if len(instance.Spec.NodeSelectorLabels) == 0 {
		err := fmt.Errorf("NodeSelectorLabels is mandatory to run OvS-DPDK")
		reqLogger.Error(err, "NodeSelectorLabels is empty")
		return reconcile.Result{}, err
	}

	mLabels := client.MatchingLabels(instance.Spec.NodeSelectorLabels)
	log.Info("Get Node Selector labels", "Labels", mLabels)

	// Fetch the Nodes
	nodeList := &corev1.NodeList{}
	err = r.client.List(context.TODO(), nodeList, mLabels)
	if err != nil {
		reqLogger.Error(err, "Failed to list nodes")
		return reconcile.Result{}, err
	}

	for _, item := range nodeList.Items {
		reqLogger.Info("List of selected nodes to run OvS-DPDK", "Node", item.Name)
		// Node section does not have any use now, its only for log
	}

	objKey := types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}

	// Create a ConfigMap with the node and interface configs and use the ConfigMap object in the prepare command
	cm, err := r.newConfigMapForCR(instance, objKey)
	if err != nil {
		reqLogger.Error(err, "Failed to create ConfigMap object")
		return reconcile.Result{}, err
	}

	// Set OvsDpdkConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, cm, r.scheme); err != nil {
		reqLogger.Error(err, "Failed to set controller reference to ConfigMap")
		return reconcile.Result{}, err
	}

	// Create or Update the ConfigMap object
	configMapUpdated := false
	found := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), objKey, found)
	if err != nil && errors.IsNotFound(err) {
		// Not found, create ConfigMap
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.client.Create(context.TODO(), cm)
		if err != nil {
			reqLogger.Error(err, "Failed to create ConfigMap")
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get ConfigMap object")
		return reconcile.Result{}, err
	} else {
		// ConfigMap exists, update it
		var foundIface []ovsdpdkv1.InterfaceConfig
		var foundNode ovsdpdkv1.NodeConfig

		err = json.Unmarshal([]byte(string(found.Data[CONFIG_MAP_KEY_INTERFACE])), &foundIface)
		if err != nil {
			reqLogger.Error(err, "Failed to Unmarshall interface config")
			return reconcile.Result{}, err
		}

		err = json.Unmarshal([]byte(string(found.Data[CONFIG_MAP_KEY_NODE])), &foundNode)
		if err != nil {
			reqLogger.Error(err, "Failed to Unmarshall node config")
			return reconcile.Result{}, err
		}

		if !reflect.DeepEqual(foundIface, instance.Spec.InterfaceConfig) || !reflect.DeepEqual(foundNode, instance.Spec.NodeConfig) {
			log.Info("ConfigMap exits, Config updated", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			configMapUpdated = true
			err = r.client.Update(context.TODO(), cm)
			if err != nil {
				reqLogger.Error(err, "Failed to update ConfigMap")
				return reconcile.Result{}, err
			}
		} else {
			log.Info("ConfigMap exits, Config is same")
		}
	}

	// Define a new DaemeonSet object
	err = r.syncDaemonSetForCR(instance, objKey, configMapUpdated)
	if err != nil {
		reqLogger.Error(err, "Failed to sync DaemonSet for OvSDPDK Prepare")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Reconcile successful")
	return reconcile.Result{}, nil
}

func (r *ReconcileOvsDpdkConfig) newConfigMapForCR(cr *ovsdpdkv1.OvsDpdkConfig, objKey types.NamespacedName) (*corev1.ConfigMap, error) {
	interfaceConfig, err := json.Marshal(cr.Spec.InterfaceConfig)
	if err != nil {
		log.Error(err, "Failed to Marshal InterfaceConfig")
		return nil, err
	}

	nodeConfig, err := json.Marshal(cr.Spec.NodeConfig)
	if err != nil {
		log.Error(err, "Failed to Marshal NodeConfig")
		return nil, err
	}

	configData := make(map[string]string)
	configData[CONFIG_MAP_KEY_INTERFACE] = string(interfaceConfig)
	configData[CONFIG_MAP_KEY_NODE] = string(nodeConfig)
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      objKey.Name,
			Namespace: objKey.Namespace,
		},
		Data: configData,
	}, nil
}

func (r *ReconcileOvsDpdkConfig) getOpertaorImage(objKey types.NamespacedName) (string, error) {
	// Fetch the operator Deployment
	deployment := &appsv1.Deployment{}
	objKey.Name = "ovsdpdk-network-operator"
	err := r.client.Get(context.TODO(), objKey, deployment)
	if err != nil {
		log.Error(err, "Failed to get operator Deployment object")
		return "", err
	}

	image := deployment.Spec.Template.Spec.Containers[0].Image
	newImage := strings.Replace(image, "ovsdpdk-network-operator", "ovsdpdk-network-prepare", 1)
	return newImage, nil
}

func (r *ReconcileOvsDpdkConfig) syncDaemonSetForCR(cr *ovsdpdkv1.OvsDpdkConfig, objKey types.NamespacedName, configMapUpdated bool) error {
	image, err := r.getOpertaorImage(objKey)
	if err != nil {
		return err
	}

	data := MakeRenderData()
	data.Data["Name"] = objKey.Name
	data.Data["Namespace"] = objKey.Namespace
	data.Data["Image"] = image
	data.Data["NodeSelector"] = cr.Spec.NodeSelectorLabels
	data.Data["ReleaseVersion"] = os.Getenv("RELEASEVERSION")
	data.Data["ResourcePrefix"] = os.Getenv("RESOURCE_PREFIX")

	obj, err := r.renderDsForCR(OVSDPDK_NETWORK_PREPARE_DS, &data)
	if err != nil {
		log.Error(err, "Fail to render OvS-DPDK Prepare DaemonSet manifests")
		return err
	}

	if obj.GetKind() != "DaemonSet" {
		err = fmt.Errorf("Only DaemonSet Kind is expected")
		log.Error(err, "Invalid Kind", "Kind", obj.GetKind())
		return err
	}

	scheme := kscheme.Scheme
	ds := &appsv1.DaemonSet{}
	err = scheme.Convert(obj, ds, nil)
	if err != nil {
		log.Error(err, "Failed to convert to DaemonSet")
		return err
	}

	ds.Spec.Template.Spec.NodeSelector = cr.Spec.NodeSelectorLabels

	// Set OvsDpdkConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, ds, r.scheme); err != nil {
		return err
	}

	// Check if this DaemonSet already exists
	found := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), objKey, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new DaemonSet", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
		err = r.client.Create(context.TODO(), ds)
		if err != nil {
			log.Error(err, "Failed to Create DaemonSet object")
			return err
		}

		// DaemonSet created successfully - don't requeue
		return nil
	} else if err != nil {
		log.Error(err, "Failed get DaemonSet object")
		return err
	} else if configMapUpdated {
		// DaemonSet is existing, if ConfigMap is updated, update DaemonSet too
		ds.Spec.Template.Labels["dirty"] = strconv.FormatInt(time.Now().Unix(), 10)
		err = r.client.Update(context.TODO(), ds)
		if err != nil {
			log.Error(err, "Failed to update DaemonSet object")
			return err
		}
	}

	return nil
}

func (r *ReconcileOvsDpdkConfig) renderDsForCR(path string, d *RenderData) (*uns.Unstructured, error) {
	tmpl := template.New(path).Option("missingkey=error")
	if d.Funcs != nil {
		tmpl.Funcs(d.Funcs)
	}

	// Add universal functions
	tmpl.Funcs(template.FuncMap{"getOr": getOr, "isSet": isSet})
	tmpl.Funcs(sprig.TxtFuncMap())

	source, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error(err, "Failed to read file", "Path", path)
		return nil, err
	}

	if _, err := tmpl.Parse(string(source)); err != nil {
		log.Error(err, "Failed to parse template")
		return nil, err
	}

	rendered := bytes.Buffer{}
	if err := tmpl.Execute(&rendered, d.Data); err != nil {
		log.Error(err, "Failed to render template")
		return nil, err
	}

	// special case - if the entire file is whitespace, skip
	if len(strings.TrimSpace(rendered.String())) == 0 {
		log.V(2).Info("No content available")
		return nil, nil
	}

	obj := unstructured.Unstructured{}
	decoder := yaml.NewYAMLOrJSONDecoder(&rendered, 4096)
	for {
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			log.Error(err, "Failed to Decode content")
			return nil, err
		}
	}

	return &obj, nil
}

// getOr returns the value of m[key] if it exists, fallback otherwise.
// As a special case, it also returns fallback if the value of m[key] is
// the empty string
func getOr(m map[string]interface{}, key, fallback string) interface{} {
	val, ok := m[key]
	if !ok {
		return fallback
	}

	s, ok := val.(string)
	if ok && s == "" {
		return fallback
	}

	return val
}

// isSet returns the value of m[key] if key exists, otherwise false
// Different from getOr because it will return zero values.
func isSet(m map[string]interface{}, key string) interface{} {
	val, ok := m[key]
	if !ok {
		return false
	}
	return val
}
