package models

import (
	"fmt"
	"log"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

const (
	EndpointWorkItemTypes          = "workitemtypes"
	EndpointWorkItems              = "workitems"
	EndpointWorkItemLinkCategories = "workitemlinkcategories"
	EndpointWorkItemLinkTypes      = "workitemlinktypes"
	EndpointWorkItemLinks          = "workitemlinks"
)

// NewWorkItemLinkRepository creates a work item link repository based on gorm
func NewWorkItemLinkRepository(db *gorm.DB) *GormWorkItemLinkRepository {
	return &GormWorkItemLinkRepository{db}
}

// GormWorkItemLinkRepository implements WorkItemLinkRepository using gorm
type GormWorkItemLinkRepository struct {
	db *gorm.DB
}

// Create creates a new work item link in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Create(ctx context.Context, link *WorkItemLink) (*app.WorkItemLink, error) {
	if err := link.CheckValidForCreation(); err != nil {
		return nil, err
	}

	// Fetch the work item link type first in order to check that
	// the given source and target work items match the correct work item types.
	db := r.db.Model(&link.LinkType).Where("id=?", link.LinkTypeID)
	if db.Error != nil {
		return nil, NewInternalError(fmt.Sprintf("Failed to find work item link type: %s", db.Error.Error()))
	}
	if db.RecordNotFound() {
		return nil, NewNotFoundError("work item link type", link.LinkTypeID.String())
	}

	// Fetch the source work item
	db = db.Model(&link.Source).Where("id=?", link.SourceID)
	if db.Error != nil {
		return nil, NewInternalError(fmt.Sprintf("Failed to find source work item: %s", db.Error.Error()))
	}
	if db.RecordNotFound() {
		return nil, NewNotFoundError("work item", link.SourceID)
	}

	// Fetch the target work item
	db = db.Model(&link.Target).Where("id=?", link.TargetID)
	if db.Error != nil {
		return nil, NewInternalError(fmt.Sprintf("Failed to find target work item: %s", db.Error.Error()))
	}
	if db.RecordNotFound() {
		return nil, NewNotFoundError("work item", link.TargetID)
	}

	// Check that the work item types of the source and target match those specified in the link type
	if link.LinkType.SourceTypeName != link.Source.Type {
		return nil, NewBadParameterError("link.source.type", link.Source.Type).Expected(link.Source.Type)
	}
	if link.LinkType.TargetTypeName != link.Target.Type {
		return nil, NewBadParameterError("link.Target.type", link.Target.Type).Expected(link.Target.Type)
	}

	db = db.Create(link)
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}

	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkFromModel(link)
	return &result, nil
}

// Load returns the work item link for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Load(ctx context.Context, ID string) (*app.WorkItemLink, error) {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return nil, NewNotFoundError("work item link", ID)
	}
	log.Printf("loading work item link %s", id.String())
	res := WorkItemLink{}
	db := r.db.Model(&res).Where("id=?", ID).First(&res)
	if db.RecordNotFound() {
		log.Printf("not found work item link, res=%v", res)
		return nil, NewNotFoundError("work item link", id.String())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkFromModel(&res)
	return &result, nil
}

// List returns all work item links
// TODO: Handle pagination
func (r *GormWorkItemLinkRepository) List(ctx context.Context) (*app.WorkItemLinkArray, error) {
	// We don't have any where clause or paging at the moment.
	var where string
	var parameters []interface{}
	var start *int
	var limit *int
	var rows []WorkItemLink
	db := r.db.Where(where, parameters...)
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	db = db.Find(&rows)
	if db.Error != nil {
		return nil, db.Error
	}
	res := app.WorkItemLinkArray{}
	res.Data = make([]*app.WorkItemLink, len(rows))
	for index, value := range rows {
		cat := ConvertLinkFromModel(&value)
		res.Data[index] = &cat
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkArrayMeta{
		TotalCount: len(rows),
	}
	return &res, nil
}

// Delete deletes the work item link with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkRepository) Delete(ctx context.Context, ID string) error {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return NewNotFoundError("work item link", ID)
	}
	var cat = WorkItemLink{
		ID: id,
	}
	log.Printf("work item link to delete %v\n", cat)
	db := r.db.Delete(&cat)
	if db.Error != nil {
		return NewInternalError(db.Error.Error())
	}
	if db.RowsAffected == 0 {
		return NewNotFoundError("work item link", id.String())
	}
	return nil
}

// Save updates the given work item link in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Save(ctx context.Context, lt app.WorkItemLink) (*app.WorkItemLink, error) {
	res := WorkItemLink{}
	if lt.Data.ID == nil {
		return nil, NotFoundError{entity: "work item link", ID: "nil"}
	}
	db := r.db.Model(&res).Where("id=?", *lt.Data.ID).First(&res)
	if db.RecordNotFound() {
		log.Printf("work item link not found, res=%v", res)
		return nil, NewNotFoundError("work item link", *lt.Data.ID)
	}
	if lt.Data.Attributes.Version == nil || res.Version != *lt.Data.Attributes.Version {
		return nil, NewVersionConflictError("version conflict")
	}
	if err := ConvertLinkToModel(&lt, &res); err != nil {
		return nil, err
	}
	res.Version = res.Version + 1
	db = db.Save(&res)
	if db.Error != nil {
		log.Print(db.Error.Error())
		return nil, NewInternalError(db.Error.Error())
	}
	log.Printf("updated work item link to %v\n", res)
	result := ConvertLinkFromModel(&res)
	return &result, nil
}
