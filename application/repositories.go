package application

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"golang.org/x/net/context"
)

// WorkItemRepository encapsulates storage & retrieval of work items
type WorkItemRepository interface {
	Load(ctx context.Context, ID string) (*app.WorkItem, error)
	Save(ctx context.Context, wi app.WorkItem) (*app.WorkItem, error)
	Delete(ctx context.Context, ID string) error
	Create(ctx context.Context, typeID string, fields map[string]interface{}, creator string) (*app.WorkItem, error)
	List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.WorkItem, uint64, error)
}

// WorkItemTypeRepository encapsulates storage & retrieval of work item types
type WorkItemTypeRepository interface {
	Load(ctx context.Context, name string) (*app.WorkItemType, error)
	Create(ctx context.Context, extendedTypeID *string, name string, fields map[string]app.FieldDefinition) (*app.WorkItemType, error)
	List(ctx context.Context, start *int, length *int) ([]*app.WorkItemType, error)
}

// TrackerRepository encapsulate storage & retrieval of tracker configuration
type TrackerRepository interface {
	Load(ctx context.Context, ID string) (*app.Tracker, error)
	Save(ctx context.Context, t app.Tracker) (*app.Tracker, error)
	Delete(ctx context.Context, ID string) error
	Create(ctx context.Context, url string, typeID string) (*app.Tracker, error)
	List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.Tracker, error)
}

// TrackerQueryRepository encapsulate storage & retrieval of tracker queries
type TrackerQueryRepository interface {
	Create(ctx context.Context, query string, schedule string, tracker string) (*app.TrackerQuery, error)
	Save(ctx context.Context, tq app.TrackerQuery) (*app.TrackerQuery, error)
	Load(ctx context.Context, ID string) (*app.TrackerQuery, error)
	Delete(ctx context.Context, ID string) error
	List(ctx context.Context) ([]*app.TrackerQuery, error)
}

// SearchRepository encapsulates searching of woritems,users,etc
type SearchRepository interface {
	SearchFullText(ctx context.Context, searchStr string, start *int, length *int) ([]*app.WorkItem, uint64, error)
}

// WorkItemLinkCategoryRepository encapsulates storage & retrieval of work item link categories
type WorkItemLinkCategoryRepository interface {
	Create(ctx context.Context, name *string, description *string) (*app.WorkItemLinkCategory, error)
	Load(ctx context.Context, ID string) (*app.WorkItemLinkCategory, error)
	List(ctx context.Context) (*app.WorkItemLinkCategoryArray, error)
	Delete(ctx context.Context, ID string) error
	Save(ctx context.Context, linkCat app.WorkItemLinkCategory) (*app.WorkItemLinkCategory, error)
}

// WorkItemLinkTypeRepository encapsulates storage & retrieval of work item link types
type WorkItemLinkTypeRepository interface {
	Create(ctx context.Context, name *string, description *string, sourceType *string, targetType *string, forwardName *string, reverseName *string, linkCategory *string) (*app.WorkItemLinkType, error)
	Load(ctx context.Context, ID string) (*app.WorkItemLinkType, error)
	List(ctx context.Context) (*app.WorkItemLinkTypeArray, error)
	Delete(ctx context.Context, ID string) error
	Save(ctx context.Context, linkCat app.WorkItemLinkType) (*app.WorkItemLinkType, error)
}
