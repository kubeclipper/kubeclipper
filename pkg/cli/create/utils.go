package create

import (
	"context"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

func getComponentAvailableVersions(client *kc.Client, offline bool, component string) sets.Set[string] {
	var metas *kc.ComponentMeta
	var err error
	if offline {
		metas, err = client.GetComponentMeta(context.TODO(), map[string][]string{"online": {"false"}})
	} else {
		metas, err = client.GetComponentMeta(context.TODO(), map[string][]string{"online": {"true"}})
	}
	if err != nil {
		logger.Errorf("get component meta failed: %s. please check .kc/config", err)
		return nil
	}
	set := sets.New[string]()
	for _, resource := range metas.Addons {
		if resource.Name == component {
			set.Insert(resource.Version)
		}
	}
	return set
}
