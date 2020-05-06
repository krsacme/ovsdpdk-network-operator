package controller

import (
	"github.com/krsacme/ovsdpdk-network-operator/pkg/controller/ovsdpdkconfig"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, ovsdpdkconfig.Add)
}
