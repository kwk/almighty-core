package controller_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSearchController(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchControllerTestSuite struct {
	gormtestsupport.DBTestSuite
	db                             *gormapplication.GormDB
	svc                            *goa.Service
	testIdentity                   account.Identity
	wiRepo                         *workitem.GormWorkItemRepository
	controller                     *SearchController
	spaceBlackBoxTestConfiguration *config.Registry
	testDir                        string
}

func (s *searchControllerTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.testDir = filepath.Join("test-files", "search")
	s.db = gormapplication.NewGormDB(s.DB)
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "searchControllerTestSuite user", "test provider")
	require.NoError(s.T(), err)
	s.testIdentity = *testIdentity

	s.wiRepo = workitem.NewWorkItemRepository(s.DB)
	spaceBlackBoxTestConfiguration, err := config.Get()
	require.NoError(s.T(), err)
	s.spaceBlackBoxTestConfiguration = spaceBlackBoxTestConfiguration
	s.svc = testsupport.ServiceAsUser("WorkItemComment-Service", s.testIdentity)
	s.controller = NewSearchController(s.svc, gormapplication.NewGormDB(s.DB), spaceBlackBoxTestConfiguration)
}

func (s *searchControllerTestSuite) TestSearchWorkItems() {
	// given
	q := "specialwordforsearch"
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = q
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), q, r.Attributes[workitem.SystemTitle])
}

func (s *searchControllerTestSuite) TestSearchPagination() {
	// given
	q := "specialwordforsearch2"
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = q
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	svc := goa.New("TestSearchPagination")
	svc.Context = goa.NewContext(context.Background(), nil, &http.Request{URL: &url.URL{Scheme: "https", Host: "foo.bar.com"}}, nil)
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), svc.Context, svc, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	// defaults in paging.go is 'pageSizeDefault = 20'
	assert.Equal(s.T(), "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.First)
	assert.Equal(s.T(), "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.Last)
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), q, r.Attributes[workitem.SystemTitle])
}

func (s *searchControllerTestSuite) TestSearchWithEmptyValue() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch"
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	q := ""
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, jerrs := test.ShowSearchBadRequest(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotNil(s.T(), jerrs)
	require.Len(s.T(), jerrs.Errors, 1)
	require.NotNil(s.T(), jerrs.Errors[0].ID)
}

func (s *searchControllerTestSuite) TestSearchWithDomainPortCombination() {
	description := "http://localhost:8080/detail/154687364529310 is related issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch_new"
		wi.Fields[workitem.SystemDescription] = expectedDescription
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	q := `"http://localhost:8080/detail/154687364529310"`
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchControllerTestSuite) TestSearchURLWithoutPort() {
	description := "This issue is related to http://localhost/detail/876394"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch_without_port"
		wi.Fields[workitem.SystemDescription] = expectedDescription
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	q := `"http://localhost/detail/876394"`
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchControllerTestSuite) TestUnregisteredURLWithPort() {
	description := "Related to http://some-other-domain:8080/different-path/154687364529310/ok issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch_new"
		wi.Fields[workitem.SystemDescription] = expectedDescription
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	q := `http://some-other-domain:8080/different-path/`
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchControllerTestSuite) TestUnwantedCharactersRelatedToSearchLogic() {
	expectedDescription := rendering.NewMarkupContentFromLegacy("Related to http://example-domain:8080/different-path/ok issue")
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch_new"
		wi.Fields[workitem.SystemDescription] = expectedDescription
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	// add url: in the query, that is not expected by the code hence need to make sure it gives expected result.
	q := `http://url:some-random-other-domain:8080/different-path/`
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotNil(s.T(), sr.Data)
	assert.Empty(s.T(), sr.Data)
}

func (s *searchControllerTestSuite) getWICreatePayload() *app.CreateWorkitemsPayload {
	spaceID := space.SystemSpace
	spaceRelatedURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(spaceID.String()))
	witRelatedURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.WorkitemtypeHref(workitem.SystemTask.String()))
	c := app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: APIStringTypeWorkItemType,
						ID:   workitem.SystemTask,
					},
					Links: &app.GenericLinks{
						Self:    &witRelatedURL,
						Related: &witRelatedURL,
					},
				},
				Space: app.NewSpaceRelation(spaceID, spaceRelatedURL),
			},
		},
	}
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	return &c
}

func getServiceAsUser(testIdentity account.Identity) *goa.Service {
	return testsupport.ServiceAsUser("TestSearch-Service", testIdentity)
}

// searchByURL copies much of the codebase from search_testing.go->ShowSearchOK
// and customises the values to add custom Host in the call.
func (s *searchControllerTestSuite) searchByURL(customHost, queryString string) *app.SearchWorkItemList {
	var resp interface{}
	var respSetter goatest.ResponseSetterFunc = func(r interface{}) { resp = r }
	newEncoder := func(io.Writer) goa.Encoder { return respSetter }
	s.svc.Encoder = goa.NewHTTPEncoder()
	s.svc.Encoder.Register(newEncoder, "*/*")
	rw := httptest.NewRecorder()
	query := url.Values{}
	u := &url.URL{
		Path:     fmt.Sprintf("/api/search"),
		RawQuery: query.Encode(),
		Host:     customHost,
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(s.T(), err)
	prms := url.Values{}
	prms["q"] = []string{queryString} // any value will do
	goaCtx := goa.NewContext(goa.WithAction(s.svc.Context, "SearchTest"), rw, req, prms)
	showCtx, err := app.NewShowSearchContext(goaCtx, req, s.svc)
	require.NoError(s.T(), err)
	// Perform action
	err = s.controller.Show(showCtx)
	// Validate response
	require.NoError(s.T(), err)
	require.Equal(s.T(), 200, rw.Code)
	mt, ok := resp.(*app.SearchWorkItemList)
	require.True(s.T(), ok)
	return mt
}

// verifySearchByKnownURLs performs actual tests on search result and knwonURL map
func (s *searchControllerTestSuite) verifySearchByKnownURLs(wi *app.WorkItemSingle, host, searchQuery string) {
	result := s.searchByURL(host, searchQuery)
	assert.NotEmpty(s.T(), result.Data)
	assert.Equal(s.T(), *wi.Data.ID, *result.Data[0].ID)

	known := search.GetAllRegisteredURLs()
	require.NotNil(s.T(), known)
	assert.NotEmpty(s.T(), known)
	assert.Contains(s.T(), known[search.HostRegistrationKeyForListWI].URLRegex, host)
	assert.Contains(s.T(), known[search.HostRegistrationKeyForBoardWI].URLRegex, host)
}

// TestAutoRegisterHostURL checks if client's host is neatly registered as a KnwonURL or not
// Uses helper functions verifySearchByKnownURLs, searchByURL, getWICreatePayload
func (s *searchControllerTestSuite) TestAutoRegisterHostURL() {
	wiCtrl := NewWorkitemsController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	// create a WI, search by `list view URL` of newly created item
	//fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	newWI := s.getWICreatePayload()
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, wiCtrl, space.SystemSpace, newWI)
	require.NotNil(s.T(), wi)
	customHost := "own.domain.one"
	queryString := fmt.Sprintf("http://%s/work-item/list/detail/%d", customHost, wi.Data.Attributes[workitem.SystemNumber])
	s.verifySearchByKnownURLs(wi, customHost, queryString)

	// Search by `board view URL` of newly created item
	customHost2 := "own.domain.two"
	queryString2 := fmt.Sprintf("http://%s/work-item/board/detail/%d", customHost2, wi.Data.Attributes[workitem.SystemNumber])
	s.verifySearchByKnownURLs(wi, customHost2, queryString2)
}

func (s *searchControllerTestSuite) TestSearchWorkItemsSpaceContext() {
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Identities(1, tf.SetIdentityUsernames("pranav")),
		tf.Spaces(2),
		tf.WorkItems(3+5, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("pranav").ID.String()
			wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
			if idx < 3 {
				wi.SpaceID = fxt.Spaces[0].ID
				wi.Fields[workitem.SystemTitle] = testsupport.CreateRandomValidTestName("shutter_island common_word random - ")
			} else {
				wi.SpaceID = fxt.Spaces[1].ID
				wi.Fields[workitem.SystemTitle] = testsupport.CreateRandomValidTestName("inception common_word random - ")
			}
			return nil
		}),
	)

	// when
	q := "common_word"
	space1IDStr := fxt.Spaces[0].ID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &space1IDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 3)
	for _, item := range sr.Data {
		// make sure that retrived items are from space 1 only
		assert.Contains(s.T(), item.Attributes[workitem.SystemTitle], "shutter_island common_word")
	}
	space2IDStr := fxt.Spaces[1].ID.String()
	_, sr = test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &space2IDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 5)
	for _, item := range sr.Data {
		// make sure that retrived items are from space 2 only
		assert.Contains(s.T(), item.Attributes[workitem.SystemTitle], "inception common_word")
	}

	// when searched without spaceID then it should get all related WI
	_, sr = test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, nil)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 8)
}

func (s *searchControllerTestSuite) TestSearchWorkItemsWithoutSpaceContext() {
	// given 2 spaces with 10 workitems in the first and 5 in the second space
	// random title used in work items
	searchByMe := uuid.NewV4().String()
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Spaces(2),
		tf.WorkItems(10+5, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
			if idx < 10 {
				wi.SpaceID = fxt.Spaces[0].ID
				wi.Fields[workitem.SystemTitle] = testsupport.CreateRandomValidTestName(searchByMe + " common_word random - ")
			} else {
				wi.SpaceID = fxt.Spaces[1].ID
				wi.Fields[workitem.SystemTitle] = testsupport.CreateRandomValidTestName(searchByMe + " common_word random - ")
			}
			return nil
		}),
	)

	q := searchByMe
	// when search without space context
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, nil)
	// then
	require.NotEmpty(s.T(), sr.Data)
	toBeFound := id.Map{}
	for _, wi := range fxt.WorkItems {
		toBeFound[wi.ID] = struct{}{}
	}
	for _, wi := range sr.Data {
		_, ok := toBeFound[*wi.ID]
		assert.True(s.T(), ok, "found unexpected work item: %s", wi.Attributes[workitem.SystemTitle])
		delete(toBeFound, *wi.ID)
	}
	require.Empty(s.T(), toBeFound, "failed to find these work items: %s", toBeFound)
}

func (s *searchControllerTestSuite) TestSearchFilter() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "specialwordforsearch"
			return nil
		}),
	)
	// when
	filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.WorkItems[0].SpaceID)
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), "specialwordforsearch", r.Attributes[workitem.SystemTitle])
}

func (s *searchControllerTestSuite) TestSearchByWorkItemTypeGroup() {
	s.T().Run(http.StatusText(http.StatusOK), func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.CreateWorkItemEnvironment(),
			// TODO(kwk): Decide if these type groups should go to CreateWorkItemEnvironment()
			tf.WorkItemTypeGroups(4, func(fxt *tf.TestFixture, idx int) error {
				witg := fxt.WorkItemTypeGroups[idx]
				switch idx {
				case 0:
					witg.Name = "Scenarios"
					witg.TypeList = []uuid.UUID{
						workitem.SystemScenario,
						workitem.SystemFundamental,
						workitem.SystemPapercuts,
					}
				case 1:
					witg.Name = "Experiences"
					witg.TypeList = []uuid.UUID{
						workitem.SystemExperience,
						workitem.SystemValueProposition,
					}
				case 2:
					witg.Name = "Requirements"
					witg.TypeList = []uuid.UUID{
						workitem.SystemFeature,
						workitem.SystemBug,
					}
				case 3:
					witg.Name = "Execution"
					witg.TypeList = []uuid.UUID{
						workitem.SystemTask,
						workitem.SystemBug,
						workitem.SystemFeature,
					}
				}
				return nil
			}),
		)
		svc := testsupport.ServiceAsUser("TestUpdateWI-Service", *fxt.Identities[0])
		workitemsCtrl := NewWorkitemsController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
		// given work items of different types and in different states
		type testWI struct {
			Title          string
			WorkItemTypeID uuid.UUID
			State          string
			SpaceID        uuid.UUID
		}
		testWIs := []testWI{
			{"closed feature", workitem.SystemFeature, workitem.SystemStateClosed, fxt.Spaces[0].ID},
			{"open feature", workitem.SystemFeature, workitem.SystemStateOpen, fxt.Spaces[0].ID},
			{"closed bug", workitem.SystemBug, workitem.SystemStateClosed, fxt.Spaces[0].ID},
			{"open bug", workitem.SystemBug, workitem.SystemStateOpen, fxt.Spaces[0].ID},
			{"open experience", workitem.SystemExperience, workitem.SystemStateOpen, fxt.Spaces[0].ID},
			{"closed experience", workitem.SystemExperience, workitem.SystemStateClosed, fxt.Spaces[0].ID},
			{"open task", workitem.SystemTask, workitem.SystemStateOpen, fxt.Spaces[0].ID},
			{"closed task", workitem.SystemTask, workitem.SystemStateClosed, fxt.Spaces[0].ID},
			{"open scenario", workitem.SystemScenario, workitem.SystemStateOpen, fxt.Spaces[0].ID},
			{"closed scenario", workitem.SystemScenario, workitem.SystemStateClosed, fxt.Spaces[0].ID},
			{"open fundamental", workitem.SystemFundamental, workitem.SystemStateOpen, fxt.Spaces[0].ID},
			{"closed fundamental", workitem.SystemFundamental, workitem.SystemStateClosed, fxt.Spaces[0].ID},
		}
		for _, wi := range testWIs {
			payload := minimumRequiredCreateWithTypeAndSpace(wi.WorkItemTypeID, wi.SpaceID)
			payload.Data.Attributes[workitem.SystemTitle] = wi.Title
			payload.Data.Attributes[workitem.SystemState] = wi.State
			_, _ = test.CreateWorkitemsCreated(t, svc.Context, svc, workitemsCtrl, wi.SpaceID, &payload)
		}

		// helper function that checks if the given to be found work item titles
		// exist in the result list that originate from a search query.
		checkToBeFound := func(t *testing.T, toBeFound map[string]struct{}, results []*app.WorkItem) {
			require.Len(t, results, len(toBeFound))
			for _, wi := range results {
				title, ok := wi.Attributes[workitem.SystemTitle].(string)
				require.True(t, ok)
				_, ok = toBeFound[title]
				if ok {
					delete(toBeFound, title)
				}
			}
			require.Empty(t, toBeFound, "not all work items could be found: %+v", toBeFound)
		}

		// when
		t.Run("Scenarios", func(t *testing.T) {
			// given
			filter := fmt.Sprintf(`
			{"$AND": [
				{"`+search.TypeGroupName+`": "Scenarios"},
				{"space": "%s"}
			]}`, fxt.Spaces[0].ID)
			// when
			_, sr := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
			// then
			toBeFound := map[string]struct{}{
				"open scenario":      {},
				"closed scenario":    {},
				"open fundamental":   {},
				"closed fundamental": {},
			}
			checkToBeFound(t, toBeFound, sr.Data)
		})
		t.Run("Experiences", func(t *testing.T) {
			// given
			filter := fmt.Sprintf(`
			{"$AND": [
				{"`+search.TypeGroupName+`": "Experiences"},
				{"space": "%s"}
			]}`, fxt.Spaces[0].ID)
			// when
			_, sr := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
			// then
			toBeFound := map[string]struct{}{
				"open experience":   {},
				"closed experience": {},
			}
			checkToBeFound(t, toBeFound, sr.Data)
		})
		t.Run("Requirements", func(t *testing.T) {
			// given
			filter := fmt.Sprintf(`
			{"$AND": [
				{"`+search.TypeGroupName+`": "Requirements"},
				{"space": "%s"}
			]}`, fxt.Spaces[0].ID)
			// when
			_, sr := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
			// then
			toBeFound := map[string]struct{}{
				"open feature":   {},
				"closed feature": {},
				"open bug":       {},
				"closed bug":     {},
			}
			checkToBeFound(t, toBeFound, sr.Data)
		})
		t.Run("Execution", func(t *testing.T) {
			// given
			filter := fmt.Sprintf(`
			{"$AND": [
				{"`+search.TypeGroupName+`": "Execution"},
				{"space": "%s"}
			]}`, fxt.Spaces[0].ID)
			// when
			_, sr := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
			// then
			toBeFound := map[string]struct{}{
				"open task":      {},
				"closed task":    {},
				"open bug":       {},
				"closed bug":     {},
				"open feature":   {},
				"closed feature": {},
			}
			checkToBeFound(t, toBeFound, sr.Data)
		})
		t.Run("unknown hierarchy", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
			filter := fmt.Sprintf(`
			{"$AND": [
				{"`+search.TypeGroupName+`": "%s"},
				{"space": "%s"}
			]}`, "unknown work item type group", fxt.Spaces[0].ID)
			// when
			_, sr := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
			// then
			require.Empty(t, sr.Data)
		})
	})
}

// It creates 1 space
// creates and adds 2 collaborators in the space
// creates 2 iterations within it
// 8 work items with different states & iterations & assignees & types
// and tests multiple combinations of space, state, iteration, assignee, type
func (s *searchControllerTestSuite) TestSearchQueryScenarioDriven() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Identities(3, tf.SetIdentityUsernames("spaceowner", "alice", "bob")),
		tf.Iterations(2, tf.SetIterationNames("sprint1", "sprint2")),
		tf.Labels(4, tf.SetLabelNames("important", "backend", "ui", "rest")),
		tf.WorkItemTypes(2, tf.SetWorkItemTypeNames("bug", "feature")),
		tf.WorkItems(3+5+1, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			if idx < 3 {
				wi.Fields[workitem.SystemTitle] = "There is a special case about it."
				wi.Fields[workitem.SystemState] = workitem.SystemStateResolved
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("sprint1").ID.String()
				wi.Fields[workitem.SystemLabels] = []string{fxt.LabelByName("important").ID.String(), fxt.LabelByName("backend").ID.String()}
				wi.Fields[workitem.SystemAssignees] = []string{fxt.IdentityByUsername("alice").ID.String()}
				wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("spaceowner").ID.String()
				wi.Type = fxt.WorkItemTypeByName("bug").ID
			} else if idx < 3+5 {
				wi.Fields[workitem.SystemTitle] = "some random title"
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("sprint2").ID.String()
				wi.Fields[workitem.SystemLabels] = []string{fxt.LabelByName("ui").ID.String()}
				wi.Fields[workitem.SystemAssignees] = []string{fxt.IdentityByUsername("bob").ID.String()}
				wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("spaceowner").ID.String()
				wi.Type = fxt.WorkItemTypeByName("feature").ID
			} else {
				wi.Fields[workitem.SystemTitle] = "some other random title"
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("sprint2").ID.String()
				wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("spaceowner").ID.String()
				wi.Type = fxt.WorkItemTypeByName("feature").ID
			}
			return nil
		}),
	)
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()

	s.T().Run("label IN IMPORTANT, UI", func(t *testing.T) {
		// following test does not include any "space" deliberately, hence if there
		// is any work item in the test-DB having state=resolved following count
		// will fail
		filter := fmt.Sprintf(`
				{"label": {"$IN": ["%s", "%s"]}}`,
			fxt.LabelByName("important").ID, fxt.LabelByName("ui").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, result)
		fmt.Println(result.Data)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 8) // 3 important + 5 UI
	})

	s.T().Run("space=ID AND (label=Backend OR iteration=sprint2)", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"label": "%s"},
						{"iteration": "%s"}
					]}
				]}`,
			spaceIDStr, fxt.LabelByName("backend").ID, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) // 3 items with Backend label & 5+1 items with sprint2
	})

	s.T().Run("space=ID AND label=UI", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"label": "%s"}
				]}`,
			spaceIDStr, fxt.LabelByName("ui").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 5) // 5 items having UI label
	})

	s.T().Run("label=UI OR label=Backend", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$OR": [
					{"label":"%s"},
					{"label": "%s"}
				]}`,
			fxt.LabelByName("ui").ID, fxt.LabelByName("backend").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 8)
	})

	s.T().Run("space=ID AND label=REST : expect 0 itmes", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"label": "%s"}
				]}`,
			spaceIDStr, fxt.LabelByName("rest").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		assert.Len(t, result.Data, 0) // no items having REST label
	})

	s.T().Run("space=ID AND label != Backend", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"label": "%s", "negate": true}
				]}`,
			spaceIDStr, fxt.LabelByName("backend").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 5+1) // 6 items are not having Backend label
	})

	s.T().Run("state=resolved AND iteration=sprint1", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"state": "%s"},
					{"iteration": "%s"}
				]}`,
			workitem.SystemStateResolved, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		require.Len(t, result.Data, 3) // resolved items having sprint1 are 3
	})

	s.T().Run("state=resolved AND iteration=sprint1 using EQ", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"state": {"$EQ": "%s"}},
					{"iteration": {"$EQ": "%s"}}
				]}`,
			workitem.SystemStateResolved, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		require.Len(t, result.Data, 3) // resolved items having sprint1 are 3
	})

	s.T().Run("state=resolved AND iteration=sprint2", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"state": "%s"},
					{"iteration": "%s"}
				]}`,
			workitem.SystemStateResolved, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.Len(t, result.Data, 0) // No items having state=resolved && sprint2
	})

	s.T().Run("state=resolved OR iteration=sprint2", func(t *testing.T) {
		// following test does not include any "space" deliberately, hence if there
		// is any work item in the test-DB having state=resolved following count
		// will fail
		filter := fmt.Sprintf(`
				{"$OR": [
					{"state": "%s"},
					{"iteration": "%s"}
				]}`,
			workitem.SystemStateResolved, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) // resolved items + items in sprint2
	})

	s.T().Run("space=spaceID AND title=special with $SUBSTR", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"title": {"$SUBSTR":"%s"}}
				]}`,
			spaceIDStr, "special")
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3)
	})

	s.T().Run("state IN resolved, closed", func(t *testing.T) {
		// following test does not include any "space" deliberately, hence if there
		// is any work item in the test-DB having state=resolved following count
		// will fail
		filter := fmt.Sprintf(`
				{"state": {"$IN": ["%s", "%s"]}}`,
			workitem.SystemStateResolved, workitem.SystemStateClosed)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) // state = resolved or state = closed
	})

	s.T().Run("space=ID AND (state=resolved OR iteration=sprint2)", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"state": "%s"},
						{"iteration": "%s"}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateResolved, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1)
	})

	s.T().Run("space=ID AND (state=resolved OR iteration=sprint2) using EQ", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space": {"$EQ": "%s"}},
					{"$OR": [
						{"state": {"$EQ": "%s"}},
						{"iteration": {"$EQ": "%s"}}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateResolved, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1)
	})

	s.T().Run("space=ID AND (state!=resolved AND iteration=sprint1)", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$AND": [
						{"state": "%s", "negate": true},
						{"iteration": "%s"}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateResolved, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		assert.Len(t, result.Data, 0)
	})

	s.T().Run("space=ID AND (state!=open AND iteration!=fake-iterationID)", func(t *testing.T) {
		fakeIterationID := uuid.NewV4()
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space": {"$EQ": "%s"}},
					{"$AND": [
						{"state": "%s", "negate": true},
						{"iteration": "%s", "negate": true}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateOpen, fakeIterationID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 9) // all items are other than open state & in other thatn fake itr
	})

	s.T().Run("space!=ID AND (state!=open AND iteration!=fake-iterationID)", func(t *testing.T) {
		fakeIterationID := uuid.NewV4()
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space": {"$NE": "%s"}},
					{"$AND": [
						{"state": "%s", "negate": true},
						{"iteration": "%s", "negate": true}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateOpen, fakeIterationID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		assert.Empty(t, result.Data)
	})

	s.T().Run("space=ID AND (state!=open AND iteration!=fake-iterationID) using NE", func(t *testing.T) {
		fakeIterationID := uuid.NewV4()
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$AND": [
						{"state": {"$NE": "%s"}},
						{"iteration": {"$NE": "%s"}}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateOpen, fakeIterationID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 9) // all items are other than open state & in other thatn fake itr
	})

	s.T().Run("space=FakeID AND state=closed", func(t *testing.T) {
		fakeSpaceID1 := uuid.NewV4().String()
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"state": "%s"}
				]}`,
			fakeSpaceID1, workitem.SystemStateOpen)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &fakeSpaceID1)
		assert.Len(t, result.Data, 0) // we have 5 closed items but they are in different space
	})

	s.T().Run("space=spaceID AND state=closed AND assignee=bob", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"assignee":"%s"},
					{"state": "%s"}
				]}`,
			spaceIDStr, fxt.IdentityByUsername("bob").ID, workitem.SystemStateClosed)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 5) // we have 5 closed items assigned to bob
	})

	s.T().Run("space=spaceID AND iteration=sprint1 AND assignee=alice", func(t *testing.T) {
		// Let's see what alice did in sprint1
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"assignee":"%s"},
					{"iteration": "%s"}
				]}`,
			spaceIDStr, fxt.IdentityByUsername("alice").ID, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3) // alice worked on 3 issues in sprint1
	})

	s.T().Run("space=spaceID AND creator=spaceowner", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"creator":"%s"}
				]}`,
			spaceIDStr, fxt.IdentityByUsername("spaceowner").ID.String())
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 9) // we have 9 items created by spaceowner
	})

	s.T().Run("space=spaceID AND state!=closed AND iteration=sprint1 AND assignee=alice", func(t *testing.T) {
		// Let's see non-closed issues alice working on from sprint1
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"assignee":"%s"},
					{"state":"%s", "negate": true},
					{"iteration": "%s"}
				]}`,
			spaceIDStr, fxt.IdentityByUsername("alice").ID, workitem.SystemStateClosed, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3)
	})

	s.T().Run("space=spaceID AND (state=closed or state=resolved)", func(t *testing.T) {
		// get me all closed and resolved work items from my space
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"state":"%s"},
						{"state":"%s"}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateClosed, workitem.SystemStateResolved)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) //resolved + closed
	})

	s.T().Run("space=spaceID AND (type=bug OR type=feature)", func(t *testing.T) {
		// get me all bugs or features in myspace
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"type":"%s"},
						{"type":"%s"}
					]}
				]}`,
			spaceIDStr, fxt.WorkItemTypeByName("bug").ID, fxt.WorkItemTypeByName("feature").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) //bugs + features
	})

	s.T().Run("space=spaceID AND (workitemtype=bug OR workitemtype=feature)", func(t *testing.T) {
		// get me all bugs or features in myspace
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"workitemtype":"%s"},
						{"workitemtype":"%s"}
					]}
				]}`,
			spaceIDStr, fxt.WorkItemTypeByName("bug").ID, fxt.WorkItemTypeByName("feature").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) //bugs + features
	})

	s.T().Run("space=spaceID AND (type=bug AND state=resolved AND (assignee=bob OR assignee=alice))", func(t *testing.T) {
		// get me all Resolved bugs assigned to bob or alice
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$AND": [
						{"$AND": [{"type":"%s"},{"state":"%s"}]},
						{"$OR": [{"assignee":"%s"},{"assignee":"%s"}]}
					]}
				]}`,
			spaceIDStr, fxt.WorkItemTypeByName("bug").ID, workitem.SystemStateResolved, fxt.IdentityByUsername("bob").ID, fxt.IdentityByUsername("alice").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3) //resolved bugs
	})

	s.T().Run("space=spaceID AND (workitemtype=bug AND state=resolved AND (assignee=bob OR assignee=alice))", func(t *testing.T) {
		// get me all Resolved bugs assigned to bob or alice
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$AND": [
						{"$AND": [{"workitemtype":"%s"},{"state":"%s"}]},
						{"$OR": [{"assignee":"%s"},{"assignee":"%s"}]}
					]}
				]}`,
			spaceIDStr, fxt.WorkItemTypeByName("bug").ID, workitem.SystemStateResolved, fxt.IdentityByUsername("bob").ID, fxt.IdentityByUsername("alice").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3) //resolved bugs
	})

	s.T().Run("bad expression missing curly brace", func(t *testing.T) {
		filter := fmt.Sprintf(`{"state": "0fe7b23e-c66e-43a9-ab1b-fbad9924fe7c"`)
		res, jerrs := test.ShowSearchBadRequest(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.NotNil(t, jerrs.Errors[0].ID)
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGolden(t, filepath.Join(s.testDir, "show", "bad_expression_missing_curly_brace.error.golden.json"), jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "bad_expression_missing_curly_brace.headers.golden.json"), res.Header())
	})

	s.T().Run("non existing key", func(t *testing.T) {
		filter := fmt.Sprintf(`{"nonexistingkey": "0fe7b23e-c66e-43a9-ab1b-fbad9924fe7c"}`)
		res, jerrs := test.ShowSearchBadRequest(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.NotNil(t, jerrs.Errors[0].ID)
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGolden(t, filepath.Join(s.testDir, "show", "non_existing_key.error.golden.json"), jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "non_existing_key.headers.golden.json"), res.Header())
	})

	s.T().Run("assignee=null before WI creation", func(t *testing.T) {
		filter := fmt.Sprintf(`
					{"$AND": [
						{"assignee":null}
					]}`,
		)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(s.T(), result)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 1)
	})

	s.T().Run("assignee=null after WI creation (top-level)", func(t *testing.T) {
		filter := fmt.Sprintf(`
					{"assignee":null}`,
		)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 1)
	})

	s.T().Run("assignee=null with negate", func(t *testing.T) {
		filter := fmt.Sprintf(`{"$AND": [{"assignee":null, "negate": true}]}`)
		res, jerrs := test.ShowSearchBadRequest(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.NotNil(t, jerrs.Errors[0].ID)
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGolden(t, filepath.Join(s.testDir, "show", "assignee_null_negate.error.golden.json"), jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "assignee_null_negate.headers.golden.json"), res.Header())
	})
}

func (s *searchControllerTestSuite) TestSearchByJoinedData() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Iterations(2),
		tf.WorkItems(5, func(fxt *tf.TestFixture, idx int) error {
			switch idx {
			case 0, 1, 2, 3:
				fxt.WorkItems[idx].Fields[workitem.SystemIteration] = fxt.Iterations[0].ID.String()
			default:
				fxt.WorkItems[idx].Fields[workitem.SystemIteration] = fxt.Iterations[1].ID.String()
			}
			return nil
		}),
	)
	spaceIDStr := fxt.Spaces[0].ID.String()
	s.T().Run("matching name", func(t *testing.T) {
		// given
		filter := fmt.Sprintf(`{"iteration.name": "%s"}`, fxt.Iterations[0].Name)
		// when
		resWriter, list := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, ptr.String(spaceIDStr))
		// then
		require.NotNil(t, resWriter)
		require.NotNil(t, list)
		assert.Len(t, list.Data, 4)
		toBeFound := id.MapFromSlice(id.Slice{
			fxt.WorkItems[0].ID,
			fxt.WorkItems[1].ID,
			fxt.WorkItems[2].ID,
			fxt.WorkItems[3].ID,
		})
		for _, wi := range list.Data {
			_, ok := toBeFound[*wi.ID]
			require.True(t, ok, "unknown work item found: %s", *wi.ID)
			delete(toBeFound, *wi.ID)
		}
		require.Empty(t, toBeFound, "failed to found all work items: %+s", toBeFound)
	})
}

// TestIncludedParents verifies the Included list of parents
func (s *searchControllerTestSuite) TestIncludedParents() {

	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItems(5, tf.SetWorkItemTitles("A", "B", "C", "D", "E")),
		tf.WorkItemLinksCustom(3, func(fxt *tf.TestFixture, idx int) error {
			l := fxt.WorkItemLinks[idx]
			l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
			switch idx {
			case 0:
				l.SourceID = fxt.WorkItemByTitle("A").ID
				l.TargetID = fxt.WorkItemByTitle("B").ID
			case 1:
				l.SourceID = fxt.WorkItemByTitle("B").ID
				l.TargetID = fxt.WorkItemByTitle("C").ID
			case 2:
				l.SourceID = fxt.WorkItemByTitle("A").ID
				l.TargetID = fxt.WorkItemByTitle("D").ID
			}
			return nil
		}),
	)

	A := fxt.WorkItemByTitle("A").ID
	B := fxt.WorkItemByTitle("B").ID
	C := fxt.WorkItemByTitle("C").ID
	D := fxt.WorkItemByTitle("D").ID
	E := fxt.WorkItemByTitle("E").ID

	spaceIDStr := fxt.Spaces[0].ID.String()

	printCb := func(ID uuid.UUID) string {
		return fmt.Sprintf("%s (%s)", fxt.WorkItemByID(ID).Fields[workitem.SystemTitle].(string), ID)
	}

	s.T().Run("in topology A-B-C and A-D search for", func(t *testing.T) {
		testFunc := func(t *testing.T, searchForTitles []string, expectedData, expectedIncludes id.Slice, treeView bool) {
			matches := id.MapFromSlice(expectedData)
			included := id.MapFromSlice(expectedIncludes)

			// Build title filter and test name
			require.NotEmpty(t, searchForTitles)
			titleFilter := ""
			for _, title := range searchForTitles {
				f := fmt.Sprintf(`{"title":"%[1]s"}`, title)
				if titleFilter == "" {
					titleFilter = f
				} else {
					titleFilter = fmt.Sprintf(`{"$OR": [%[1]s, %[2]s]}`, f, titleFilter)
				}
			}
			filter := fmt.Sprintf(`"$AND": [%[1]s, {"space": "%[2]s"}]`, titleFilter, spaceIDStr)
			testName := strings.Join(searchForTitles, ",")
			if treeView {
				filter = filter + "," + fmt.Sprintf(`"$OPTS":{"%[1]s": true}`, search.OptTreeViewKey)
				testName += " with tree-view=true"
			} else {
				testName += " with tree-view=false"
			}
			filter = "{" + filter + "}"

			t.Run(testName, func(t *testing.T) {
				t.Logf("Running with filter: %s", filter)
				// when
				_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
				// then
				require.NotEmpty(t, result.Data)
				assert.Len(t, result.Data, len(searchForTitles))
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", strings.Replace("in topology A-B-C and A-D search for", " ", "_", -1), strings.Replace(testName+".res.golden.json", " ", "_", -1)), result)

				// test what's included in the "data" portion of the response
				for _, wi := range result.Data {
					_, ok := matches[*wi.ID]
					if wi := fxt.WorkItemByID(*wi.ID); wi != nil {
						t.Logf("found data work item: %s (%s)", wi.Fields[workitem.SystemTitle].(string), wi.ID)
					}
					assert.True(t, ok, "failed to find work item in list of expected \"data\" results: %s (%s)", wi.Attributes[workitem.SystemTitle].(string), wi.ID)
					delete(matches, *wi.ID)
				}
				assert.Empty(t, matches, "failed to find these work items in \"data\" section: %s", matches.ToString(", ", printCb))

				assert.True(t, len(result.Included) >= len(included), "length of included elements (%d) must be at least %d", len(result.Included), len(included))
				for _, ele := range result.Included {
					appWI, ok := ele.(app.WorkItem)
					if ok {
						_, ok = included[*appWI.ID]
						t.Logf("found included work item: %s (%s)", fxt.WorkItemByID(*appWI.ID).Fields[workitem.SystemTitle].(string), *appWI.ID)
						assert.True(t, ok, "failed to find work item in list of expected \"included\" results: %s (%s)", appWI.Attributes[workitem.SystemTitle].(string), *appWI.ID)
						delete(included, *appWI.ID)
					}
				}
				assert.Empty(t, included, "failed to find these work items in \"included\" section: %s", included.ToString(", ", printCb))
				// check order of execution
				if len(result.Included) > 1 {
					var expectedOrder []interface{}
					for i := 0; i < 2; i++ {
						expectedOrder = append(expectedOrder, fxt.WorkItems[i].ID) // expectedOrder = [A, B]
					}
					includedData := make([]app.WorkItem, len(result.Included))
					for i, v := range result.Included {
						includedData[i].ID = v.(app.WorkItem).ID
						assert.Equal(t, expectedOrder[i], *includedData[i].ID)
					}
				}

				if treeView {
					t.Run("check that all non-root work items have a parent relationship", func(t *testing.T) {
						hasParent := func(wi app.WorkItem) {
							// treat the root differently
							if *wi.ID == A || *wi.ID == E {
								return
							}
							title := fxt.WorkItemByID(*wi.ID).Fields[workitem.SystemTitle].(string)
							require.NotNil(t, wi.Relationships, "work item %s is missing relationships", title)
							require.NotNil(t, wi.Relationships.Parent, "work item %s is missing relationships.parent", title)
							require.NotNil(t, wi.Relationships.Parent.Data, "work item %s is missing relationships.parent.data", title)
							parentID := wi.Relationships.Parent.Data.ID
							switch *wi.ID {
							case B:
								require.Equal(t, parentID, A)
							case C:
								require.Equal(t, parentID, B)
							case D:
								require.Equal(t, parentID, A)
							}
						}
						// check data array
						for _, wi := range result.Data {
							hasParent(*wi)
						}
						// check included array
						for _, ele := range result.Included {
							wi, ok := ele.(app.WorkItem)
							if ok {
								hasParent(wi)
							}
						}
					})
				}
			})
		}
		// // Without tree-view query option
		testFunc(t, []string{"A"}, id.Slice{A}, nil, false)
		testFunc(t, []string{"B"}, id.Slice{B}, nil, false)
		testFunc(t, []string{"C"}, id.Slice{C}, nil, false)
		testFunc(t, []string{"D"}, id.Slice{D}, nil, false)
		testFunc(t, []string{"E"}, id.Slice{E}, nil, false)
		// With tree-view query option
		testFunc(t, []string{"A"}, id.Slice{A}, nil, true)
		testFunc(t, []string{"B"}, id.Slice{B}, id.Slice{A}, true)
		testFunc(t, []string{"C"}, id.Slice{C}, id.Slice{B, A}, true)
		testFunc(t, []string{"D"}, id.Slice{D}, id.Slice{A}, true)
		testFunc(t, []string{"E"}, id.Slice{E}, nil, true)
		// search for parent and child elements without tree-view query option
		testFunc(t, []string{"B", "C"}, id.Slice{B, C}, nil, false)
		// search for parent and child elements with tree-view query option
		testFunc(t, []string{"B", "C"}, id.Slice{B, C}, id.Slice{A}, true)
	})
}

func (s *searchControllerTestSuite) TestIncludedChildren() {

	// Suppose we have this topology:
	//
	//   A
	//   |_B
	//     |_C
	//     | |_D
	//     |
	//     |_E
	//
	// We need to make sure that E is included when C is a match AND the
	// tree-view option is "on". If C is a match and trew-view option is "off",
	// then don't include E.
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItems(5, tf.SetWorkItemTitles("A", "B", "C", "D", "E")),
		tf.WorkItemLinksCustom(4,
			tf.BuildLinks(append(tf.LinkChain("A", "B", "C", "D"), tf.L("B", "E"))...),
			func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[idx].LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
				return nil
			},
		),
	)

	A := fxt.WorkItemByTitle("A").ID
	B := fxt.WorkItemByTitle("B").ID
	C := fxt.WorkItemByTitle("C").ID
	E := fxt.WorkItemByTitle("E").ID
	spaceIDStr := fxt.Spaces[0].ID.String()

	testFolder := "in_topology_A-B-C-D_and_B-E_search_for_B_and_C"
	s.T().Run("in topology A-B-C-D and B-E search for", func(t *testing.T) {
		t.Run("B,C with tree-view = true", func(t *testing.T) {
			// when
			filter := fmt.Sprintf(`{"$AND":[{"space":"%[1]s"}, {"$OR": [{"title":"B"}, {"title":"C"}]}], "$OPTS":{"%[2]s": true}}`, spaceIDStr, search.OptTreeViewKey)
			_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
			// then
			require.NotEmpty(t, result.Data)
			// check "data" section
			toBeFound := id.Slice{B, C}.ToMap()
			for _, wi := range result.Data {
				_, ok := toBeFound[*wi.ID]
				require.True(t, ok, "found unexpected work item in data section: %s", wi.Attributes[workitem.SystemTitle].(string))
				delete(toBeFound, *wi.ID)
				// check parent of each work item
				require.NotNil(t, wi.Relationships)
				require.NotNil(t, wi.Relationships.Parent)
				require.NotNil(t, wi.Relationships.Parent.Data)
				if *wi.ID == B {
					require.Equal(t, A, wi.Relationships.Parent.Data.ID)
				}
				if *wi.ID == C {
					require.Equal(t, B, wi.Relationships.Parent.Data.ID)
				}
			}
			require.Empty(t, toBeFound, "failed to find these work items: %s", toBeFound.ToString(", ", func(ID uuid.UUID) string {
				return fxt.WorkItemByID(ID).Fields[workitem.SystemTitle].(string)
			}))
			// check "included" section
			toBeFound = id.Slice{A, E}.ToMap()
			for _, ifObj := range result.Included {
				// technically an included object can be a pointer or an object
				var wi *app.WorkItem
				obj, ok := ifObj.(app.WorkItem)
				if ok {
					wi = &obj
				} else {
					wi, ok = ifObj.(*app.WorkItem)
					if !ok {
						continue
					}
				}
				_, ok = toBeFound[*wi.ID]
				require.True(t, ok, "found unexpected work item in data section: %s", wi.Attributes[workitem.SystemTitle].(string))
				delete(toBeFound, *wi.ID)
				// check parent of each work item
				require.NotNil(t, wi.Relationships)
				if *wi.ID == E {
					require.NotNil(t, wi.Relationships.Parent)
					require.NotNil(t, wi.Relationships.Parent.Data)
					require.Equal(t, B, wi.Relationships.Parent.Data.ID)
				}
				if *wi.ID == A {
					require.Empty(t, wi.Relationships.Parent)
				}
			}
			require.Empty(t, toBeFound, "failed to find these work items: %s", toBeFound.ToString(", ", func(ID uuid.UUID) string {
				return fxt.WorkItemByID(ID).Fields[workitem.SystemTitle].(string)
			}))
			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", testFolder, "include_children.res.payload.golden.json"), result)
		})

		t.Run("B,C with tree-view = false", func(t *testing.T) {
			// when
			filter := fmt.Sprintf(`{"$AND":[{"space":"%[1]s"}, {"$OR": [{"title":"B"}, {"title":"C"}]}], "$OPTS":{"%[2]s": false}}`, spaceIDStr, search.OptTreeViewKey)
			_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
			// then
			require.NotEmpty(t, result.Data)
			require.Empty(t, result.Included)
			// check "data" section
			toBeFound := id.Slice{B, C}.ToMap()
			for _, wi := range result.Data {
				_, ok := toBeFound[*wi.ID]
				require.True(t, ok, "found unexpected work item in data section: %s", wi.Attributes[workitem.SystemTitle].(string))
				delete(toBeFound, *wi.ID)
				// check parent of each work item
				require.NotNil(t, wi.Relationships)
				require.Empty(t, wi.Relationships.Parent)
			}
			require.Empty(t, toBeFound, "failed to find these work items: %s", toBeFound.ToString(", ", func(ID uuid.UUID) string {
				return fxt.WorkItemByID(ID).Fields[workitem.SystemTitle].(string)
			}))
			// check "included" section
			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", testFolder, "do_not_include_children.res.payload.golden.json"), result)
		})
	})
}

func (s *searchControllerTestSuite) TestUpdateWorkItem() {

	s.T().Run("assignees", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.CreateWorkItemEnvironment(),
			tf.WorkItems(2,
				tf.SetWorkItemField(workitem.SystemTitle, "assigned", "unassigned"),
				func(fxt *tf.TestFixture, idx int) error {
					if idx == 0 {
						fxt.WorkItems[idx].Fields[workitem.SystemAssignees] = []string{fxt.Identities[0].ID.String()}
					}
					return nil
				},
			),
		)
		filter := fmt.Sprintf(`{"$AND":[{"space":"%s"},{"assignee":null}]}`, fxt.Spaces[0].ID.String())
		t.Run("filter null", func(t *testing.T) {
			// when
			_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
			// then
			require.Len(t, result.Data, 1)
			require.Equal(t, fxt.WorkItemByTitle("unassigned").ID, *result.Data[0].ID)

			t.Run("assignee should be nil if assignee field is not touched during update", func(t *testing.T) {
				wi := result.Data[0]
				workitemCtrl := NewWorkitemController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)

				wi.Attributes[workitem.SystemTitle] = "Updated Test WI"
				payload2 := app.UpdateWorkitemPayload{Data: wi}
				_, updated := test.UpdateWorkitemOK(t, s.svc.Context, s.svc, workitemCtrl, *wi.ID, &payload2)
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "filter_assignee_null_update_work_item.golden.json"), updated)

				_, result = test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "filter_assignee_null_show_after_update_work_item.golden.json"), updated)
				assert.Nil(s.T(), result.Data[0].Attributes[workitem.SystemAssignees])

			})
		})
	})
	s.T().Run("labels", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Labels(1),
			tf.WorkItems(2,
				tf.SetWorkItemField(workitem.SystemTitle, "labelled", "unlabelled"),
				func(fxt *tf.TestFixture, idx int) error {
					if idx == 0 {
						fxt.WorkItems[idx].Fields[workitem.SystemLabels] = []string{fxt.Labels[0].ID.String()}
					}
					return nil
				},
			),
		)
		filter := fmt.Sprintf(`{"$AND":[{"space":"%s"},{"label":{"$EQ":null}}]}`, fxt.Spaces[0].ID.String())
		t.Run("filter null", func(t *testing.T) {
			// when
			_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
			// then
			require.Len(t, result.Data, 1)
			require.Equal(t, fxt.WorkItemByTitle("unlabelled").ID, *result.Data[0].ID)

			t.Run("assignee should be nil if label field is not touched during update", func(t *testing.T) {
				wi := result.Data[0]
				workitemCtrl := NewWorkitemController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
				wi.Attributes[workitem.SystemTitle] = "Updated Test WI"
				payload2 := app.UpdateWorkitemPayload{Data: wi}
				_, updated := test.UpdateWorkitemOK(t, s.svc.Context, s.svc, workitemCtrl, *wi.ID, &payload2)
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "filter_label_null_update_work_item.golden.json"), updated)

				_, result = test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, nil)
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "filter_label_null_show_after_update_work_item.golden.json"), updated)
				assert.Nil(s.T(), result.Data[0].Attributes[workitem.SystemLabels])
			})
		})
	})
}

func (s *searchControllerTestSuite) TestSearchCodebases() {

	s.T().Run("Single match", func(t *testing.T) {
		// given
		tf.NewTestFixture(s.T(), s.DB,
			tf.Identities(1, tf.SetIdentityUsernames("spaceowner")),
			tf.Codebases(2, func(fxt *tf.TestFixture, idx int) error {
				fxt.Codebases[idx].URL = fmt.Sprintf("http://foo.com/single/%d", idx)
				return nil
			}),
		) // when
		_, codebaseList := test.CodebasesSearchOK(t, nil, nil, s.controller, nil, nil, "http://foo.com/single/0")
		// then
		require.NotNil(t, codebaseList)
		require.NotNil(t, codebaseList.Data)
		require.Len(t, codebaseList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "search_codebase_per_url_single_match.json"), codebaseList)
	})

	s.T().Run("Multi-match", func(t *testing.T) {
		// given
		count := 5
		tf.NewTestFixture(s.T(), s.DB,
			tf.Identities(1, tf.SetIdentityUsernames("spaceowner")),
			tf.Spaces(count),
			tf.Codebases(count, func(fxt *tf.TestFixture, idx int) error {
				fxt.Codebases[idx].URL = fmt.Sprintf("http://foo.com/multi/0") // both codebases have the same URL...
				fxt.Codebases[idx].SpaceID = fxt.Spaces[idx].ID                // ... but they belong to different spaces
				return nil
			}),
		) // when
		_, codebaseList := test.CodebasesSearchOK(t, nil, nil, s.controller, nil, nil, "http://foo.com/multi/0")
		// then
		require.NotNil(t, codebaseList)
		require.NotNil(t, codebaseList.Data)
		require.Len(t, codebaseList.Data, count)
		require.Len(t, codebaseList.Included, count)
		// custom sorting of data to make sure the comparison works as expected
		// sorting codebases in `data` by the ID of their part space
		sort.Sort(SortableCodebasesByID(codebaseList.Data))
		// for included spaces, we must sort the spaces by their ID
		sort.Sort(SortableIncludedSpacesByID(codebaseList.Included))
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "search_codebase_per_url_multi_match.json"), codebaseList)
	})
}

// SortableCodebasesByID a custom type that implement `sort.Interface` for sorting CodeBases by ID
type SortableCodebasesByID []*app.Codebase

func (s SortableCodebasesByID) Len() int {
	return len(s)
}
func (s SortableCodebasesByID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SortableCodebasesByID) Less(i, j int) bool {
	return strings.Compare(*s[i].Relationships.Space.Data.ID, *s[j].Relationships.Space.Data.ID) < 0
}

// SortableIncludedSpacesByID a custom type that implement `sort.Interface` for sorting Spaces by ID
type SortableIncludedSpacesByID []interface{}

func (s SortableIncludedSpacesByID) Len() int {
	return len(s)
}
func (s SortableIncludedSpacesByID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SortableIncludedSpacesByID) Less(i, j int) bool {
	if _, ok := s[i].(app.Space); !ok {
		return false
	}
	if _, ok := s[j].(app.Space); !ok {
		return false
	}
	return strings.Compare(s[i].(app.Space).ID.String(), s[j].(app.Space).ID.String()) < 0
}

func TestWorkItemPtrSliceSort(t *testing.T) {
	t.Run("by work item title", func(t *testing.T) {
		// given
		a := &app.WorkItem{Attributes: map[string]interface{}{workitem.SystemTitle: "A"}}
		b := &app.WorkItem{Attributes: map[string]interface{}{workitem.SystemTitle: "B"}}
		c := &app.WorkItem{Attributes: map[string]interface{}{workitem.SystemTitle: "C"}}
		s := WorkItemPtrSlice{c, a, b}
		// when
		sort.Sort(s)
		// then
		require.Equal(t, WorkItemPtrSlice{a, b, c}, s)
	})
	t.Run("by work item ID", func(t *testing.T) {
		// given
		a := &app.WorkItem{ID: ptr.UUID(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"))}
		b := &app.WorkItem{ID: ptr.UUID(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"))}
		c := &app.WorkItem{ID: ptr.UUID(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000003"))}
		s := WorkItemPtrSlice{c, a, b}
		// when
		sort.Sort(s)
		// then
		require.Equal(t, WorkItemPtrSlice{a, b, c}, s)
	})
}

func TestWorkItemInterfaceSliceSort(t *testing.T) {
	t.Run("by work item title", func(t *testing.T) {
		// given objects and pointers
		var a interface{} = app.WorkItem{Attributes: map[string]interface{}{workitem.SystemTitle: "A"}}
		var b interface{} = &app.WorkItem{Attributes: map[string]interface{}{workitem.SystemTitle: "B"}}
		var c interface{} = app.WorkItem{Attributes: map[string]interface{}{workitem.SystemTitle: "C"}}
		s := WorkItemInterfaceSlice{c, a, b}
		// when
		sort.Sort(s)
		// then
		require.Equal(t, WorkItemInterfaceSlice{a, b, c}, s)
	})
	t.Run("by work item ID", func(t *testing.T) {
		// given objects and pointers
		var a interface{} = app.WorkItem{ID: ptr.UUID(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"))}
		var b interface{} = &app.WorkItem{ID: ptr.UUID(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000002"))}
		var c interface{} = app.WorkItem{ID: ptr.UUID(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000003"))}
		s := WorkItemInterfaceSlice{c, a, b}
		// when
		sort.Sort(s)
		// then
		require.Equal(t, WorkItemInterfaceSlice{a, b, c}, s)
	})
}
