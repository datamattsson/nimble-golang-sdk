// Copyright 2020 Hewlett Packard Enterprise Development LP

package client

import (
	"fmt"
	"github.com/hpe-storage/common-host-libs/jsonutil"
	"github.com/hpe-storage/nimble-golang-sdk/pkg/client/v1/nimbleos"
	"github.com/hpe-storage/nimble-golang-sdk/pkg/param"
	"reflect"
)

// Key Manager stores encryption keys for the array volumes / dedupe domains.
const (
	keyManagerPath = "key_managers"
)

// KeyManagerObjectSet
type KeyManagerObjectSet struct {
	Client *GroupMgmtClient
}

// CreateObject creates a new KeyManager object
func (objectSet *KeyManagerObjectSet) CreateObject(payload *nimbleos.KeyManager) (*nimbleos.KeyManager, error) {
	resp, err := objectSet.Client.Post(keyManagerPath, payload, &nimbleos.KeyManager{})
	if err != nil {
		return nil, err
	}

	return resp.(*nimbleos.KeyManager), err
}

// UpdateObject Modify existing KeyManager object
func (objectSet *KeyManagerObjectSet) UpdateObject(id string, payload *nimbleos.KeyManager) (*nimbleos.KeyManager, error) {
	resp, err := objectSet.Client.Put(keyManagerPath, id, payload, &nimbleos.KeyManager{})
	if err != nil {
		return nil, err
	}

	return resp.(*nimbleos.KeyManager), err
}

// DeleteObject deletes the KeyManager object with the specified ID
func (objectSet *KeyManagerObjectSet) DeleteObject(id string) error {
	return fmt.Errorf("Unsupported operation 'delete' on KeyManager")
}

// GetObject returns a KeyManager object with the given ID
func (objectSet *KeyManagerObjectSet) GetObject(id string) (*nimbleos.KeyManager, error) {
	resp, err := objectSet.Client.Get(keyManagerPath, id, nimbleos.KeyManager{})
	if err != nil {
		return nil, err
	}

	// null check
	if resp == nil {
		return nil, nil
	}
	return resp.(*nimbleos.KeyManager), err
}

// GetObjectList returns the list of KeyManager objects
func (objectSet *KeyManagerObjectSet) GetObjectList() ([]*nimbleos.KeyManager, error) {
	resp, err := objectSet.Client.List(keyManagerPath)
	if err != nil {
		return nil, err
	}
	return buildKeyManagerObjectSet(resp), err
}

// GetObjectListFromParams returns the list of KeyManager objects using the given params query info
func (objectSet *KeyManagerObjectSet) GetObjectListFromParams(params *param.GetParams) ([]*nimbleos.KeyManager, error) {
	keyManagerObjectSetResp, err := objectSet.Client.ListFromParams(keyManagerPath, params)
	if err != nil {
		return nil, err
	}
	return buildKeyManagerObjectSet(keyManagerObjectSetResp), err
}

// generated function to build the appropriate response types
func buildKeyManagerObjectSet(response interface{}) []*nimbleos.KeyManager {
	values := reflect.ValueOf(response)
	results := make([]*nimbleos.KeyManager, values.Len())

	for i := 0; i < values.Len(); i++ {
		value := &nimbleos.KeyManager{}
		jsonutil.Decode(values.Index(i).Interface(), value)
		results[i] = value
	}

	return results
}
