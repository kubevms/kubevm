package controller

import (
	"github.com/randomswdev/kubevm/pkg/controller/virtualmachinedeployment"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, virtualmachinedeployment.Add)
}
