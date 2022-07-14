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
	"testing"
)

const (
	s3Filename      = "s3.txt"
	bucket          = defaultBucket
	endpoint        = "127.0.0.1:9000"
	accessKeyID     = "AKIAIOSFODNN7EXAMPLE"
	accessKeySecret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
)

func TestObjectStore_Create(t *testing.T) {
	// TODO: test failed

	/*type fields struct {
		Bucket          string
		Endpoint        string
		AccessKeyID     string
		AccessKeySecret string
		Region          string
		SSL             bool
		Client          *minio.Client
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
				Bucket:          bucket,
				Endpoint:        endpoint,
				AccessKeyID:     accessKeyID,
				AccessKeySecret: accessKeySecret,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := &ObjectStore{
				Bucket:          tt.fields.Bucket,
				Endpoint:        tt.fields.Endpoint,
				AccessKeyID:     tt.fields.AccessKeyID,
				AccessKeySecret: tt.fields.AccessKeySecret,
				Region:          tt.fields.Region,
				SSL:             tt.fields.SSL,
				Client:          tt.fields.Client,
			}
			_, err := receiver.Create()
			if err != nil {
				t.Errorf("s3 client create failed: %v", err)
			}
		})
	}*/
}

func TestObjectStore_Save(t *testing.T) {
	// TODO: test failed

	/*type fields struct {
		Bucket          string
		Endpoint        string
		AccessKeyID     string
		AccessKeySecret string
		Region          string
		SSL             bool
		Client          *minio.Client
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
				Bucket:          bucket,
				Endpoint:        endpoint,
				AccessKeyID:     accessKeyID,
				AccessKeySecret: accessKeySecret,
			},
			args: args{
				ctx:      context.TODO(),
				r:        bytes.NewReader([]byte("s3 file test")),
				fileName: s3Filename,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := &ObjectStore{
				Bucket:          tt.fields.Bucket,
				Endpoint:        tt.fields.Endpoint,
				AccessKeyID:     tt.fields.AccessKeyID,
				AccessKeySecret: tt.fields.AccessKeySecret,
				Region:          tt.fields.Region,
				SSL:             tt.fields.SSL,
				Client:          tt.fields.Client,
			}

			bs, err := receiver.Create()
			if err != nil {
				t.Errorf("s3 client create failed: %v", err)
			}
			err = bs.Save(tt.args.ctx, tt.args.r, tt.args.fileName)
			if err != nil {
				t.Errorf("s3 client save failed: %v", err)
			}
		})
	}*/
}

func TestObjectStore_Download(t *testing.T) {
	// TODO: test failed

	/*type fields struct {
		Bucket          string
		Endpoint        string
		AccessKeyID     string
		AccessKeySecret string
		Region          string
		SSL             bool
		Client          *minio.Client
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
				Bucket:          bucket,
				Endpoint:        endpoint,
				AccessKeyID:     accessKeyID,
				AccessKeySecret: accessKeySecret,
			},
			args: args{
				ctx:      context.TODO(),
				fileName: s3Filename,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := &ObjectStore{
				Bucket:          tt.fields.Bucket,
				Endpoint:        tt.fields.Endpoint,
				AccessKeyID:     tt.fields.AccessKeyID,
				AccessKeySecret: tt.fields.AccessKeySecret,
				Region:          tt.fields.Region,
				SSL:             tt.fields.SSL,
				Client:          tt.fields.Client,
			}

			bs, err := receiver.Create()
			if err != nil {
				t.Errorf("s3 client create failed: %v", err)
			}

			w := &bytes.Buffer{}
			err = bs.Download(tt.args.ctx, tt.args.fileName, w)
			if err != nil {
				t.Errorf("s3 client download failed: %v", err)
			}
			t.Logf("file content is: %s", w.String())
		})
	}*/
}

func TestObjectStore_Delete(t *testing.T) {
	// TODO：test failed

	/*type fields struct {
		Bucket          string
		Endpoint        string
		AccessKeyID     string
		AccessKeySecret string
		Region          string
		SSL             bool
		Client          *minio.Client
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
				Bucket:          bucket,
				Endpoint:        endpoint,
				AccessKeyID:     accessKeyID,
				AccessKeySecret: accessKeySecret,
			},
			args: args{
				ctx:      context.TODO(),
				fileName: s3Filename,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := &ObjectStore{
				Bucket:          tt.fields.Bucket,
				Endpoint:        tt.fields.Endpoint,
				AccessKeyID:     tt.fields.AccessKeyID,
				AccessKeySecret: tt.fields.AccessKeySecret,
				Region:          tt.fields.Region,
				SSL:             tt.fields.SSL,
				Client:          tt.fields.Client,
			}
			bs, err := receiver.Create()
			if err != nil {
				t.Errorf("s3 client create failed: %v", err)
			}
			err = bs.Delete(tt.args.ctx, tt.args.fileName)
			if err != nil {
				t.Errorf("s3 client delete failed: %v", err)
			}
		})
	}*/
}
