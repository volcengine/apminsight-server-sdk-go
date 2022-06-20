package main

import (
	"log"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/callopt"
	"github.com/gin-gonic/gin"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex/example/server/kitex_gen/api"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex/example/server/kitex_gen/api/hello"
)

func caller(ginC *gin.Context) {
	c, err := hello.NewClient("example_rpc_service", //set destService. important
		client.WithHostPorts("0.0.0.0:8888"),
		client.WithSuite(kitex.NewClientSuite(aitracer.GlobalTracer())))
	if err != nil {
		log.Fatal(err)
	}

	req := &api.Request{Message: "Is trace on?"}

	// use gin.Request.Context()
	resp, err := c.Echo(ginC.Request.Context(), req, callopt.WithRPCTimeout(3*time.Second))

	if err != nil {
		log.Println(err)
		ginC.JSON(200, "rpc fail")
		return
	}
	log.Println(resp)
	ginC.JSON(200, "rpc success")
}
