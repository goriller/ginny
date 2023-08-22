package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type reportable struct {
	opts   *options
	logger logging.Logger
}

type reporter struct {
	ctx                      context.Context
	logger                   logging.Logger
	opts                     *options
	service                  string
	method                   string
	kind                     string
	typ                      interceptors.GRPCType
	startCallLogged          bool
	hasLoggingRequestContent bool
}

func (r *reportable) reporter(ctx context.Context, typ interceptors.GRPCType,
	service string, method string, kind string,
) (interceptors.Reporter, context.Context) {
	fields := commonFields(ctx, kind, service, method, typ)
	logging.InjectFields(ctx, fields)
	return &reporter{
		ctx:             ctx,
		typ:             typ,
		service:         service,
		method:          method,
		startCallLogged: false,
		opts:            r.opts,
		logger:          r.logger,
		kind:            kind,
	}, ctx
}

func commonFields(ctx context.Context, kind, service, method string, typ interceptors.GRPCType) logging.Fields {
	fields := logging.Fields{
		"action", "/" + service + "/" + method,
		"protocol", "grpc/" + kind + "/" + string(typ),
	}

	return fields
}

// PostCall implement
func (c *reporter) PostCall(err error, duration time.Duration) {
	switch shouldLog(c.opts.shouldLog, FullMethod(c.service, c.method), err).Enable {
	case logging.FinishCall:
		if errors.Is(err, io.EOF) {
			err = nil
		}
		c.logMessage(c.logger, err, "finished call", duration, nil)
	default:
		return
	}
}

// PostMsgSend implement
func (c *reporter) PostMsgSend(resp interface{}, err error, duration time.Duration) {
	if c.startCallLogged {
		return
	}
	payloadDecision := shouldLog(c.opts.shouldLog, FullMethod(c.service, c.method), err)
	fields := logging.Fields{}
	if err == nil {
		if payloadDecision.Response {
			fields.AppendUnique(logging.Fields{"response_" + keyContent, bodyString(resp, payloadDecision.ClearBytes)})
		}
		if c.opts.responseFieldExtractorFunc != nil {
			data := c.opts.responseFieldExtractorFunc(FullMethod(c.service, c.method), resp)
			fields.AppendUnique(extractMap(data, "response_", payloadDecision.Response))
		}
	}
	if payloadDecision.Enable == logging.PayloadSent {
		c.startCallLogged = true
		c.logMessage(c.logger, err, "started call", duration, fields)
	}
}

func bodyString(val interface{}, clearBytes bool) (logStr string) {
	if bodyProto, ok := val.(proto.Message); ok {
		if clearBytes {
			clearMessageBytes(proto.Clone(bodyProto).ProtoReflect())
		}
		b, err := protojson.Marshal(bodyProto)
		if err != nil {
			logStr = "error:" + err.Error()
		} else {
			if len(b) > 2048 {
				b = b[:2048]
			}
			logStr = string(b)
		}
	}
	return
}

func shouldLog(decider Decider, fullMethod string, err error) PayloadDecision {
	if strings.HasPrefix(fullMethod, "/grpc.health.v1.Health/") {
		return PayloadDecision{}
	}
	return decider(fullMethod, err)
}

func FullMethod(service, method string) string {
	return fmt.Sprintf("/%s/%s", service, method)
}

// PostMsgReceive implement
func (c *reporter) PostMsgReceive(req interface{}, err error, duration time.Duration) {
	if c.startCallLogged {
		return
	}
	fields := logging.Fields{}
	if c.opts.requestFieldExtractorFunc != nil {
		if valMap := c.opts.requestFieldExtractorFunc(FullMethod(c.service, c.method), req); valMap != nil {
			fields = logging.ExtractFields(c.ctx)
		}
	}
	payloadDecision := shouldLog(c.opts.shouldLog, FullMethod(c.service, c.method), err)
	if payloadDecision.Request && err == nil {
		fields.AppendUnique(logging.Fields{keyRequestContent, bodyString(req, payloadDecision.ClearBytes)})
	}
	if payloadDecision.Enable == logging.PayloadReceived {
		c.startCallLogged = true
		c.logMessage(c.logger, err, "started call", duration, fields)
	}
}

func (c *reporter) logMessage(logger logging.Logger, err error, msg string, duration time.Duration, fields logging.Fields) {
	logLevel := logging.LevelInfo
	if err != nil {
		statusError, _ := status.FromError(err)
		fields.AppendUnique(logging.Fields{"status", strconv.Itoa(int(statusError.Code()))})
		msg = statusError.Message()
		logLevel = c.opts.levelFunc(statusError.Code())
	}
	fields.AppendUnique(extractFields(logging.ExtractFields(c.ctx), c.hasLoggingRequestContent))
	fields.AppendUnique(logging.Fields{"time_ms", fmt.Sprintf("%v", float32(duration.Nanoseconds()/1000)/1000)})
	logger.Log(c.ctx, logLevel, msg)
}

const keyContent = "content"

var keyRequestContent = "request." + keyContent

// extractFields returns all fields from tags.
func extractFields(data logging.Fields, skipContentKey bool) logging.Fields {
	fields := logging.Fields{}
	ts := data.Iterator()
	for ts.Next() {
		k, v := ts.At()
		if skipContentKey && k == keyRequestContent {
			continue
		}
		fields = append(fields, logging.Fields{k, v})
	}
	return fields
}

// extractMap returns all fields from tags.
func extractMap(data map[string]string, prefix string, skipContentKey bool) logging.Fields {
	var fields logging.Fields
	for k, v := range data {
		if skipContentKey && k == keyContent {
			continue
		}
		fields = append(fields, prefix+k, v)
	}
	return fields
}
