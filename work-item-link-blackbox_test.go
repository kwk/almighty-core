package main_test

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"golang.org/x/net/context"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemLinkSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type WorkItemLinkSuite struct {
	suite.Suite
	db                       *gorm.DB
	workItemLinkTypeCtrl     *WorkItemLinkTypeController
	workItemLinkCategoryCtrl *WorkItemLinkCategoryController
	workItemLinkCtrl         *WorkItemLinkController
	workItemCtrl             *WorkitemController
	workItemSvc              *goa.Service

	// These IDs can safely be used by all tests
	bug1ID               uint64
	bug2ID               uint64
	bug3ID               uint64
	feature1ID           uint64
	userLinkCategoryID   string
	bugBlockerLinkTypeID string

	// Store IDs of resources that need to be removed at the beginning or end of a test
	deleteWorkItemLinks []string
	deleteWorkItems     []string
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *WorkItemLinkSuite) SetupSuite() {
	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	s.db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())

	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}

	// Make sure the database is populated with the correct types (e.g. system.bug etc.)
	if err := models.Transactional(DB, func(tx *gorm.DB) error {
		return migration.PopulateCommonTypes(context.Background(), tx, models.NewWorkItemTypeRepository(tx))
	}); err != nil {
		panic(err.Error())
	}

	pub, err := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	require.Nil(s.T(), err)
	priv, err := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	require.Nil(s.T(), err)

	svc := goa.New("TestWorkItemLinkType-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemLinkTypeCtrl)

	svc = goa.New("TestWorkItemLinkCategory-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkCategoryCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemLinkCategoryCtrl)

	svc = goa.New("TestWorkItemLink-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkCtrl = NewWorkItemLinkController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemLinkCtrl)

	s.workItemSvc = testsupport.ServiceAsUser("TestWorkItem-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	require.NotNil(s.T(), s.workItemSvc)
	s.workItemCtrl = NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemCtrl)
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *WorkItemLinkSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

// cleanup removes all DB entries that will be created or have been created
// with this test suite. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *WorkItemLinkSuite) cleanup() {
	db := s.db

	// First delete work item links and then the types;
	// otherwise referential integrity will be violated.
	for _, id := range s.deleteWorkItemLinks {
		db = db.Unscoped().Delete(&models.WorkItemLink{ID: satoriuuid.FromStringOrNil(id)})
		require.Nil(s.T(), db.Error)
	}
	s.deleteWorkItemLinks = nil

	// Delete all work item links for now
	db.Unscoped().Delete(&models.WorkItemLink{})
	require.Nil(s.T(), db.Error)

	// Delete work item link types and categories by name.
	// They will be created during the tests but have to be deleted by name
	// rather than ID, unlike the work items or work item links.
	db = db.Unscoped().Delete(&models.WorkItemLinkType{Name: "bug-blocker"})
	require.Nil(s.T(), db.Error)
	db = db.Unscoped().Delete(&models.WorkItemLinkCategory{Name: "user"})
	require.Nil(s.T(), db.Error)

	// Last but not least delete the work items
	for _, idStr := range s.deleteWorkItems {
		id, err := strconv.ParseUint(idStr, 10, 64)
		require.Nil(s.T(), err)
		db = db.Unscoped().Delete(&models.WorkItem{ID: id})
		require.Nil(s.T(), db.Error)
	}
	s.deleteWorkItems = nil

}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item links that we will create already exist.
// It will also make sure that some resources that we rely on do exists.
func (s *WorkItemLinkSuite) SetupTest() {
	s.cleanup()

	var err error

	// Create 3 work items (bug1, bug2, and feature1)
	bug1Payload := CreateWorkItem(models.SystemBug, "bug1")
	_, bug1 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, bug1Payload)
	require.NotNil(s.T(), bug1)
	s.deleteWorkItems = append(s.deleteWorkItems, bug1.ID)
	s.bug1ID, err = strconv.ParseUint(bug1.ID, 10, 64)
	require.Nil(s.T(), err)
	fmt.Printf("Created bug1 with ID: %s\n", bug1.ID)

	bug2Payload := CreateWorkItem(models.SystemBug, "bug2")
	_, bug2 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, bug2Payload)
	require.NotNil(s.T(), bug2)
	s.deleteWorkItems = append(s.deleteWorkItems, bug2.ID)
	s.bug2ID, err = strconv.ParseUint(bug2.ID, 10, 64)
	require.Nil(s.T(), err)
	fmt.Printf("Created bug2 with ID: %s\n", bug2.ID)

	bug3Payload := CreateWorkItem(models.SystemBug, "bug3")
	_, bug3 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, bug3Payload)
	require.NotNil(s.T(), bug3)
	s.deleteWorkItems = append(s.deleteWorkItems, bug3.ID)
	s.bug3ID, err = strconv.ParseUint(bug2.ID, 10, 64)
	require.Nil(s.T(), err)
	fmt.Printf("Created bug3 with ID: %s\n", bug3.ID)

	feature1Payload := CreateWorkItem(models.SystemFeature, "feature1")
	_, feature1 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, feature1Payload)
	require.NotNil(s.T(), feature1)
	s.deleteWorkItems = append(s.deleteWorkItems, feature1.ID)
	s.feature1ID, err = strconv.ParseUint(feature1.ID, 10, 64)
	require.Nil(s.T(), err)
	fmt.Printf("Created feature with ID: %s\n", feature1.ID)

	// Create a work item link category
	createLinkCategoryPayload := CreateWorkItemLinkCategory("user")
	_, workItemLinkCategory := test.CreateWorkItemLinkCategoryCreated(s.T(), nil, nil, s.workItemLinkCategoryCtrl, createLinkCategoryPayload)
	require.NotNil(s.T(), workItemLinkCategory)
	//s.deleteWorkItemLinkCategories = append(s.deleteWorkItemLinkCategories, *workItemLinkCategory.Data.ID)
	s.userLinkCategoryID = *workItemLinkCategory.Data.ID
	fmt.Printf("Created link category with ID: %s\n", *workItemLinkCategory.Data.ID)

	// Create work item link type payload
	createLinkTypePayload := CreateWorkItemLinkType("bug-blocker", models.SystemBug, models.SystemBug, s.userLinkCategoryID)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.workItemLinkTypeCtrl, createLinkTypePayload)
	require.NotNil(s.T(), workItemLinkType)
	//s.deleteWorkItemLinkTypes = append(s.deleteWorkItemLinkTypes, *workItemLinkType.Data.ID)
	s.bugBlockerLinkTypeID = *workItemLinkType.Data.ID
	fmt.Printf("Created link type with ID: %s\n", *workItemLinkType.Data.ID)
}

// The TearDownTest method will be run after every test in the suite.
func (s *WorkItemLinkSuite) TearDownTest() {
	s.cleanup()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// CreateWorkItemLinkCategory creates a work item link category
func CreateWorkItemLinkCategory(name string) *app.CreateWorkItemLinkCategoryPayload {
	description := "This work item link category is managed by an admin user."
	// Use the goa generated code to create a work item link category
	return &app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			Type: models.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
}

// CreateWorkItem defines a work item link
func CreateWorkItem(workItemType string, title string) *app.CreateWorkItemPayload {
	payload := app.CreateWorkItemPayload{
		Type: workItemType,
		Fields: map[string]interface{}{
			models.SystemTitle:   title,
			models.SystemCreator: "konrad",
			models.SystemState:   "closed"},
	}
	return &payload
}

// CreateWorkItemLinkType defines a work item link type
func CreateWorkItemLinkType(name string, sourceType string, targetType string, categoryID string) *app.CreateWorkItemLinkTypePayload {
	description := "Specify that one bug blocks another one."
	lt := models.WorkItemLinkType{
		Name:           name,
		Description:    &description,
		SourceTypeName: sourceType,
		TargetTypeName: targetType,
		ForwardName:    "forward name string for " + name,
		ReverseName:    "reverse name string for " + name,
		LinkCategoryID: satoriuuid.FromStringOrNil(categoryID),
	}
	payload := models.ConvertLinkTypeFromModel(&lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.CreateWorkItemLinkTypePayload{
		Data: payload.Data,
	}
}

// CreateWorkItemLink defines a work item link
func CreateWorkItemLink(sourceID uint64, targetID uint64, linkTypeID string) *app.CreateWorkItemLinkPayload {
	lt := models.WorkItemLink{
		SourceID:   sourceID,
		TargetID:   targetID,
		LinkTypeID: satoriuuid.FromStringOrNil(linkTypeID),
	}
	payload := models.ConvertLinkFromModel(&lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.CreateWorkItemLinkPayload{
		Data: payload.Data,
	}
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemLink(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(WorkItemLinkSuite))
}

// TestCreateWorkItemLink tests if we can create the work item link
func (s *WorkItemLinkSuite) TestCreateAndDeleteWorkItemLink() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkCtrl, createPayload)
	require.NotNil(s.T(), workItemLink)

	// Delete this work item link during cleanup
	s.deleteWorkItemLinks = append(s.deleteWorkItemLinks, *workItemLink.Data.ID)

	_ = test.DeleteWorkItemLinkOK(s.T(), nil, nil, s.workItemLinkCtrl, *workItemLink.Data.ID)
}

func (s *WorkItemLinkSuite) TestCreateWorkItemLinkBadRequest() {
	// Linking a bug and a feature isn't allowed for the bug blocker link type,
	// thererfore this will cause a bad parameter error (which results in a bad request error).
	createPayload := CreateWorkItemLink(s.bug1ID, s.feature1ID, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), nil, nil, s.workItemLinkCtrl, createPayload)
}

func (s *WorkItemLinkSuite) TestDeleteWorkItemLinkNotFound() {
	test.DeleteWorkItemLinkNotFound(s.T(), nil, nil, s.workItemLinkCtrl, "1e9a8b53-73a6-40de-b028-5177add79ffa")
}

func (s *WorkItemLinkSuite) TestUpdateWorkItemLinkNotFound() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.userLinkCategoryID)
	notExistingId := satoriuuid.FromStringOrNil("46bbce9c-8219-4364-a450-dfd1b501654e") // This ID does not exist
	notExistingIdStr := notExistingId.String()
	createPayload.Data.ID = &notExistingIdStr
	// Wrap data portion in an update payload instead of a create payload
	updateLinkPayload := &app.UpdateWorkItemLinkPayload{
		Data: createPayload.Data,
	}
	test.UpdateWorkItemLinkNotFound(s.T(), nil, nil, s.workItemLinkCtrl, *updateLinkPayload.Data.ID, updateLinkPayload)
}

func (s *WorkItemLinkSuite) TestUpdateWorkItemLinkOK() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkCtrl, createPayload)
	require.NotNil(s.T(), workItemLink)
	// Delete this work item link during cleanup
	s.deleteWorkItemLinks = append(s.deleteWorkItemLinks, *workItemLink.Data.ID)
	// Specify new description for link type that we just created
	// Wrap data portion in an update payload instead of a create payload
	updateLinkPayload := &app.UpdateWorkItemLinkPayload{
		Data: workItemLink.Data,
	}
	updateLinkPayload.Data.Relationships.Target.Data.ID = strconv.FormatUint(s.bug3ID, 10)
	_, l := test.UpdateWorkItemLinkOK(s.T(), nil, nil, s.workItemLinkCtrl, *updateLinkPayload.Data.ID, updateLinkPayload)
	require.NotNil(s.T(), l.Data)
	require.NotNil(s.T(), l.Data.Relationships)
	require.NotNil(s.T(), l.Data.Relationships.Target.Data)
	require.Equal(s.T(), strconv.FormatUint(s.bug3ID, 10), l.Data.Relationships.Target.Data.ID)
}

func (s *WorkItemLinkSuite) TestUpdateWorkItemLinkBadRequest() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	updateLinkPayload := &app.UpdateWorkItemLinkPayload{
		Data: createPayload.Data,
	}
	updateLinkPayload.Data.Type = "This should be workitemlinks" // Causes bad request
	test.UpdateWorkItemLinkBadRequest(s.T(), nil, nil, s.workItemLinkCtrl, *updateLinkPayload.Data.ID, updateLinkPayload)
}

// TestShowWorkItemLinkOK tests if we can fetch the "system" work item link
func (s *WorkItemLinkSuite) TestShowWorkItemLinkOK() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkCtrl, createPayload)
	require.NotNil(s.T(), workItemLink)
	// Delete this work item link during cleanup
	s.deleteWorkItemLinks = append(s.deleteWorkItemLinks, *workItemLink.Data.ID)
	expected := models.WorkItemLink{}
	require.Nil(s.T(), models.ConvertLinkToModel(workItemLink, &expected))

	_, readIn := test.ShowWorkItemLinkOK(s.T(), nil, nil, s.workItemLinkCtrl, *workItemLink.Data.ID)
	require.NotNil(s.T(), readIn)
	// Convert to model space and use equal function
	actual := models.WorkItemLink{}
	require.Nil(s.T(), models.ConvertLinkToModel(readIn, &actual))
	require.True(s.T(), expected.Equal(actual))
}

func (s *WorkItemLinkSuite) TestShowWorkItemLinkNotFoundDueToBadID() {
	_ = test.ShowWorkItemLinkNotFound(s.T(), nil, nil, s.workItemLinkCtrl, "something that is not a UUID")
}

// TestShowWorkItemLinkNotFound tests if we can fetch a non existing work item link
func (s *WorkItemLinkSuite) TestShowWorkItemLinkNotFound() {
	test.ShowWorkItemLinkNotFound(s.T(), nil, nil, s.workItemLinkCtrl, "88727441-4a21-4b35-aabe-007f8273cd19")
}

// TestListWorkItemLinkOK tests if we can find the work item links
// "bug-blocker" and "related" in the list of work item links
func (s *WorkItemLinkSuite) TestListWorkItemLinkOK() {
	createPayload1 := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink1 := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkCtrl, createPayload1)
	require.NotNil(s.T(), workItemLink1)
	// Delete this work item link during cleanup
	s.deleteWorkItemLinks = append(s.deleteWorkItemLinks, *workItemLink1.Data.ID)
	expected1 := models.WorkItemLink{}
	require.Nil(s.T(), models.ConvertLinkToModel(workItemLink1, &expected1))

	createPayload2 := CreateWorkItemLink(s.bug2ID, s.bug3ID, s.bugBlockerLinkTypeID)
	_, workItemLink2 := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkCtrl, createPayload2)
	require.NotNil(s.T(), workItemLink2)
	// Delete this work item link during cleanup
	s.deleteWorkItemLinks = append(s.deleteWorkItemLinks, *workItemLink2.Data.ID)
	expected2 := models.WorkItemLink{}
	require.Nil(s.T(), models.ConvertLinkToModel(workItemLink2, &expected2))

	// Fetch a single work item link
	_, linkCollection := test.ListWorkItemLinkOK(s.T(), nil, nil, s.workItemLinkCtrl)
	require.NotNil(s.T(), linkCollection)
	require.Nil(s.T(), linkCollection.Validate())
	// Check the number of found work item links
	require.NotNil(s.T(), linkCollection.Data)
	require.Condition(s.T(), func() bool {
		return (len(linkCollection.Data) >= 2)
	}, "At least two work item links must exist (%s and %s), but only %d exist.", *workItemLink1.Data.ID, *workItemLink2.Data.ID, len(linkCollection.Data))
	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(linkCollection.Data) && toBeFound > 0; i++ {
		if *linkCollection.Data[i].ID == *workItemLink1.Data.ID || *linkCollection.Data[i].ID == *workItemLink2.Data.ID {
			s.T().Log("Found work item link in collection: ", *linkCollection.Data[i].ID)
			toBeFound--
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required work item links (%s and %s) where found.", *workItemLink1.Data.ID, *workItemLink2.Data.ID)
}

func getWorkItemLinkTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	if err != nil {
		t.Fatal("Could not parse different private key ", err)
	}

	createWorkItemLinkPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"attributes": {
					"version": 0
				},
				"id": "40bbdd3d-8b5d-4fd6-ac90-7236b669af04",
				"relationships": {
					"link_type": {
						"data": {
						"id": "6c5610be-30b2-4880-9fec-81e4f8e4fd76",
						"type": "workitemlinktypes"
						}
					},
					"source": {
						"data": {
						"id": "1234",
						"type": "workitems"
						}
					},
					"target": {
						"data": {
						"id": "1234",
						"type": "workitems"
						}
					}
				},
				"type": "workitemlinks"
			}
		}
  		`))
	return []testSecureAPI{
		// Create Work Item API with different parameters
		{
			method:             http.MethodPost,
			url:                endpointWorkItemLinks,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinks,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinks,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinks,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkPayloadString,
			jwtToken:           "",
		},
		// Update Work Item API with different parameters
		{
			method:             http.MethodPatch,
			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkPayloadString,
			jwtToken:           "",
		},
		// Delete Work Item API with different parameters
		{
			method:             http.MethodDelete,
			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           "",
		},
		// Try fetching a random work item link
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                endpointWorkItemLinks + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "not_found",
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *WorkItemLinkSuite) TestUnauthorizeWorkItemLinkCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), getWorkItemLinkTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemLink-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemLinkController(service, gormapplication.NewGormDB(DB))
		app.MountWorkItemLinkController(service, controller)
		return nil
	})
}

func TestNewWorkItemLinkControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemLinkController(nil, nil)
	})
}
