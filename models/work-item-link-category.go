package models

import (
	convert "github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	satoriuuid "github.com/satori/go.uuid"
)

// WorkItemLinkCategory represents the category of a work item link as it is stored in the db
type WorkItemLinkCategory struct {
	gormsupport.Lifecycle
	// ID
	ID satoriuuid.UUID `gorm:"primary_key"`
	// Name is the unique name of this work item link category.
	Name string
	// Description is an optional description of the work item link category
	Description *string
	// Version for optimistic concurrency control
	Version int
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLinkCategory{}
var _ convert.Equaler = (*WorkItemLinkCategory)(nil)

// Equal returns true if two WorkItemLinkCategory objects are equal; otherwise false is returned.
func (self WorkItemLinkCategory) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkCategory)
	if !ok {
		return false
	}
	if !self.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if self.ID != other.ID {
		return false
	}
	if self.Name != other.Name {
		return false
	}
	if self.Version != other.Version {
		return false
	}
	if self.Description != nil && other.Description != nil {
		if *self.Description != *other.Description {
			return false
		}
	} else {
		if self.Description != other.Description {
			return false
		}
	}
	return true
}
