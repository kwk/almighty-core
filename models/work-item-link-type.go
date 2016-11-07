package models

import (
	"log"

	"github.com/almighty/almighty-core/app"
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
	Version int

	SourceTypeName string
	SourceType     WorkItemType `gorm:"ForeignKey:SourceTypeName;AssociationForeignKey:Name"`

	TargetTypeName string
	TargetType     WorkItemType `gorm:"ForeignKey:TargetTypeName;AssociationForeignKey:Name"`

	ForwardName string
	ReverseName string

	LinkCategoryID satoriuuid.UUID
	LinkCategory   WorkItemLinkCategory `gorm:"ForeignKey:LinkCategoryID;AssociationForeignKey:ID"`
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
	if !satoriuuid.Equal(self.ID, other.ID) {
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
	if self.SourceTypeName != other.SourceTypeName {
		return false
	}
	if !self.SourceType.Equal(other.SourceType) {
		return false
	}
	if self.TargetTypeName != other.TargetTypeName {
		return false
	}
	if !self.TargetType.Equal(other.TargetType) {
		return false
	}
	if self.ForwardName != other.ForwardName {
		return false
	}
	if self.ReverseName != other.ReverseName {
		return false
	}
	if !satoriuuid.Equal(self.LinkCategoryID, other.LinkCategoryID) {
		return false
	}
	if !self.LinkCategory.Equal(other.LinkCategory) {
		return false
	}
	return true
}

// CheckValidForCreation returns an error if the work item link type
// cannot be used for the creation of a new work item link type.
func (t *WorkItemLinkType) CheckValidForCreation() error {
	if t.Name == "" {
		return BadParameterError{parameter: "name", value: t.Name}
	}
	if t.SourceTypeName == "" {
		return BadParameterError{parameter: "source_type_name", value: t.SourceTypeName}
	}
	if t.TargetTypeName == "" {
		return BadParameterError{parameter: "target_type_name", value: t.TargetTypeName}
	}
	if t.ForwardName == "" {
		return BadParameterError{parameter: "forward_name", value: t.ForwardName}
	}
	if t.ReverseName == "" {
		return BadParameterError{parameter: "reverse_name", value: t.ReverseName}
	}
	if t.LinkCategoryID == satoriuuid.Nil {
		return BadParameterError{parameter: "link_category_id", value: t.LinkCategoryID}
	}
	return nil
}

// ConvertLinkTypeFromModel converts a work item link type from model to REST representation
func ConvertLinkTypeFromModel(t *WorkItemLinkType) app.WorkItemLinkType {
	var converted = app.WorkItemLinkType{
		Data: &app.WorkItemLinkTypeData{
			Type: workitemlinktypes,
			ID:   t.ID.String(),
			Attributes: &app.WorkItemLinkTypeAttributes{
				Name:        &t.Name,
				Description: t.Description,
				Version:     &t.Version,
				ForwardName: &t.ForwardName,
				ReverseName: &t.ReverseName,
			},
			Relationships: &app.WorkItemLinkTypeRelationships{
				LinkCategory: &app.RelationWorkItemLinkCategory{
					Data: &app.RelationWorkItemLinkCategoryData{
						Type: workitemlinkcategories,
						ID:   t.LinkCategoryID.String(),
					},
				},
				SourceType: &app.RelationWorkItemType{
					Data: &app.RelationWorkItemTypeData{
						Type: workitemtypes,
						ID:   t.SourceTypeName,
					},
				},
				TargetType: &app.RelationWorkItemType{
					Data: &app.RelationWorkItemTypeData{
						Type: workitemtypes,
						ID:   t.TargetTypeName,
					},
				},
			},
		},
	}
	return converted
}

// ConvertLinkTypeToModel converts the incoming app representation of a work item link type to the model layout.
// Values are only overwrriten if they are set in "in", otherwise the values in "out" remain.
// NOTE: Only the LinkCategoryID, SourceTypeName, and TargetTypeName fields will be set.
//       You need to preload the elements after calling this function.
func ConvertLinkTypeToModel(in *app.WorkItemLinkType, out *WorkItemLinkType) error {
	attrs := in.Data.Attributes
	rel := in.Data.Relationships

	id, err := satoriuuid.FromString(in.Data.ID)
	if err != nil {
		log.Printf("Error when converting %s to UUID: %s", in.Data.ID, err.Error())
		// treat as not found: clients don't know it must be a UUID
		return NotFoundError{entity: "work item link type", ID: id.String()}
	}
	out.ID = id

	if in.Data.Type != workitemlinktypes {
		return BadParameterError{parameter: "data.type", value: in.Data.Type}
	}

	if attrs != nil {
		// If the name is not nil, it MUST NOT be empty
		if attrs.Name != nil {
			if *attrs.Name == "" {
				return BadParameterError{parameter: "data.attributes.name", value: *attrs.Name}
			}
			out.Name = *attrs.Name
		}

		if attrs.Description != nil {
			out.Description = attrs.Description
		}

		if attrs.Version != nil {
			out.Version = *attrs.Version
		}

		// If the forwardName is not nil, it MUST NOT be empty
		if attrs.ForwardName != nil {
			if *attrs.ForwardName == "" {
				return BadParameterError{parameter: "data.attributes.forward_name", value: *attrs.ForwardName}
			}
			out.ForwardName = *attrs.ForwardName
		}

		// If the ReverseName is not nil, it MUST NOT be empty
		if attrs.ReverseName != nil {
			if *attrs.ReverseName == "" {
				return BadParameterError{parameter: "data.attributes.reverse_name", value: *attrs.ReverseName}
			}
			out.ReverseName = *attrs.ReverseName
		}
	}

	if rel != nil && rel.LinkCategory != nil && rel.LinkCategory.Data != nil {
		d := rel.LinkCategory.Data
		// If the the link category is not nil, it MUST be "workitemlinkcategories"
		if d.Type != workitemlinkcategories {
			return BadParameterError{parameter: "data.relationships.link_category.data.type", value: d.Type}
		}
		// The the link category MUST NOT be empty
		if d.ID == "" {
			return BadParameterError{parameter: "data.relationships.link_category.data.id", value: d.ID}
		}
		out.LinkCategoryID, err = satoriuuid.FromString(d.ID)
		if err != nil {
			log.Printf("Error when converting %s to UUID: %s", in.Data.ID, err.Error())
			// treat as not found: clients don't know it must be a UUID
			return NotFoundError{entity: "work item link category", ID: d.ID}
		}
	}

	if rel != nil && rel.SourceType != nil && rel.SourceType.Data != nil {
		d := rel.SourceType.Data
		// If the the link type is not nil, it MUST be "workitemlinktypes"
		if d.Type != workitemtypes {
			return BadParameterError{parameter: "data.relationships.source_type.data.type", value: d.Type}
		}
		// The the link type MUST NOT be empty
		if d.ID == "" {
			return BadParameterError{parameter: "data.relationships.source_type.data.id", value: d.ID}
		}
		out.SourceTypeName = d.ID

	}

	if rel != nil && rel.TargetType != nil && rel.TargetType.Data != nil {
		d := rel.TargetType.Data
		// If the the link type is not nil, it MUST be "workitemlinktypes"
		if d.Type != workitemtypes {
			return BadParameterError{parameter: "data.relationships.target_type.data.type", value: d.Type}
		}
		// The the link type MUST NOT be empty
		if d.ID == "" {
			return BadParameterError{parameter: "data.relationships.target_type.data.id", value: d.ID}
		}
		out.TargetTypeName = d.ID
	}

	return nil
}
