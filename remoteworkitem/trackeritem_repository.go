package remoteworkitem

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// upload imports the items into database
func upload(db *gorm.DB, tID int, item TrackerItemContent) error {
	remoteID := item.ID
	content := string(item.Content)

	var ti TrackerItem
	if db.Where("remote_item_id = ? AND tracker_id = ?", remoteID, tID).Find(&ti).RecordNotFound() {
		ti = TrackerItem{
			Item:         content,
			RemoteItemID: remoteID,
			TrackerID:    uint64(tID)}
		return db.Create(&ti).Error
	}
	ti.Item = content
	return db.Save(&ti).Error
}

// Map a remote work item into an ALM work item and persist it into the database.
func convert(db *gorm.DB, tID int, item TrackerItemContent, providerType string) (*app.WorkItem, error) {
	remoteID := item.ID
	content := string(item.Content)
	trackerItem := TrackerItem{Item: content, RemoteItemID: remoteID, TrackerID: uint64(tID)}

	// Converting the remote item to a local work item
	remoteTrackerItemConvertFunc, ok := RemoteWorkItemImplRegistry[providerType]
	if !ok {
		return nil, BadParameterError{parameter: providerType, value: providerType}
	}
	remoteTrackerItem, err := remoteTrackerItemConvertFunc(trackerItem)
	if err != nil {
		return nil, InternalError{simpleError{message: fmt.Sprintf(" Error parsing the tracker data: %s", err.Error())}}
	}
	remoteWorkItem, err := MapToLocal(remoteTrackerItem, RemoteWorkItemKeyMaps[providerType])
	if err != nil {
		return nil, ConversionError{simpleError{message: fmt.Sprintf("Error mapping to local work item: %s", err.Error())}}
	}
	workItem, err := lookupIdentities(db, remoteWorkItem, providerType)
	if err != nil {
		return nil, InternalError{simpleError{message: fmt.Sprintf("Error bind assignees: %s", err.Error())}}
	}

	return upsert(db, *workItem)
}

// lookupIdentities looks up creator and assignee remote identities to local identities (already existing or to be created)
func lookupIdentities(db *gorm.DB, remoteWorkItem RemoteWorkItem, providerType string) (*app.WorkItem, error) {
	identityRepository := account.NewIdentityRepository(db)
	workItem := app.WorkItem{
		ID:     remoteWorkItem.ID,
		Type:   remoteWorkItem.Type,
		Fields: make(map[string]interface{}),
	}
	// copy all fields from remoteworkitem into result workitem
	for fieldName, fieldValue := range remoteWorkItem.Fields {
		// creator
		if fieldName == remoteCreatorLogin {
			if fieldValue == nil {
				workItem.Fields[workitem.SystemCreator] = nil
				continue
			}
			creatorLogin := fieldValue.(string)
			creatorProfileURL := remoteWorkItem.Fields[remoteCreatorProfileURL].(string)
			identity, err := identityRepository.Lookup(context.Background(), creatorLogin, creatorProfileURL, providerType)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create identity during lookup")
			}
			// associate the identities to the work item
			workItem.Fields[workitem.SystemCreator] = identity.ID.String()
		} else if fieldName == remoteCreatorProfileURL {
			// ignore here, it is being processed above
		} else
		// assignees
		if fieldName == remoteAssigneeLogins {
			if fieldValue == nil {
				workItem.Fields[workitem.SystemAssignees] = make([]string, 0)
				continue
			}
			var identities []string
			assigneeLogins := fieldValue.([]string)
			assigneeProfileURLs := remoteWorkItem.Fields[remoteAssigneeProfileURLs].([]string)
			for i, assigneeLogin := range assigneeLogins {
				assigneeProfileURL := assigneeProfileURLs[i]
				identity, err := identityRepository.Lookup(context.Background(), assigneeLogin, assigneeProfileURL, providerType)
				if err != nil {
					return nil, err
				}
				identities = append(identities, identity.ID.String())
			}
			// associate the identities to the work item
			workItem.Fields[workitem.SystemAssignees] = identities
		} else if fieldName == remoteAssigneeProfileURLs {
			// ignore here, it is being processed above
		} else {
			// copy other fields
			workItem.Fields[fieldName] = fieldValue
		}
	}
	return &workItem, nil
}

func upsert(db *gorm.DB, workItem app.WorkItem) (*app.WorkItem, error) {
	wir := workitem.NewWorkItemRepository(db)
	// Get the remote item identifier ( which is currently the url ) to check if the work item exists in the database.
	workItemRemoteID := workItem.Fields[workitem.SystemRemoteItemID]
	log.Info(nil, map[string]interface{}{
		"wiID": workItemRemoteID,
	}, "Upsert on workItemRemoteID=%s", workItemRemoteID)
	// Querying the database to fetch the work item (if it exists)
	sqlExpression := criteria.Equals(criteria.Field(workitem.SystemRemoteItemID), criteria.Literal(workItemRemoteID))
	existingWorkItem, err := wir.Fetch(context.Background(), sqlExpression)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var resultWorkItem *app.WorkItem
	if existingWorkItem != nil {
		log.Info(nil, map[string]interface{}{
			"wiID": existingWorkItem.ID,
		}, "Workitem exists, will be updated")
		for key, value := range workItem.Fields {
			existingWorkItem.Fields[key] = value
		}
		resultWorkItem, err = wir.Save(context.Background(), *existingWorkItem)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		log.Info(nil, nil, "Workitem does not exist, will be created")
		c := workItem.Fields[workitem.SystemCreator]
		var creator uuid.UUID
		if c != nil {
			if creator, err = uuid.FromString(c.(string)); err != nil {
				return nil, errors.Wrapf(err, "Failed to convert creator id into a UUID: %s", err.Error())
			}
		}
		resultWorkItem, err = wir.Create(context.Background(), workitem.SystemBug, workItem.Fields, creator)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	log.Info(nil, map[string]interface{}{
		"wiID": workItem.ID,
	}, "Result workitem: %v", resultWorkItem)

	return resultWorkItem, nil

}
