package rules

import (
	"github.com/fabric8-services/fabric8-wit/actions/change"
)

// ActionNoOp is a dummy action rule that does nothing and has no sideffects.
type ActionNoOp struct {
}

// make sure the rule is implementing the interface.
var _ Action = ActionNoOp{}

// OnChange implementes Action
func (act ActionNoOp) OnChange(newContext change.Detector, contextChanges change.Set, configuration string, actionChanges change.Set) (change.Detector, change.Set, error) {
	return newContext, nil, nil
}
