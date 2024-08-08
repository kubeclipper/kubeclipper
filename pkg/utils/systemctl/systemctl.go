/*
 *
 *  * Copyright 2024 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package systemctl

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

const (
	defaultJobMode = "replace"

	loadStateNotFound   = "not-found"
	activeStateInactive = "inactive"
	activeStateActive   = "active"
)

// lookupUnitStatus returns the status of the unit if it exists.
// equivalent to `systemctl show <unit-name> --property=ActiveState`
func lookupUnitStatus(ctx context.Context, conn *dbus.Conn, name string) (*dbus.UnitStatus, error) {
	units, err := conn.ListUnitsByNamesContext(ctx, []string{name})
	if err != nil {
		return nil, fmt.Errorf("failed to lookup unit %s: %w", name, err)
	}
	if len(units) == 0 || units[0].LoadState == loadStateNotFound {
		return nil, nil
	}
	return &units[0], nil
}

// ReloadDeamon reloads the systemd daemon.
// equivalent to `systemctl daemon-reload`
func ReloadDeamon(ctx context.Context) error {
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect dbus: %w", err)
	}
	defer conn.Close()

	return conn.ReloadContext(ctx)
}

// EnableUnit enables the unit if it exists.
// equivalent to `systemctl enable <unit-name> --force`
func EnableUnit(ctx context.Context, name string) error {
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect dbus: %w", err)
	}
	defer conn.Close()

	us, err := lookupUnitStatus(ctx, conn, name)
	if err != nil {
		return err
	}
	if us == nil {
		return fmt.Errorf("unit %s not found", name)
	}

	if _, _, err = conn.EnableUnitFilesContext(ctx, []string{name}, false, true); err != nil {
		return fmt.Errorf("failed to enable unit %s: %w", name, err)
	}

	return nil
}

// DisableUnit disables the unit if it exists.
// equivalent to `systemctl disable <unit-name> --force`
func DisableUnit(ctx context.Context, name string) error {
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect dbus: %w", err)
	}
	defer conn.Close()

	us, err := lookupUnitStatus(ctx, conn, name)
	if err != nil || us == nil {
		return err
	}

	if _, err = conn.DisableUnitFilesContext(ctx, []string{name}, false); err != nil {
		return fmt.Errorf("failed to disable unit %s: %w", name, err)
	}

	return nil
}

// StartUnit starts a unit idempotently if it exists.
// equivalent to `systemctl start <unit-name>`
func StartUnit(ctx context.Context, name string) error {
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect dbus: %w", err)
	}
	defer conn.Close()

	us, err := lookupUnitStatus(ctx, conn, name)
	if err != nil {
		return err
	}
	if us == nil {
		return fmt.Errorf("unit %s not found", name)
	}
	if us.ActiveState == activeStateActive {
		logger.Debugf("unit %s is already active", name)
		return nil
	}

	res := make(chan string)
	if _, err := conn.StartUnitContext(ctx, name, defaultJobMode, res); err != nil {
		return fmt.Errorf("failed to start unit %s: %w", name, err)
	}
	if result := <-res; result != "done" {
		return fmt.Errorf("failed to start unit %s: %s", name, result)
	}

	return nil
}

// RestartUnit restarts a unit idempotently only if it exists.
// equivalent to `systemctl restart <unit-name>`
func RestartUnit(ctx context.Context, name string) error {
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect dbus: %w", err)
	}
	defer conn.Close()

	us, err := lookupUnitStatus(ctx, conn, name)
	if err != nil {
		return err
	}
	if us == nil {
		return fmt.Errorf("unit %s not found", name)
	}

	res := make(chan string)
	if _, err := conn.RestartUnitContext(ctx, name, defaultJobMode, res); err != nil {
		return fmt.Errorf("failed to restart unit %s: %w", name, err)
	}
	if result := <-res; result != "done" {
		return fmt.Errorf("failed to restart unit %s: %s", name, result)
	}

	return nil
}

// StartUnit stops a unit idempotently if it exists.
// equivalent to `systemctl stop <unit-name>`
func StopUnit(ctx context.Context, name string) error {
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect dbus: %w", err)
	}
	defer conn.Close()

	us, err := lookupUnitStatus(ctx, conn, name)
	if err != nil {
		return err
	}
	if us == nil {
		logger.Warnf("unit %s not found", name)
		return nil
	}
	if us.ActiveState == activeStateInactive {
		logger.Debugf("unit %s is already inactive", name)
		return nil
	}

	res := make(chan string)
	if _, err := conn.StopUnitContext(ctx, name, defaultJobMode, res); err != nil {
		return fmt.Errorf("failed to stop unit %s: %w", name, err)
	}
	if result := <-res; result != "done" {
		return fmt.Errorf("failed to stop unit %s: %s", name, result)
	}

	return nil
}

// ReloadUnit reloads a unit.
// equivalent to `systemctl reload <unit-name>`
func ReloadUnit(ctx context.Context, name string) error {
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect dbus: %w", err)
	}
	defer conn.Close()

	res := make(chan string)
	if _, err := conn.ReloadUnitContext(ctx, name, defaultJobMode, res); err != nil {
		return fmt.Errorf("failed to reload unit %s: %w", name, err)
	}
	if result := <-res; result != "done" {
		return fmt.Errorf("failed to reload unit %s: %s", name, result)
	}

	return nil
}
