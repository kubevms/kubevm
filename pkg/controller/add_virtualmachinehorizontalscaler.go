package controller

import (
	"github.com/randomswdev/kubevm/pkg/controller/virtualmachinehorizontalscaler"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, virtualmachinehorizontalscaler.Add)
}
