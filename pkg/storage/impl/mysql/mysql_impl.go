// Copyright 2023 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package mysql

import (
	"github.com/rs/zerolog"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"fybrik.io/fybrik/pkg/logging"
	"fybrik.io/fybrik/pkg/model/storagemanager"
	"fybrik.io/fybrik/pkg/model/taxonomy"
	"fybrik.io/fybrik/pkg/storage/registrator"
)

// Storage manager implementation for MySQL
type MySQLImpl struct {
	Name taxonomy.ConnectionType
	Log  zerolog.Logger
}

// implementation of AgentInterface for MySQL
func NewMySQLImpl() *MySQLImpl {
	return &MySQLImpl{Name: "mysql", Log: logging.LogInit(logging.CONNECTOR, "StorageManager")}
}

// register the implementation for MySQL
func init() {
	mysqlImpl := NewMySQLImpl()
	if err := registrator.Register(mysqlImpl); err != nil {
		mysqlImpl.Log.Error().Err(err)
	}
}

// return the supported connection type
func (impl *MySQLImpl) GetConnectionType() taxonomy.ConnectionType {
	return impl.Name
}

// storage allocation - placeholder
func (impl *MySQLImpl) AllocateStorage(request *storagemanager.AllocateStorageRequest, client kclient.Client) (taxonomy.Connection, error) {
	return taxonomy.Connection{Name: impl.Name}, nil
}

// storage deletion - placeholder
func (impl *MySQLImpl) DeleteStorage(request *storagemanager.DeleteStorageRequest, client kclient.Client) error {
	return nil
}
