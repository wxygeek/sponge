package handler

import (
	"math/rand"
	"text/template"
	"time"
)

func init() {
	var err error
	handlerLogicTmpl, err = template.New("handlerLogic").Parse(handlerLogicTmplRaw)
	if err != nil {
		panic(err)
	}
	routerTmpl, err = template.New("handlerRouter").Parse(routerTmplRaw)
	if err != nil {
		panic(err)
	}
	httpErrCodeTmpl, err = template.New("httpErrCode").Parse(httpErrCodeTmplRaw)
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())
}

var (
	handlerLogicTmpl    *template.Template
	handlerLogicTmplRaw = `// Code generated by https://github.com/zhufuyi/sponge

package handler

import (
	"context"

	//"github.com/zhufuyi/sponge/pkg/gin/middleware"

	serverNameExampleV1 "moduleNameExample/api/serverNameExample/v1"
)

{{- range .PbServices}}

var _ serverNameExampleV1.{{.Name}}Logicer = (*{{.LowerName}}Handler)(nil)

type {{.LowerName}}Handler struct {
	// example: 
	// 	{{.LowerName}}Dao dao.{{.Name}}Dao
}

// New{{.Name}}Handler create a handler
func New{{.Name}}Handler() serverNameExampleV1.{{.Name}}Logicer {
	return &{{.LowerName}}Handler{
		// example:
		// 	{{.LowerName}}Dao: dao.New{{.Name}}Dao(
		// 		model.GetDB(),
		// 		cache.New{{.Name}}Cache(model.GetCacheType()),
		// 	),
	}
}

{{- range .Methods}}

{{if eq .InvokeType 0}}{{if .Path}}{{.Comment}}
func (h *{{.LowerServiceName}}Handler) {{.MethodName}}(ctx context.Context, req *serverNameExampleV1.{{.Request}}) (*serverNameExampleV1.{{.Reply}}, error) {
	panic("implement me")

	// fill in the business logic code here
	// example:
	//	    err := req.Validate()
	//	    if err != nil {
	//		    logger.Warn("req.Validate error", logger.Err(err), logger.Any("req", req), middleware.CtxRequestIDField(ctx))
	//		    return nil, ecode.InvalidParams.Err()
	//	    }
	//
	// 	reply, err := h.{{.LowerServiceName}}Dao.{{.MethodName}}(ctx, &model.{{.ServiceName}}{
{{- range .RequestFields}}
	//     	{{.Name}}: req.{{.Name}},
{{- end}}
	//     })
	// 	if err != nil {
	//			logger.Warn("{{.MethodName}} error", logger.Err(err), middleware.CtxRequestIDField(ctx))
	//			return nil, ecode.InternalServerError.Err()
	//		}
	//
	//     return &serverNameExampleV1.{{.Reply}}{
{{- range .ReplyFields}}
	//     	{{.Name}}: reply.{{.Name}},
{{- end}}
	//     }, nil
}{{end}}{{end}}

{{- end}}

// ---------- Do not delete or move this split line, this is the merge code marker ----------

{{- end}}
`

	routerTmpl    *template.Template
	routerTmplRaw = `// Code generated by https://github.com/zhufuyi/sponge

package routers

import (
	"github.com/gin-gonic/gin"

	"github.com/zhufuyi/sponge/pkg/logger"
	//"github.com/zhufuyi/sponge/pkg/middleware"

	serverNameExampleV1 "moduleNameExample/api/serverNameExample/v1"
	"moduleNameExample/internal/handler"
)

func init() {
	allMiddlewareFns = append(allMiddlewareFns, func(c *middlewareConfig) {
{{- range .PbServices}}
		{{.LowerName}}Middlewares(c)
{{- end}}
	})

	allRouteFns = append(allRouteFns,
		func(r *gin.Engine, groupPathMiddlewares map[string][]gin.HandlerFunc, singlePathMiddlewares map[string][]gin.HandlerFunc) {
{{- range .PbServices}}
			{{.LowerName}}Router(r, groupPathMiddlewares, singlePathMiddlewares, handler.New{{.Name}}Handler())
{{- end}}
		})
}

{{- range .PbServices}}

func {{.LowerName}}Router(
	r *gin.Engine,
	groupPathMiddlewares map[string][]gin.HandlerFunc,
	singlePathMiddlewares map[string][]gin.HandlerFunc,
	iService serverNameExampleV1.{{.Name}}Logicer) {
	serverNameExampleV1.Register{{.Name}}Router(
		r,
		groupPathMiddlewares,
		singlePathMiddlewares,
		iService,
		serverNameExampleV1.With{{.Name}}HTTPResponse(),
		serverNameExampleV1.With{{.Name}}Logger(logger.Get()),
		serverNameExampleV1.With{{.Name}}ErrorToHTTPCode(
			// Set some error codes to standard http return codes,
			// by default there is already ecode.InternalServerError and ecode.ServiceUnavailable
			// example:
			// 	ecode.Forbidden, ecode.LimitExceed,
		),
	)
}

// you can set the middleware of a route group, or set the middleware of a single route, 
// or you can mix them, pay attention to the duplication of middleware when mixing them, 
// it is recommended to set the middleware of a single route in preference
func {{.LowerName}}Middlewares(c *middlewareConfig) {
	// set up group route middleware, group path is left prefix rules,
	// if the left prefix is hit, the middleware will take effect, e.g. group route is /api/v1, route /api/v1/{{.LowerName}}/:id  will take effect
	// c.setGroupPath("/api/v1/{{.LowerName}}", middleware.Auth())

	// set up single route middleware, just uncomment the code and fill in the middlewares, nothing else needs to be changed
{{- range .Methods}}
	{{if eq .InvokeType 0}}{{if .Path}}//c.setSinglePath("{{.Method}}", "{{.Path}}", middleware.Auth()){{end}}{{end}}
{{- end}}
}

// ---------- Do not delete or move this split line, this is the merge code marker ----------

{{- end}}
`

	httpErrCodeTmpl *template.Template
	//nolint
	httpErrCodeTmplRaw = `// Code generated by https://github.com/zhufuyi/sponge

package ecode

import (
	"github.com/zhufuyi/sponge/pkg/errcode"
)

{{- range .PbServices}}

// {{.LowerName}} business-level http error codes.
// the {{.LowerName}}NO value range is 1~100, if the same number appears, it will cause a failure to start the service.
var (
	{{.LowerName}}NO       = {{.RandNumber}}
	{{.LowerName}}Name     = "{{.LowerName}}"
	{{.LowerName}}BaseCode = errcode.HCode({{.LowerName}}NO)
// --blank line--
{{- range $i, $v := .Methods}}
	{{if eq .InvokeType 0}}{{if .Path}}Err{{.MethodName}}{{.ServiceName}}   = errcode.NewError({{.LowerServiceName}}BaseCode+{{$v.AddOne $i}}, "failed to {{.MethodName}} "+{{.LowerServiceName}}Name){{end}}{{end}}
{{- end}}
	// error codes are globally unique, adding 1 to the previous error code
)

// ---------- Do not delete or move this split line, this is the merge code marker ----------

{{- end}}
`
)
