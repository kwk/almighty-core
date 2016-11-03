package models

import (
	"log"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

// NewWorkItemLinkCategoryRepository creates a work item link category repository based on gorm
func NewWorkItemLinkCategoryRepository(db *gorm.DB) *GormWorkItemLinkCategoryRepository {
	return &GormWorkItemLinkCategoryRepository{db}
}

// GormWorkItemLinkCategoryRepository implements WorkItemLinkCategoryRepository using gorm
type GormWorkItemLinkCategoryRepository struct {
	db *gorm.DB
}

// Create creates a new work item link category in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Create(ctx context.Context, name *string, description *string) (*app.WorkItemLinkCategory, error) {

	if name == nil || *name == "" {
		return nil, BadParameterError{parameter: "name", value: name}
	}

	created := WorkItemLinkCategory{
		// Omit "lifecycle" and "ID" fields as they will be filled by the DB
		Name:        *name,
		Description: description,
	}

	if err := r.db.Create(&created).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}

	// Convert the created link category entry into a JSONAPI response
	result, err := ConvertLinkCategoryFromModel(&created)
	if err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}

	return &result, nil
}

// Load returns the work item link category for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Load(ctx context.Context, ID string) (*app.WorkItemLinkCategory, error) {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return nil, NotFoundError{entity: "work item link category", ID: ID}
	}
	log.Printf("loading work item link category %s", id.String())
	res := WorkItemLinkCategory{}
	if r.db.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"work item link category", id.String()}
	}

	// Convert the created link category entry into a JSONAPI response
	result, err := ConvertLinkCategoryFromModel(&res)
	if err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	return &result, nil
}

// List returns all work item link categories
// TODO: Handle pagination
func (r *GormWorkItemLinkCategoryRepository) List(ctx context.Context) (*app.WorkItemLinkCategoryArray, error) {

	// We don't have any where clause or paging at the moment.
	var where string
	var parameters []interface{}
	var start *int
	var limit *int

	var rows []WorkItemLinkCategory
	db := r.db.Where(where, parameters...)
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	res := app.WorkItemLinkCategoryArray{}
	res.Data = make([]*app.WorkItemLinkCategory, len(rows))

	for index, value := range rows {
		cat, err := ConvertLinkCategoryFromModel(&value)
		if err != nil {
			return nil, InternalError{simpleError{err.Error()}}
		}
		res.Data[index] = &cat
	}

	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkCategoryArrayMeta{
		TotalCount: len(rows),
	}

	return &res, nil
}

// Delete deletes the work item link category with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Delete(ctx context.Context, ID string) error {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return NotFoundError{entity: "work item link category", ID: ID}
	}

	var cat = WorkItemLinkCategory{
		ID: id,
	}

	tx := r.db

	if err := tx.Delete(cat).Error; err != nil {
		if tx.RecordNotFound() {
			//return JSONAPIErrors{
			//	Errors:
			//	Code:   "404",
			//	Title:  "Work item link category not found",
			//	Detail: fmt.Sprintf("The work item link category with ID %s wasn't found: %s", ID.String(), err),
			//	Source: {
			//		"parameter": "id",
			//	},
			//}
			return NotFoundError{entity: "work item link category", ID: id.String()}
		}
		return InternalError{simpleError{err.Error()}}
	}

	return nil
}

// Save updates the given work item link category in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Save(ctx context.Context, linkCat app.WorkItemLinkCategory) (*app.WorkItemLinkCategory, error) {
	res := WorkItemLinkCategory{}
	id, err := satoriuuid.FromString(linkCat.Data.ID)
	if err != nil {
		log.Printf("Error when converting %s to UUID: %s", linkCat.Data.ID, err.Error())
		// treat as not found: clients don't know it must be a UUID
		return nil, NotFoundError{entity: "work item link category", ID: id.String()}
	}

	if linkCat.Data.Type != "workitemlinkcategories" {
		return nil, BadParameterError{parameter: "data.type", value: linkCat.Data.Type}
	}

	log.Printf("looking for work item link category with id %s", id.String())
	tx := r.db
	if tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{entity: "work item link category", ID: id.String()}
	}
	if linkCat.Data.Attributes.Version == nil || res.Version != *linkCat.Data.Attributes.Version {
		return nil, VersionConflictError{simpleError{"version conflict"}}
	}

	description := ""
	if linkCat.Data.Attributes.Description != nil {
		description = *linkCat.Data.Attributes.Description
	}

	name := ""
	if linkCat.Data.Attributes.Name != nil {
		name = *linkCat.Data.Attributes.Name
	}

	newLinkCat := WorkItemLinkCategory{
		ID:          id,
		Name:        name,
		Description: &description,
		Version:     *linkCat.Data.Attributes.Version + 1,
	}

	if err := tx.Save(&newLinkCat).Error; err != nil {
		log.Print(err.Error())
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("updated work item link category to %v\n", newLinkCat)
	result, err := ConvertLinkCategoryFromModel(&newLinkCat)
	if err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	return &result, nil
}

// ConvertLinkCategoryFromModel converts from model to app representation
func ConvertLinkCategoryFromModel(t *WorkItemLinkCategory) (app.WorkItemLinkCategory, error) {
	var converted = app.WorkItemLinkCategory{
		Data: &app.WorkItemLinkCategoryData{
			Type: "workitemlinkcategories",
			ID:   t.ID.String(),
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &t.Name,
				Description: t.Description,
				Version:     &t.Version,
			},
		},
	}
	return converted, nil
}
