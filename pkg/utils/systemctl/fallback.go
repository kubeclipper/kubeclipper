package systemctl

import (
	"context"

	"github.com/coreos/go-systemd/v22/dbus"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

// ListUnitsByNamesContextFallback 兼容旧版本 systemd
// Issue: https://github.com/coreos/go-systemd/issues/223
func ListUnitsByNamesContextFallback(ctx context.Context, conn *dbus.Conn, name string) ([]dbus.UnitStatus, error) {
	var fallback bool
	list := make([]dbus.UnitStatus, 0)
	units, err := conn.ListUnitsByNamesContext(ctx, []string{name})
	if err != nil {
		fallback = true
		logger.Debugf("ListUnitsByNames is not implemented in your systemd version (requires at least systemd 230), fallback to ListUnits: %v", err)
		// 注意：这个方法不会返回全部的 unit, disabled and inactive will thus not be returned.
		units, err = conn.ListUnitsContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	for _, unit := range units {
		if unit.Name == name {
			list = append(list, unit)
		}
	}

	// grab data on subscribed units that didn't show up in ListUnits in fallback mode, most
	// likely due to being inactive
	if fallback {
		// 因此对于旧版本的环境：我们要在根据 name 查询一次
		us, err := getUnitState(ctx, conn, name)
		if err != nil {
			return nil, err
		}
		list = append(list, *us)
	}

	return list, nil
}

func getUnitState(ctx context.Context, d *dbus.Conn, name string) (*dbus.UnitStatus, error) {
	info, err := d.GetUnitPropertiesContext(ctx, name)
	if err != nil {
		return nil, err
	}
	return &dbus.UnitStatus{
		Name:        name,
		LoadState:   info["LoadState"].(string),
		ActiveState: info["ActiveState"].(string),
		SubState:    info["SubState"].(string),
	}, nil
}
