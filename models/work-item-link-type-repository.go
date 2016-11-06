package models

import (
	"log"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

const (
	workitemlinkcategories = "workitemlinkcategories"
	workitemlinktypes      = "workitemlinktypes"
	workitemlinks          = "workitemlinks"
	workitemtypes          = "workitemtypes"
)

// NewWorkItemLinkTypeRepository creates a work item link type repository based on gorm
func NewWorkItemLinkTypeRepository(db *gorm.DB) *GormWorkItemLinkTypeRepository {
	return &GormWorkItemLinkTypeRepository{db}
}

// GormWorkItemLinkTypeRepository implements WorkItemLinkTypeRepository using gorm
type GormWorkItemLinkTypeRepository struct {
	db *gorm.DB
}

// Create creates a new work item link type in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Create(ctx context.Context, name *string, description *string, sourceType *string, targetType *string, forwardName *string, reverseName *string, linkCategory *string) (*app.WorkItemLinkType, error) {

	if name == nil || *name == "" {
		return nil, BadParameterError{parameter: "name", value: name}
	}
	if sourceType == nil || *sourceType == "" {
		return nil, BadParameterError{parameter: "sourceType", value: sourceType}
	}
	if targetType == nil || *targetType == "" {
		return nil, BadParameterError{parameter: "targetType", value: targetType}
	}
	if forwardName == nil || *forwardName == "" {
		return nil, BadParameterError{parameter: "forwardName", value: forwardName}
	}
	if reverseName == nil || *reverseName == "" {
		return nil, BadParameterError{parameter: "reverseName", value: reverseName}
	}
	if linkCategory == nil || *linkCategory == "" {
		return nil, BadParameterError{parameter: "linkCat", value: linkCategory}
	}
	linkCat, err := satoriuuid.FromString(*linkCategory)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return nil, NotFoundError{entity: "work item link category", ID: linkCat.String()}
	}

	created := WorkItemLinkType{
		// Omit "lifecycle" and "ID" fields as they will be filled by the DB
		Name:         *name,
		Description:  description,
		SourceType:   *sourceType,
		TargetType:   *targetType,
		ForwardName:  *forwardName,
		ReverseName:  *reverseName,
		LinkCategory: linkCat,
	}

	db := r.db.Create(&created)
	if db.Error != nil {
		return nil, InternalError{simpleError{db.Error.Error()}}
	}

	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkTypeFromModel(&created)

	return &result, nil
}

// Load returns the work item link type for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Load(ctx context.Context, ID string) (*app.WorkItemLinkType, error) {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return nil, NotFoundError{entity: "work item link type", ID: ID}
	}
	log.Printf("loading work item link type %s", id.String())
	res := WorkItemLinkType{}
	db := r.db.First(&res, id)
	if db.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"work item link type", id.String()}
	}

	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkTypeFromModel(&res)
	return &result, nil
}

// List returns all work item link types
// TODO: Handle pagination
func (r *GormWorkItemLinkTypeRepository) List(ctx context.Context) (*app.WorkItemLinkTypeArray, error) {

	// We don't have any where clause or paging at the moment.
	var where string
	var parameters []interface{}
	var start *int
	var limit *int

	var rows []WorkItemLinkType
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
	res := app.WorkItemLinkTypeArray{}
	res.Data = make([]*app.WorkItemLinkType, len(rows))

	for index, value := range rows {
		cat := ConvertLinkTypeFromModel(&value)
		res.Data[index] = &cat
	}

	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkTypeArrayMeta{
		TotalCount: len(rows),
	}

	return &res, nil
}

// Delete deletes the work item link type with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkTypeRepository) Delete(ctx context.Context, ID string) error {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return NotFoundError{entity: "work item link type", ID: ID}
	}

	var cat = WorkItemLinkType{
		ID: id,
	}

	log.Printf("work item link type to delete %v\n", cat)

	db := r.db.Delete(&cat)
	if db.Error != nil {
		return InternalError{simpleError{db.Error.Error()}}
	}

	if db.RowsAffected == 0 {
		return NotFoundError{entity: "work item link type", ID: id.String()}
	}
	return nil
}

// Save updates the given work item link type in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Save(ctx context.Context, lt app.WorkItemLinkType) (*app.WorkItemLinkType, error) {

	res := WorkItemLinkType{}
	db := r.db.First(&res, lt.Data.ID)
	if db.RecordNotFound() {
		log.Printf("work item link type not found, res=%v", res)
		return nil, NotFoundError{entity: "work item link type", ID: lt.Data.ID}
	}

	if lt.Data.Attributes.Version == nil || res.Version != *lt.Data.Attributes.Version {
		return nil, VersionConflictError{simpleError{"version conflict"}}
	}

	if err := ConvertLinkTypeToModel(&lt, &res); err != nil {
		return nil, err
	}

	res.Version = res.Version + 1

	db = db.Save(&res)
	if db.Error != nil {
		log.Print(db.Error.Error())
		return nil, InternalError{simpleError{db.Error.Error()}}
	}
	log.Printf("updated work item link category to %v\n", res)
	result := ConvertLinkTypeFromModel(&res)
	return &result, nil
}
