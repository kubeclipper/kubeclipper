package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

type ForEachObjectInYAMLActionFunc func(context.Context, *discovery.DiscoveryClient, dynamic.Interface, *Object) error

var applyfn = func(ctx context.Context, dc *discovery.DiscoveryClient, dyn dynamic.Interface, obj *Object) error {

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	mapping, err := mapper.RESTMapping(obj.GVK.GroupKind(), obj.GVK.Version)
	if err != nil {
		return err
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dyn.Resource(mapping.Resource).Namespace(obj.Data.GetNamespace())
	} else {
		dr = dyn.Resource(mapping.Resource)
	}

	data, err := json.Marshal(obj.Data)
	if err != nil {
		return err
	}

	_, err = dr.Patch(ctx, obj.Data.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "kubeclipper",
	})

	return err
}

var deletefn = func(ctx context.Context, dc *discovery.DiscoveryClient, dyn dynamic.Interface, obj *Object) error {

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	mapping, err := mapper.RESTMapping(obj.GVK.GroupKind(), obj.GVK.Version)
	if err != nil {
		return err
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dyn.Resource(mapping.Resource).Namespace(obj.Data.GetNamespace())
	} else {
		dr = dyn.Resource(mapping.Resource)
	}

	err = dr.Delete(ctx, obj.Data.GetName(), metav1.DeleteOptions{})

	return err
}

func emulateKubectlHandler(writer http.ResponseWriter, request *http.Request, action Action, restConfig *restclient.Config) {
	if request.Method != http.MethodPost {
		http.Error(writer, "only POST is supported", http.StatusMethodNotAllowed)
		return
	}
	dc, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	var fn ForEachObjectInYAMLActionFunc
	switch action {
	case ActionApply:
		fn = applyfn
	case ActionDelete:
		fn = deletefn
	default:
		http.Error(writer, "unknown action", http.StatusInternalServerError)
	}

	if errs := ForEachObjectInYAML(context.Background(), dc, dyn, body, fn); errs != nil {
		http.Error(writer, errorsToString(errs), http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
}

func errorsToString(errs []error) string {
	var buf bytes.Buffer
	for _, err := range errs {
		buf.WriteString(err.Error())
		if err != errs[len(errs)-1] {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

type Object struct {
	Data *unstructured.Unstructured
	GVK  *schema.GroupVersionKind
}

func decodeYAML(data []byte) (<-chan *Object, <-chan error) {

	var (
		chanErr        = make(chan error)
		chanObj        = make(chan *Object)
		multidocReader = utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	)

	go func() {
		defer close(chanErr)
		defer close(chanObj)

		for {
			buf, err := multidocReader.Read()
			if err != nil {
				if err == io.EOF {
					return
				}
				chanErr <- errors.Wrap(err, "failed to read yaml data")
				return
			}

			obj := &unstructured.Unstructured{}
			_, gvk, err := decUnstructured.Decode(buf, nil, obj)
			if err != nil {
				chanErr <- errors.Wrap(err, "failed to unmarshal yaml data")
				return
			}

			chanObj <- &Object{
				Data: obj,
				GVK:  gvk,
			}
		}
	}()

	return chanObj, chanErr
}

func ForEachObjectInYAML(
	ctx context.Context,
	dc *discovery.DiscoveryClient,
	dyn dynamic.Interface,
	data []byte,
	actionFn ForEachObjectInYAMLActionFunc) []error {
	var errs []error
	chanObj, chanErr := decodeYAML(data)
	for {
		select {
		case obj := <-chanObj:
			if obj == nil {
				return errs
			}
			if err := actionFn(ctx, dc, dyn, obj); err != nil {
				errs = append(errs, err)
			}
		case err := <-chanErr:
			if err == nil {
				return errs
			}
			errs = append(errs, errors.Wrap(err, "received error while decoding yaml"))
			return errs
		}
	}
}
