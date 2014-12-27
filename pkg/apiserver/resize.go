package apiserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/httplog"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
)

type ResizeHandler struct {
	canonicalPrefix string
	codec           runtime.Codec
	storage         map[string]RESTStorage
	ops             *Operations
	timeout         time.Duration
}

var supportedResizables = map[string]bool{
	"replicationControllers": true,
}

const (
	paramInc       string = "inc"
	paramNamespace string = "namespace"
	urlDelimiter   string = "/"
)

func (h *ResizeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	pathParts := strings.SplitN(req.URL.Path, urlDelimiter, 3)

	if len(pathParts) < 2 {
		notFound(w, req)
		return
	}

	resizableType := pathParts[0]
	resizableName := pathParts[1]
	ctx := h.contextFromRequest(req)
	inc := req.URL.Query().Get(paramInc)

	if !h.isSupportedType(resizableType) {
		badRequest("The requested resource does not support resizing", w)
		return
	}

	storage := h.storage[resizableType]
	if storage == nil {
		httplog.LogOf(req, w).Addf("'%v' has no storage object", resizableType)
		notFound(w, req)
		return
	}

	resizableStorage, ok := storage.(Resizable)

	//shouldn't ever get here if the supported types are correct, but check anyway
	if !ok {
		badRequest("The requested resource does not support resizing", w)
		return
	}

	i, err := strconv.ParseInt(inc, 10, 32)

	if err != nil {
		badRequest(fmt.Sprintf("Unable to convert %v to an integer", inc), w)
		return
	}

	out, err := resizableStorage.Resize(ctx, resizableName, int(i))

	if err != nil {
		errorJSON(err, h.codec, w)
		return
	}

	op := createOperation(h.ops, out, true, h.timeout, nil, 0)
	finishReq(op, req, w, h.codec)
}

func (h *ResizeHandler) contextFromRequest(req *http.Request) api.Context {
	namespace := req.URL.Query().Get(paramNamespace)
	ctx := api.WithNamespaceDefaultIfNone(api.WithNamespace(api.NewDefaultContext(), namespace))
	return ctx
}

func (h *ResizeHandler) isSupportedType(s string) bool {
	_, ok := supportedResizables[s]
	return ok
}

func badRequest(msg string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(msg))
}
