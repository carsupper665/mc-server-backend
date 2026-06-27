package testplugin

import (
	"flag"
	"go-backend/common"
	"sync"

	"github.com/gin-gonic/gin"
)

func TestPlugin(c *gin.Context) {
	c.JSON(200, gin.H{"message": "this is example plugin setup."})
}

type Example struct {
	Store  map[string]TestUser
	logger *common.SysLogger
	mu     sync.RWMutex
}

type TestUser struct {
	Name string
	Age  int
}

type exampleAdd struct {
	Name string `form:"name"`
	Age  int    `form:"age"`
}

func (ex *Example) Add(c *gin.Context) {
	var req exampleAdd
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	ex.mu.Lock()
	defer ex.mu.Unlock()
	ex.Store[req.Name] = TestUser{req.Name, req.Age}
	c.JSON(200, gin.H{"message": "Add user, name: " + req.Name, "age": req.Age})
}

type exampleRemove struct {
	Name string `form:"name"`
}

var logPath = flag.String("Example Plugin Log", "Example_plugin_mgr", "specify the log name")

func NewExample() *Example {
	Logger, err := common.NewSysLogger("EP", logPath, 50000)
	if err != nil {
		panic(err)
	}
	return &Example{
		Store:  make(map[string]TestUser),
		logger: Logger,
	}
}

func (ex *Example) Remove(c *gin.Context) {
	ex.mu.Lock()
	defer ex.mu.Unlock()
	var req exampleRemove
	if err := c.ShouldBind(&req); err != nil {
		ex.logger.Errorf("bind error: %s", err.Error())
		c.Status(500)
		return
	}

	delete(ex.Store, req.Name)
	c.JSON(200, gin.H{"message": "Remove user, name: " + req.Name})
}

func (ex *Example) List(c *gin.Context) {
	ex.mu.RLock()
	defer ex.mu.RUnlock()
	users := ex.Store
	c.JSON(200, gin.H{"message": "List users: ", "count": len(users), "users": users})
}
