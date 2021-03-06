// Copyright 2020 Hewlett Packard Enterprise Development LP

package nimbleos

// Autosupport - Get status of autosupport.
// Export AutosupportFields for advance operations like search filter etc.
var AutosupportFields *Autosupport

func init() {
	IDfield := "id"
	GroupIdfield := "group_id"
	GroupNamefield := "group_name"

	AutosupportFields = &Autosupport{
		ID:        &IDfield,
		GroupId:   &GroupIdfield,
		GroupName: &GroupNamefield,
	}
}

type Autosupport struct {
	// ID - Identifier of the autosupport.
	ID *string `json:"id,omitempty"`
	// ArrayList - List of arrays in the group with autosupport information.
	ArrayList []*NsArrayAsupDetail `json:"array_list,omitempty"`
	// ArrayCount - Number of arrays in the group.
	ArrayCount *int64 `json:"array_count,omitempty"`
	// GroupId - Identifier for the group.
	GroupId *string `json:"group_id,omitempty"`
	// GroupName - Name of the group.
	GroupName *string `json:"group_name,omitempty"`
}
