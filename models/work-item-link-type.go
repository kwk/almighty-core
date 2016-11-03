package models

import (
	convert "github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	satoriuuid "github.com/satori/go.uuid"
)

// WorkItemLinkType represents the type of a work item link as it is stored in the db
type WorkItemLinkType struct {
	gormsupport.Lifecycle
	// ID
	ID satoriuuid.UUID `gorm:"primary_key"`
	// Name is the unique name of this work item link category.
	Name string
	// Description is an optional description of the work item link category
	Description *string
	// Version for optimistic concurrency control
	Version      int
	SourceType   string
	TargetType   string
	ForwardName  string
	ReverseName  string
	LinkCategory satoriuuid.UUID `gorm:"primary_key"`
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLinkType{}
var _ convert.Equaler = (*WorkItemLinkType)(nil)

// Equal returns true if two WorkItemLinkType objects are equal; otherwise false is returned.
func (self WorkItemLinkType) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkType)
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
	if self.SourceType != other.SourceType {
		return false
	}
	if self.TargetType != other.TargetType {
		return false
	}
	if self.ForwardName != other.ForwardName {
		return false
	}
	if self.ReverseName != other.ReverseName {
		return false
	}
	// TODO add check for link category
	return true
}
