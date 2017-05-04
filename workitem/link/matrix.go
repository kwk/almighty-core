package link

import (
	"time"

	convert "github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"

	uuid "github.com/satori/go.uuid"
)

// Matrix stores the allowed work item types for each each link type
type Matrix struct {
	gormsupport.Lifecycle
	// ID
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Version for optimistic concurrency control
	Version      int
	LinkTypeID   uuid.UUID `sql:"type:uuid"`
	SourceTypeID uuid.UUID `sql:"type:uuid"`
	TargetTypeID uuid.UUID `sql:"type:uuid"`
}

// Ensure Matrix implements the Equaler interface
var _ convert.Equaler = Matrix{}
var _ convert.Equaler = (*Matrix)(nil)

// Equal returns true if two Matrix objects are equal; otherwise false is returned.
func (m Matrix) Equal(u convert.Equaler) bool {
	other, ok := u.(Matrix)
	if !ok {
		return false
	}
	if !m.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if !uuid.Equal(m.ID, other.ID) {
		return false
	}
	if m.Version != other.Version {
		return false
	}
	if !uuid.Equal(m.LinkTypeID, other.LinkTypeID) {
		return false
	}
	if !uuid.Equal(m.SourceTypeID, other.SourceTypeID) {
		return false
	}
	if !uuid.Equal(m.TargetTypeID, other.TargetTypeID) {
		return false
	}
	return true
}

// CheckValidForCreation returns an error if the work item link matrix cannot be
// used for the creation of a new work item link matrix.
func (m *Matrix) CheckValidForCreation() error {
	if t.Name == "" {
		return errors.NewBadParameterError("name", t.Name)
	}
	if uuid.Equal(t.SourceTypeID, uuid.Nil) {
		return errors.NewBadParameterError("source_type_name", t.SourceTypeID)
	}
	if uuid.Equal(t.TargetTypeID, uuid.Nil) {
		return errors.NewBadParameterError("target_type_name", t.TargetTypeID)
	}
	if t.ForwardName == "" {
		return errors.NewBadParameterError("forward_name", t.ForwardName)
	}
	if t.ReverseName == "" {
		return errors.NewBadParameterError("reverse_name", t.ReverseName)
	}
	if err := CheckValidTopology(t.Topology); err != nil {
		return errs.WithStack(err)
	}
	if t.LinkCategoryID == uuid.Nil {
		return errors.NewBadParameterError("link_category_id", t.LinkCategoryID)
	}
	if t.SpaceID == uuid.Nil {
		return errors.NewBadParameterError("space_id", t.SpaceID)
	}
	return nil
}

// TableName implements gorm.tabler
func (m Matrix) TableName() string {
	return "work_item_link_matrix"
}

// GetETagData returns the field values to use to generate the ETag
func (m Matrix) GetETagData() []interface{} {
	return []interface{}{m.ID, m.Version}
}

// GetLastModified returns the last modification time
func (m Matrix) GetLastModified() time.Time {
	return m.UpdatedAt
}
