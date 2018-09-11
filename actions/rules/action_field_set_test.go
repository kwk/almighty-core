package rules

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-wit/actions/change"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/jinzhu/copier"
)

func TestSuiteActionFieldSet(t *testing.T) {
	suite.Run(t, &ActionFieldSetSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ActionFieldSetSuite struct {
	gormtestsupport.DBTestSuite
}

func (s *ActionFieldSetSuite) TestOnChange() {
	s.T().Run("side effects", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemBoards(1),
			tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
					fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{
						fxt.WorkItemBoards[0].Columns[0].ID.String(),
						fxt.WorkItemBoards[0].Columns[1].ID.String(),
					}
				}
				return nil
			}),
		)
		var newVersion workitem.WorkItem
		err := copier.Copy(&newVersion, fxt.WorkItems[0])
		require.NoError(t, err)
		newVersion.Fields[workitem.SystemState] = workitem.SystemStateOpen
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		// when
		afterActionWI, convertChanges, err := action.OnChange(newVersion, contextChanges, fmt.Sprintf(`{"%s":"%s"}`, workitem.SystemState, workitem.SystemStateResolved), nil)
		require.NoError(t, err)
		expectedChanges := change.Set{
			change.Change{
				AttributeName: workitem.SystemState,
				OldValue:      workitem.SystemStateOpen,
				NewValue:      workitem.SystemStateResolved,
			},
		}
		require.Equal(t, expectedChanges, convertChanges)
		require.Equal(t, expectedChanges[0].NewValue, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
	})

	s.T().Run("stacking", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemBoards(1),
			tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
					fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{
						fxt.WorkItemBoards[0].Columns[0].ID.String(),
						fxt.WorkItemBoards[0].Columns[1].ID.String(),
					}
				}
				return nil
			}),
		)
		var newVersion workitem.WorkItem
		err := copier.Copy(&newVersion, fxt.WorkItems[0])
		require.NoError(t, err)
		newVersion.Fields[workitem.SystemState] = workitem.SystemStateOpen
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		prevChange := change.Change{AttributeName: "foo", OldValue: "bar", NewValue: 42}
		prevChanges := change.Set{prevChange}
		// Not using constants here intentionally.
		afterActionWI, convertChanges, err := action.OnChange(newVersion, contextChanges, fmt.Sprintf(`{"%s":"%s"}`, workitem.SystemState, workitem.SystemStateResolved), prevChanges)
		require.NoError(t, err)
		require.Equal(t, change.Set{prevChange}, prevChanges, "the previous changes MUST NOT be modified when calling OnChange")
		expectedChanges := change.Set{
			prevChange,
			change.Change{
				AttributeName: workitem.SystemState,
				OldValue:      workitem.SystemStateOpen,
				NewValue:      workitem.SystemStateResolved,
			},
		}
		require.Equal(t, expectedChanges, convertChanges)
		require.Equal(t, expectedChanges[1].NewValue, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
		// doing another change, the convertChange needs to stack.
		afterActionWI, convertChanges, err = action.OnChange(afterActionWI, change.Set{}, fmt.Sprintf(`{"%s":"%s"}`, workitem.SystemState, workitem.SystemStateNew), convertChanges)
		require.NoError(t, err)
		require.Equal(t, change.Set{change.Change{AttributeName: "foo", OldValue: "bar", NewValue: 42}}, prevChanges)
		expectedChanges = change.Set{
			prevChange,
			change.Change{
				AttributeName: workitem.SystemState,
				OldValue:      workitem.SystemStateOpen,
				NewValue:      workitem.SystemStateResolved,
			},
			change.Change{
				AttributeName: workitem.SystemState,
				OldValue:      workitem.SystemStateResolved,
				NewValue:      workitem.SystemStateNew,
			},
		}
		require.Equal(t, expectedChanges, convertChanges)
		require.Equal(t, expectedChanges[2].NewValue, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
	})

	s.T().Run("unknown field", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemBoards(1),
			tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
					fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{
						fxt.WorkItemBoards[0].Columns[0].ID.String(),
						fxt.WorkItemBoards[0].Columns[1].ID.String(),
					}
				}
				return nil
			}),
		)
		var newVersion workitem.WorkItem
		err := copier.Copy(&newVersion, fxt.WorkItems[0])
		require.NoError(t, err)
		newVersion.Fields[workitem.SystemState] = workitem.SystemStateOpen
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		prevChange := change.Change{AttributeName: "foo", OldValue: "bar", NewValue: 42}
		prevChanges := change.Set{prevChange}
		_, _, err = action.OnChange(newVersion, contextChanges, `{"system.notavailable":"updatedState"}`, prevChanges)
		require.Error(t, err)
		require.Equal(t, change.Set{prevChange}, prevChanges, "the previous changes MUST NOT be modified when calling OnChange")
	})

	s.T().Run("non-json configuration", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemBoards(1),
			tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
					fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{
						fxt.WorkItemBoards[0].Columns[0].ID.String(),
						fxt.WorkItemBoards[0].Columns[1].ID.String(),
					}
				}
				return nil
			}),
		)
		var newVersion workitem.WorkItem
		err := copier.Copy(&newVersion, fxt.WorkItems[0])
		require.NoError(t, err)
		newVersion.Fields[workitem.SystemState] = workitem.SystemStateOpen
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		prevChange := change.Change{AttributeName: "foo", OldValue: "bar", NewValue: 42}
		prevChanges := change.Set{prevChange}
		_, _, err = action.OnChange(newVersion, contextChanges, "someNonJSON", prevChanges)
		require.Error(t, err)
		require.Equal(t, change.Set{prevChange}, prevChanges, "the previous changes MUST NOT be modified when calling OnChange")
	})
}
