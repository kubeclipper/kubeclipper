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

package backupstore

import (
	"bytes"
	"context"
	"io"
	"testing"
)

const (
	fsFilename = "fs.txt"
	rootDir    = "."
)

func TestFilesystemStore_Create(t *testing.T) {
	type fields struct {
		RootDir string
	}
	tests := []struct {
		name    string
		fields  fields
		want    BackupStore
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				RootDir: rootDir,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FilesystemStore{
				RootDir: tt.fields.RootDir,
			}
			_, err := fs.Create()
			if err != nil {
				t.Errorf("fs client create failed: %v", err)
			}
		})
	}
}

func TestFilesystemStore_Save(t *testing.T) {
	type fields struct {
		RootDir string
	}
	type args struct {
		ctx      context.Context
		r        io.Reader
		fileName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				RootDir: rootDir,
			},
			args: args{
				ctx:      context.TODO(),
				r:        bytes.NewReader([]byte("fs file test")),
				fileName: fsFilename,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FilesystemStore{
				RootDir: tt.fields.RootDir,
			}
			err := fs.Save(tt.args.ctx, tt.args.r, tt.args.fileName)
			if err != nil {
				t.Errorf("fs client save failed: %v", err)
			}
		})
	}
}

func TestFilesystemStore_Download(t *testing.T) {
	type fields struct {
		RootDir string
	}
	type args struct {
		ctx      context.Context
		fileName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				RootDir: rootDir,
			},
			args: args{
				ctx:      context.TODO(),
				fileName: fsFilename,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FilesystemStore{
				RootDir: tt.fields.RootDir,
			}
			w := &bytes.Buffer{}
			err := fs.Download(tt.args.ctx, tt.args.fileName, w)
			if (err != nil) != tt.wantErr {
				t.Errorf("Download() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("fs file content: %s", w.String())
		})
	}
}

func TestFilesystemStore_Delete(t *testing.T) {
	type fields struct {
		RootDir string
	}
	type args struct {
		ctx      context.Context
		fileName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				RootDir: rootDir,
			},
			args: args{
				ctx:      context.TODO(),
				fileName: fsFilename,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FilesystemStore{
				RootDir: tt.fields.RootDir,
			}
			err := fs.Delete(tt.args.ctx, tt.args.fileName)
			if err != nil {
				t.Errorf("fs client delete failed: %v", err)
			}
		})
	}
}
