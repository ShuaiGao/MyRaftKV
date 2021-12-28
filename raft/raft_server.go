package raft

import (
	"MyRaft/logger"
	"MyRaft/node"
	"MyRaft/node/rpc"
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
)

type DB struct {
	*echo.Echo
}

var db *DB

func GetDBInstance() *DB {
	return db
}

func CreateDB(address string, nodeIns *node.Node) {
	logger.Logger().Info("create_http_service", zap.String("address", address))
	e := echo.New()
	db = &DB{e}
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", hello)
	e.POST("/append/", AppendLog)
	e.Logger.Fatal(e.Start(address))
}

type AppendLogRet struct {
	code int    `json:"code"`
	msg  string `json:"msg"`
}

func AppendLog(c echo.Context) error {
	n := node.GetNode()
	if n.GetState() != node.StateLeader {
		return errors.New("node isn't leader")
	}
	log := c.QueryParam("log")
	tailItem := n.GetTailItem()
	n.Append(context.Background(), &rpc.AppendEntryRequest{
		PreIndex: uint32(tailItem.Index),
		PreTerm:  uint32(tailItem.Term),
		From:     n.Id,
		Entry:    &rpc.Entry{Term: uint32(n.Term), Index: uint32(tailItem.Index + 1), Value: log}})
	for _, node := range n.OtherNodeList {
		n.SendAppendEntry(node)
	}
	return c.JSON(http.StatusOK, &AppendLogRet{code: 0, msg: "ok"})
}
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
