package controller

import (
	"github.com/akashshinde/etcd-operator/pkg/controller/etcdcluster"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, etcdcluster.Add)
}
