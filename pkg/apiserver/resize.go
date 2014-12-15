package apiserver

import (
	"strings"
	"strconv"

	"net/http"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/httplog"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)


type ResizeHandler struct {
	canonicalPrefix	string
	codec	runtime.Codec
	storage map[string]RESTStorage
}

var supportedResizables = map[string] bool {
	"replicationControllers": true,
}

func (h *ResizeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	namespace := req.URL.Query().Get("namespace")
	ctx := api.WithNamespaceDefaultIfNone(api.WithNamespace(api.NewDefaultContext(), namespace))
	replicaCount := req.URL.Query().Get("replicas")
	inc := req.URL.Query().Get("inc")

	pathParts := strings.SplitN(req.URL.Path, "/", 3)

	if len(pathParts) < 2 {
		notFound(w, req)
		return
	}

	resizableType := pathParts[0]
	resizableName := pathParts[1]

	if !h.isSupportedType(resizableType) {
		httplog.LogOf(req, w).Addf("%s is not a supported resizable type", resizableType)
		notFound(w, req)
		return
	}

	storage := h.storage[resizableType]
	if storage == nil {
		httplog.LogOf(req, w).Addf("'%v' has no storage object", resizableType)
		notFound(w, req)
		return
	}

	resizableStorage, ok := storage.(Resizable)

	if !ok {
		w.Write([]byte("failed storage"))
		//todo handle the error
	}

	if len(replicaCount) > 0 {
		//todo handle the error
//		i, _ := strconv.ParseInt(replicaCount, 10, 32)
////		h.handleSetReplicaCount(i)
//
//
	} else {
		//todo handle the error
		i, _ := strconv.ParseInt(inc, 10, 32)
//		h.handleIncrement(i)

//		out, err := resizableStorage.Resize(ctx, resizableName, int(i))
		_, err := resizableStorage.Resize(ctx, resizableName, int(i))

		if err != nil {
			errorJSON(err, h.codec, w)
			return
		}

		obj, err := storage.Get(ctx, resizableName)

		if err != nil {
			writeJSON(http.StatusInternalServerError, h.codec, obj, w)
			return
		}

		//todo this isn't updated right away so it doesn't immediately show the desired
		//state, should I just return nothing or ok or something?
		writeJSON(http.StatusOK, h.codec, obj, w)
	}
}

func (h *ResizeHandler) validateParameters(replicaCount string, inc string) bool {
	//are they integers
	//is only one set - warn, replica count will override increment behavior
	return true
}

func (h *ResizeHandler) isSupportedType(s string) bool {
	_, ok := supportedResizables[s]
	return ok
}
