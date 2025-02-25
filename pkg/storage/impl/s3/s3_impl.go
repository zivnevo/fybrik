// Copyright 2023 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"context"
	"strings"

	"emperror.dev/errors"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	fappv1 "fybrik.io/fybrik/manager/apis/app/v1beta1"
	fappv2 "fybrik.io/fybrik/manager/apis/app/v1beta2"
	"fybrik.io/fybrik/pkg/logging"
	"fybrik.io/fybrik/pkg/model/taxonomy"
	"fybrik.io/fybrik/pkg/random"
	"fybrik.io/fybrik/pkg/serde"
	registrator "fybrik.io/fybrik/pkg/storage/registrator"
	agent "fybrik.io/fybrik/pkg/storage/registrator/agent"
	"fybrik.io/fybrik/pkg/utils"
)

const (
	nameHashLength = 10
	endpointKey    = "endpoint"
	bucketKey      = "bucket"
	objectKey      = "object_key"
)

// s3 storage manager implementaton
type S3Impl struct {
	Name taxonomy.ConnectionType
	Log  zerolog.Logger
}

func NewS3Impl() *S3Impl {
	return &S3Impl{Name: "s3", Log: logging.LogInit(logging.CONNECTOR, "StorageManager")}
}

// register the implementation for s3
func init() {
	s3Impl := NewS3Impl()
	if err := registrator.Register(s3Impl); err != nil {
		s3Impl.Log.Error().Err(err)
	}
}

// return the supported connection type
func (impl *S3Impl) GetConnectionType() taxonomy.ConnectionType {
	return impl.Name
}

// allocate storage for s3 - placeholder
func (impl *S3Impl) AllocateStorage(account *fappv2.FybrikStorageAccountSpec, secret *fappv1.SecretRef,
	opts *agent.Options, client kclient.Client) (taxonomy.Connection, error) {
	endpoint, err := agent.GetProperty(account.AdditionalProperties.Items, impl.Name, endpointKey)
	if err != nil {
		return taxonomy.Connection{}, err
	}
	key := types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}
	// Initialize minio client object.
	minioClient, err := NewClient(endpoint, key, client)
	if err != nil {
		return taxonomy.Connection{}, err
	}
	genBucketName := generateBucketName(opts)
	genObjectKey := generarateObjectKey(opts)

	if err = minioClient.MakeBucket(context.Background(), genBucketName, minio.MakeBucketOptions{}); err != nil {
		return taxonomy.Connection{}, errors.Wrapf(err, "could not create a bucket %s", genBucketName)
	}
	connection := taxonomy.Connection{
		Name: impl.Name,
		AdditionalProperties: serde.Properties{
			Items: map[string]interface{}{
				string(impl.Name): map[string]interface{}{
					endpointKey: endpoint,
					bucketKey:   genBucketName,
					objectKey:   genObjectKey,
				},
			},
		},
	}
	return connection, nil
}

// delete s3 storage
func (impl *S3Impl) DeleteStorage(connection *taxonomy.Connection, secret *fappv1.SecretRef,
	opts *agent.Options, client kclient.Client) error {
	endpoint, err := agent.GetProperty(connection.AdditionalProperties.Items, impl.Name, endpointKey)
	if err != nil {
		return err
	}
	bucket, err := agent.GetProperty(connection.AdditionalProperties.Items, impl.Name, bucketKey)
	if err != nil {
		return err
	}
	key := types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}
	// Initialize minio client object.
	minioClient, err := NewClient(endpoint, key, client)
	if err != nil {
		return err
	}
	exists, err := minioClient.BucketExists(context.Background(), bucket)
	if !exists {
		return kclient.IgnoreNotFound(err)
	}
	for object := range minioClient.ListObjects(context.Background(), bucket,
		minio.ListObjectsOptions{Recursive: true}) {
		if err := minioClient.RemoveObject(context.Background(), bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}

	return minioClient.RemoveBucket(context.Background(), bucket)
}

func generateBucketName(opts *agent.Options) string {
	suffix, _ := random.Hex(nameHashLength)
	name := opts.AppDetails.Owner.Name + "-" + opts.AppDetails.Owner.Namespace + suffix
	return utils.K8sConformName(name)
}

func generarateObjectKey(opts *agent.Options) string {
	return opts.DatasetProperties.Name + utils.Hash(opts.AppDetails.UUID, nameHashLength)
}

func NewClient(endpointArg string, secretKey types.NamespacedName, kClient kclient.Client) (*minio.Client, error) {
	prefix := "https://"
	useSSL := strings.HasPrefix(endpointArg, prefix)
	var endpoint string
	if !useSSL {
		prefix = "http://"
	}
	endpoint = strings.TrimPrefix(endpointArg, prefix)
	// Get credentials
	secret := v1.Secret{}
	if err := kClient.Get(context.Background(), secretKey, &secret); err != nil {
		return nil, errors.Wrapf(err, "could not get a secret %s", secretKey.Name)
	}

	accessKey, secretAccessKey := string(secret.Data["access_key"]), string(secret.Data["secret_key"])
	if accessKey == "" || secretAccessKey == "" {
		return nil, errors.Errorf("could not retrieve credentials from the secret %s", secretKey.Name)
	}

	// Initialize minio client object.
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretAccessKey, ""),
		Secure: useSSL,
	})
}
