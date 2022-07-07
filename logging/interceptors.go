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
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
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
	return &reporter{
		ctx:             ctx,
		typ:             typ,
		service:         service,
		method:          method,
		startCallLogged: false,
		opts:            r.opts,
		logger:          r.logger.With(fields...),
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
	switch shouldLog(c.opts.shouldLog, interceptors.FullMethod(c.service, c.method), err).Enable {
	case logging.LogFinishCall, logging.LogStartAndFinishCall:
		if errors.Is(err, io.EOF) {
			err = nil
		}
		c.logMessage(c.logger, err, "finished call", duration)
	default:
		return
	}
}

// PostMsgSend implement
func (c *reporter) PostMsgSend(resp interface{}, err error, duration time.Duration) {
	if c.startCallLogged {
		return
	}
	payloadDecision := shouldLog(c.opts.shouldLog, interceptors.FullMethod(c.service, c.method), err)
	if err == nil {
		if payloadDecision.Response {
			c.logger = c.logger.With("response_"+keyContent, bodyString(resp, payloadDecision.ClearBytes))
		}
		if c.opts.responseFieldExtractorFunc != nil {
			data := c.opts.responseFieldExtractorFunc(interceptors.FullMethod(c.service, c.method), resp)
			c.logger = c.logger.With(extractMap(data, "response_", payloadDecision.Response)...)
		}
	}
	if payloadDecision.Enable == logging.LogStartAndFinishCall {
		c.startCallLogged = true
		c.logMessage(c.logger, err, "started call", duration)
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
		return PayloadDecision{
			Enable: logging.NoLogCall,
		}
	}
	return decider(fullMethod, err)
}

// PostMsgReceive implement
func (c *reporter) PostMsgReceive(req interface{}, err error, duration time.Duration) {
	if c.startCallLogged {
		return
	}
	if c.opts.requestFieldExtractorFunc != nil {
		if valMap := c.opts.requestFieldExtractorFunc(interceptors.FullMethod(c.service, c.method), req); valMap != nil {
			t := tags.Extract(c.ctx)
			for k, v := range valMap {
				t.Set("request."+k, v)
			}
		}
	}
	payloadDecision := shouldLog(c.opts.shouldLog, interceptors.FullMethod(c.service, c.method), err)
	if payloadDecision.Request && err == nil {
		c.logger = c.logger.With(keyRequestContent, bodyString(req, payloadDecision.ClearBytes))
	}
	if payloadDecision.Enable == logging.LogStartAndFinishCall {
		c.startCallLogged = true
		c.logMessage(c.logger, err, "started call", duration)
	}
}

func (c *reporter) logMessage(logger logging.Logger, err error, msg string, duration time.Duration) {
	logLevel := logging.INFO
	if err != nil {
		statusError, _ := status.FromError(err)
		logger = logger.With("status", strconv.Itoa(int(statusError.Code())))
		msg = statusError.Message()
		logLevel = c.opts.levelFunc(statusError.Code())
	}
	logger = logger.With(extractFields(tags.Extract(c.ctx), c.hasLoggingRequestContent)...)
	logger.With("time_ms", fmt.Sprintf("%v", float32(duration.Nanoseconds()/1000)/1000)).Log(logLevel, msg)
}

const keyContent = "content"

var keyRequestContent = "request." + keyContent

// extractFields returns all fields from tags.
func extractFields(data tags.Tags, skipContentKey bool) logging.Fields {
	var fields logging.Fields
	for k, v := range data.Values() {
		if skipContentKey && k == keyRequestContent {
			continue
		}
		fields = append(fields, k, v)
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
