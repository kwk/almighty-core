package search_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/search"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunTwoQueries(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchWithTwoQueriesTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchWithTwoQueriesTest struct {
	gormtestsupport.DBTestSuite
	searchRepo *search.GormSearchRepository
}

func (s *searchWithTwoQueriesTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.searchRepo = search.NewGormSearchRepository(s.DB)
}

func (s *searchWithTwoQueriesTest) TestQ() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(),
		tf.Spaces(1, tf.SetSpaceNames("MyFirstProject")),
		tf.WorkItemTypes(5, tf.SetWorkItemTypeNames("E", "F", "U", "T", "B")),
		tf.Iterations(2, tf.SetIterationNames("MyFirstProject", "Iteration 1")),
		tf.WorkItems(10,
			tf.SetWorkItemField(workitem.SystemTitle,
				"E1", "E2", "E3",
				"F1", "F2", "F3",
				"U1", "U2", "U3",
				"T1", "T2", "T3"),
			tf.SetWorkItemTypeByName(
				"E", "E", "E",
				"F", "F", "F",
				"U", "U", "U",
				"T", "T", "T"),
			tf.SetWorkItemIterationsByName("Iteration 1"),
		),
		tf.WorkItemLinks(8,
			tf.LinkByWorkItemTtitle(
				tf.SourceTargetNamePairs{"E1", "F1", nil},
				tf.SourceTargetNamePairs{"F1", "U1", nil},
				tf.SourceTargetNamePairs{"U1", "T1", nil},
				tf.SourceTargetNamePairs{"U1", "T2", nil},
				tf.SourceTargetNamePairs{"U1", "B1", nil},
				tf.SourceTargetNamePairs{"U1", "B2", nil},
				tf.SourceTargetNamePairs{"F1", "U2", nil},
				tf.SourceTargetNamePairs{"E1", "F2", nil},
			),
		),
	)
	_ = fxt

	s.T().Run("", func(t *testing.T) {
		// when
		filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
		res, count, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, nil)
		// when
		require.Nil(t, err)
		assert.Equal(t, uint64(2), count)
		assert.Equal(t, 2, len(res))
	})
}

// 		t.Run("with offset", func(t *testing.T) {
// 			// given
// 			fxt := s.getTestFixture()
// 			// when
// 			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
// 			start := 3
// 			res, count, err := s.searchRepo.Filter(context.Background(), filter, nil, &start, nil)
// 			// then
// 			require.Nil(t, err)
// 			assert.Equal(t, uint64(2), count)
// 			assert.Equal(t, 0, len(res))
// 		})

// 		t.Run("with limit", func(t *testing.T) {
// 			// given
// 			fxt := s.getTestFixture()
// 			// when
// 			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
// 			limit := 1
// 			res, count, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, &limit)
// 			// then
// 			require.Nil(s.T(), err)
// 			assert.Equal(t, uint64(2), count)
// 			assert.Equal(t, 1, len(res))
// 		})
// 	})

// 	s.T().Run("with parent-exists filter", func(t *testing.T) {

// 		t.Run("no link created", func(t *testing.T) {
// 			// given
// 			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(3))
// 			// when
// 			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
// 			parentExists := false
// 			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
// 			// then both work items should be returned
// 			require.Nil(t, err)
// 			assert.Equal(t, uint64(3), count)
// 			assert.Equal(t, 3, len(res))
// 		})

// 		t.Run("link created", func(t *testing.T) {
// 			// given
// 			fxt := tf.NewTestFixture(t, s.DB,
// 				tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
// 					// need an explicit 'parent-of' type of link
// 					fxt.WorkItemLinkTypes[idx].ForwardName = link.TypeParentOf
// 					fxt.WorkItemLinkTypes[idx].Topology = link.TopologyTree
// 					return nil
// 				}),
// 				tf.WorkItems(3),
// 				tf.WorkItemLinks(1))
// 			// when
// 			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
// 			parentExists := false
// 			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
// 			// then only parent work item should be returned
// 			require.Nil(t, err)
// 			assert.Equal(t, uint64(2), count)
// 			require.Equal(t, 2, len(res))
// 			// item #0 is parent of #1 and item #2 is not linked to any otjer item
// 			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[2], *fxt.WorkItems[0]))
// 		})

// 		t.Run("link deleted", func(t *testing.T) {
// 			// given
// 			fxt := tf.NewTestFixture(t, s.DB,
// 				tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
// 					// need an explicit 'parent-of' type of link
// 					fxt.WorkItemLinkTypes[idx].ForwardName = link.TypeParentOf
// 					fxt.WorkItemLinkTypes[idx].Topology = link.TopologyTree
// 					return nil
// 				}),
// 				tf.WorkItems(3),
// 				tf.WorkItemLinks(1))
// 			linkRepo := link.NewWorkItemLinkRepository(s.DB)
// 			err := linkRepo.Delete(context.Background(), fxt.WorkItemLinks[0].ID, fxt.Identities[0].ID)
// 			require.Nil(t, err)
// 			// when
// 			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
// 			parentExists := false
// 			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
// 			// then both work items should be returned
// 			require.Nil(t, err)
// 			assert.Equal(t, uint64(3), count)
// 			assert.Equal(t, 3, len(res))
// 		})

// 	})
// }

// // containsAllWorkItems verifies that the `expectedWorkItems` array contains all `actualWorkitems` in the _given order_,
// // by comparing the lengths and each ID,
// func containsAllWorkItems(expectedWorkitems []workitem.WorkItem, actualWorkitems ...workitem.WorkItem) assert.Comparison {
// 	return func() bool {
// 		if len(expectedWorkitems) != len(actualWorkitems) {
// 			return false
// 		}
// 		for i, expectedWorkitem := range expectedWorkitems {
// 			if !uuid.Equal(expectedWorkitem.ID, actualWorkitems[i].ID) {
// 				return false
// 			}
// 		}
// 		return true
// 	}
// }
