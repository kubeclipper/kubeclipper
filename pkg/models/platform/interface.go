/*
 *
 *  * Copyright 2021 KubeClipper Authors.
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

//go:generate mockgen -destination mock/mock_platform.go -source interface.go Operator

package platform

import (
	"context"

	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type Operator interface {
	Reader
	Writer

	EventReader
	EventWriter
}

type Reader interface {
	GetPlatformSetting(ctx context.Context) (*v1.PlatformSetting, error)
	ReaderEx
}

type Writer interface {
	CreatePlatformSetting(ctx context.Context, platformSetting *v1.PlatformSetting) (*v1.PlatformSetting, error)
	UpdatePlatformSetting(ctx context.Context, platformSetting *v1.PlatformSetting) (*v1.PlatformSetting, error)
}

type ReaderEx interface{}

type EventReader interface {
	ListEvents(ctx context.Context, query *query.Query) (*v1.EventList, error)
	GetEvent(ctx context.Context, name string) (*v1.Event, error)
	EventReaderEx
}

type EventReaderEx interface {
	GetEventEx(ctx context.Context, name string, resourceVersion string) (*v1.Event, error)
	ListEventsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
}

type EventWriter interface {
	CreateEvent(ctx context.Context, Event *v1.Event) (*v1.Event, error)
	DeleteEvent(ctx context.Context, name string) error
	DeleteEventCollection(ctx context.Context, query *query.Query) error
}
