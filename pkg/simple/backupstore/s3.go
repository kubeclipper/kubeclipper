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
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func init() {
	RegisterProvider(&ObjectStore{})
}

var _ BackupStore = (*ObjectStore)(nil)

const (
	defaultBucket = "kc-backups"
)

type ObjectStore struct {
	Bucket          string `json:"bucket" yaml:"bucket"`
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `json:"accessKeyID" yaml:"accessKeyID"`
	AccessKeySecret string `json:"accessKeySecret" yaml:"accessKeySecret"`
	Region          string `json:"region" yaml:"region"`
	SSL             bool   `json:"ssl" yaml:"ssl"`
	Client          *minio.Client
}

func (receiver *ObjectStore) Type() string {
	return S3Storage
}

func (receiver *ObjectStore) Create() (BackupStore, error) {
	var (
		os  ObjectStore
		err error
	)

	os.Client, err = minio.New(receiver.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(receiver.AccessKeyID, receiver.AccessKeySecret, ""),
	})
	if err != nil {
		return nil, err
	}

	os.Bucket = receiver.Bucket
	if os.Bucket == "" {
		os.Bucket = defaultBucket
	}

	// TODO: 10s is too long
	// Check the bucket exists.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	existed, err := os.Client.BucketExists(ctx, os.Bucket)
	if err != nil {
		return nil, err
	}
	if !existed {
		// Create a bucket.
		if err := os.Client.MakeBucket(ctx, os.Bucket,
			minio.MakeBucketOptions{
				Region: os.Region,
			}); err != nil {
			return nil, err
		}
	}

	return &os, nil
}

func (receiver *ObjectStore) Save(ctx context.Context, r io.Reader, fileName string) (err error) {
	defer logProbe(ctx, fmt.Sprintf("save backup to %s/%s", receiver.Bucket, fileName), err)
	// -1: stream size is unknown to us
	_, err = receiver.Client.PutObject(ctx, receiver.Bucket, fileName, r, -1, minio.PutObjectOptions{})
	return err
}

func (receiver *ObjectStore) Delete(ctx context.Context, fileName string) (err error) {
	defer logProbe(ctx, fmt.Sprintf("delete backup from %s/%s", receiver.Bucket, fileName), err)
	// always delete file
	err = receiver.Client.RemoveObject(ctx, receiver.Bucket, fileName, minio.RemoveObjectOptions{
		ForceDelete: true,
	})
	return
}

func (receiver *ObjectStore) Download(ctx context.Context, fileName string, w io.Writer) (err error) {
	defer logProbe(ctx, fmt.Sprintf("download backup from %s/%s", receiver.Bucket, fileName), err)
	obj, err := receiver.Client.GetObject(ctx, receiver.Bucket, fileName, minio.GetObjectOptions{})
	if err != nil {
		return err
	}

	_, err = bufio.NewReader(obj).WriteTo(w)
	return err
}
