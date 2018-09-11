package rules

import (
	"context"
	"encoding/json"
	"reflect"

	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/actions/change"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

// ActionFieldSet takes a configuration JSON object that has field names as the
// keys and a value as the argument. It updates the given ChangeDetector and
// sets the Field[key] value to the values given. Note that this only works on
// WorkItems.
// The ActionFieldSet action can be used to set fields of a ChangeDetector.
type ActionFieldSet struct {
	Db     application.DB
	Ctx    context.Context
	UserID *uuid.UUID
}

// make sure the rule is implementing the interface.
var _ Action = ActionFieldSet{}

func (act ActionFieldSet) storeWorkItem(wi *workitem.WorkItem) (*workitem.WorkItem, error) {
	if act.Ctx == nil {
		return nil, errs.New("context is nil")
	}
	if act.Db == nil {
		return nil, errs.New("database is nil")
	}
	if act.UserID == nil {
		return nil, errs.New("userID is nil")
	}
	var storeResultWorkItem *workitem.WorkItem
	err := application.Transactional(act.Db, func(appl application.Application) error {
		var err error
		storeResultWorkItem, _, err = appl.WorkItems().Save(act.Ctx, wi.SpaceID, *wi, *act.UserID)
		if err != nil {
			return errs.Wrapf(err, "error updating work item %s", wi.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return storeResultWorkItem, nil
}

// OnChange implements Action
func (act ActionFieldSet) OnChange(newContext change.Detector, contextChanges change.Set, configuration string, actionChanges change.Set) (change.Detector, change.Set, error) {
	// check if the newContext is a WorkItem, fail otherwise.
	wiContext, ok := newContext.(workitem.WorkItem)
	if !ok {
		return nil, nil, errs.Errorf("given context is not a work item: %+v", reflect.TypeOf(newContext))
	}
	// deserialize the config JSON
	var rawType map[string]interface{}
	err := json.Unmarshal([]byte(configuration), &rawType)
	if err != nil {
		return nil, nil, errs.Wrapf(err, "failed to unmarshall from action configuration to a map: %s", configuration)
	}
	// load WIT
	wit, err := act.Db.WorkItemTypes().Load(act.Ctx, wiContext.Type)
	if err != nil {
		return nil, nil, errs.Wrapf(err, "error loading work item type: %s", wiContext.Type)
	}
	// iterate over the fields given in the configuration
	for k, v := range rawType {
		if wiContext.Fields[k] != v {
			fieldType, ok := wit.Fields[k]
			if !ok {
				return nil, nil, errs.Errorf(`field "%s" not found in work item type "%s" (%s)`, k, wit.Name, wit.ID)
			}
			actionChanges = append(actionChanges, change.Change{
				AttributeName: k,
				NewValue:      v,
				OldValue:      wiContext.Fields[k],
			})
			newValue, err := fieldType.Type.ConvertToModel(v)
			if err != nil {
				return nil, nil, errs.Wrapf(err, `error converting new value to model: %+v (%[1]T) using field "%s" from work item type "%s" (%s)`, v, k, wit.Name, wit.ID)
			}
			wiContext.Fields[k] = newValue
		}
	}
	// store the work item
	actionResultContext, err := act.storeWorkItem(&wiContext)
	if err != nil {
		return nil, nil, errs.Wrap(err, "failed to store work item")
	}
	// iterate over the resulting wi, see if all keys are there. if not, the key
	// was an unknown key.
	for k := range rawType {
		if _, ok := actionResultContext.Fields[k]; !ok {
			return nil, nil, errs.New("field attribute unknown: " + k)
		}
	}
	return *actionResultContext, actionChanges, nil
}
