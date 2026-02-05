package test

import (
	"fmt"
	"go-backend/service"
	"testing"
)

func TestFabricVer(t *testing.T) {
	res, err := service.GetAllFabricVersions()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(res)
}
