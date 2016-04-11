package limbo

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/apex/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/limbo-services/protobuf/proto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
)

const metadataHeaderPrefix = "Grpc-Metadata-"

type grpcGatewayFlag struct{}

func GetHTTPRule(method *pb.MethodDescriptorProto) *HttpRule {
	if method.Options == nil {
		return nil
	}
	v, _ := proto.GetExtension(method.Options, E_Http)
	if v == nil {
		return nil
	}
	return v.(*HttpRule)
}

/*
AnnotateContext adds context information such as metadata from the request.

If there are no metadata headers in the request, then the context returned
will be the same context.
*/
func annotateContext(ctx context.Context, req *http.Request) context.Context {
	ctx = context.WithValue(ctx, grpcGatewayFlag{}, true)

	var pairs []string
	for key, vals := range req.Header {
		for _, val := range vals {
			if key == "Authorization" {
				pairs = append(pairs, key, val)
				continue
			}
			if strings.HasPrefix(key, metadataHeaderPrefix) {
				pairs = append(pairs, key[len(metadataHeaderPrefix):], val)
			}
		}
	}

	if len(pairs) == 0 {
		return ctx
	}

	return metadata.NewContext(ctx, metadata.Pairs(pairs...))
}

func FromGateway(ctx context.Context) bool {
	v, ok := ctx.Value(grpcGatewayFlag{}).(bool)
	return v && ok
}

func respondWithGRPCError(w http.ResponseWriter, err error) {
	const fallback = `{"error": "failed to marshal error message"}`

	var (
		code   = grpc.Code(err)
		desc   = grpc.ErrorDesc(err)
		status = httpStatusFromCode(code)
		msg    struct {
			Error string `json:"error"`
		}
	)

	msg.Error = desc

	data, err := json.Marshal(&msg)
	if err != nil {
		log.WithError(err).Errorf("failed to marshal error message")
		status = http.StatusInternalServerError
		data = []byte(`{"error": "failed to marshal error message"}`)
		err = nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(data)
}

func httpStatusFromCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusForbidden
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}

	log.Errorf("Unknown GRPC error code: %v", code)
	return http.StatusInternalServerError
}
