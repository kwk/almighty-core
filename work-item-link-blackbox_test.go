package main_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

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
	bug1ID               string
	bug2ID               string
	feature1ID           string
	userLinkCategoryID   string
	bugBlockerLinkTypeID string

	// Store IDs of resources that need to be removed at the beginning or end of a test
	deleteWorkItemLinkCategories []string
	deleteWorkItemLinkTypes      []string
	deleteWorkItemLinks          []string
	deleteWorkItems              []string
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

	// Delete work item link types
	db = db.Unscoped().Delete(&models.WorkItemLinkType{Name: "bug-blocker"})
	db = db.Unscoped().Delete(&models.WorkItemLinkCategory{Name: "user"})
	//for _, id := range s.deleteWorkItemLinkTypes {
	//	db = db.Unscoped().Delete(&models.WorkItemLinkType{ID: satoriuuid.FromStringOrNil(id)})
	//	require.Nil(s.T(), db.Error)
	//}

	//// Delete work item link categories
	//for _, id := range s.deleteWorkItemLinkCategories {
	//	db = db.Unscoped().Delete(&models.WorkItemLinkCategory{ID: satoriuuid.FromStringOrNil(id)})
	//	require.Nil(s.T(), db.Error)
	//}

	// Last but not least delete the work items
	for _, idStr := range s.deleteWorkItems {
		id, err := strconv.ParseUint(idStr, 10, 64)
		require.Nil(s.T(), err)
		db = db.Unscoped().Delete(&models.WorkItem{ID: id})
		require.Nil(s.T(), db.Error)
	}
}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item links that we will create already exist.
// It will also make sure that some resources that we rely on do exists.
func (s *WorkItemLinkSuite) SetupTest() {
	s.cleanup()

	// Create 3 work items (bug1, bug2, and feature1)
	bug1Payload := CreateWorkItem(models.SystemBug, "bug1")
	_, bug1 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, bug1Payload)
	require.NotNil(s.T(), bug1)
	s.deleteWorkItems = append(s.deleteWorkItems, bug1.ID)
	s.bug1ID = bug1.ID
	fmt.Printf("Created bug1 with ID: %s\n", bug1.ID)

	bug2Payload := CreateWorkItem(models.SystemBug, "bug2")
	_, bug2 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, bug2Payload)
	require.NotNil(s.T(), bug2)
	s.deleteWorkItems = append(s.deleteWorkItems, bug2.ID)
	s.bug2ID = bug2.ID
	fmt.Printf("Created bug2 with ID: %s\n", bug2.ID)

	feature1Payload := CreateWorkItem(models.SystemFeature, "feature1")
	_, feature1 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, feature1Payload)
	require.NotNil(s.T(), feature1)
	s.deleteWorkItems = append(s.deleteWorkItems, feature1.ID)
	s.feature1ID = feature1.ID
	fmt.Printf("Created feature with ID: %s\n", feature1.ID)

	// Create a work item link category
	createLinkCategoryPayload := CreateWorkItemLinkCategory("user")
	_, workItemLinkCategory := test.CreateWorkItemLinkCategoryCreated(s.T(), nil, nil, s.workItemLinkCategoryCtrl, createLinkCategoryPayload)
	require.NotNil(s.T(), workItemLinkCategory)
	s.deleteWorkItemLinkCategories = append(s.deleteWorkItemLinkCategories, *workItemLinkCategory.Data.ID)
	s.userLinkCategoryID = *workItemLinkCategory.Data.ID
	fmt.Printf("Created link category with ID: %s\n", *workItemLinkCategory.Data.ID)

	// Create work item link type payload
	createLinkTypePayload := CreateWorkItemLinkType("bug-blocker", models.SystemBug, models.SystemBug, s.userLinkCategoryID)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.workItemLinkTypeCtrl, createLinkTypePayload)
	require.NotNil(s.T(), workItemLinkType)
	s.deleteWorkItemLinkTypes = append(s.deleteWorkItemLinkTypes, *workItemLinkType.Data.ID)
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
func CreateWorkItemLink(sourceID string, targetID string, linkTypeID string) *app.CreateWorkItemLinkPayload {
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

//func (s *WorkItemLinkSuite) TestCreateWorkItemLinkBadRequest() {
//	// Linking a bug and a feature isn't allowed for the bug blocker link type,
//	// thererfore this will cause a bad parameter error (which results in a bad request error).
//	createPayload := CreateWorkItemLink(s.bug1ID, s.feature1ID, s.bugBlockerLinkTypeID)
//	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), nil, nil, s.workItemLinkCtrl, createPayload)
//}

//  func (s *WorkItemLinkSuite) TestDeleteWorkItemLinkNotFound() {
//  	test.DeleteWorkItemLinkNotFound(s.T(), nil, nil, s.workItemLinkTypeCtrl, "1e9a8b53-73a6-40de-b028-5177add79ffa")
//  }
//
//  func (s *WorkItemLinkSuite) TestUpdateWorkItemLinkNotFound() {
//  	createPayload := s.createDemoLinkType("bug-blocker")
//  	notExistingId := satoriuuid.FromStringOrNil("46bbce9c-8219-4364-a450-dfd1b501654e") // This ID does not exist
//  	notExistingIdStr := notExistingId.String()
//  	createPayload.Data.ID = &notExistingIdStr
//  	// Wrap data portion in an update payload instead of a create payload
//  	updateLinkTypePayload := &app.UpdateWorkItemLinkPayload{
//  		Data: createPayload.Data,
//  	}
//  	test.UpdateWorkItemLinkNotFound(s.T(), nil, nil, s.workItemLinkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
//  }
//
//  func (s *WorkItemLinkSuite) TestUpdateWorkItemLinkOK() {
//  	createPayload := s.createDemoLinkType("bug-blocker")
//  	_, workItemLinkType := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkTypeCtrl, createPayload)
//  	require.NotNil(s.T(), workItemLinkType)
//  	// Specify new description for link type that we just created
//  	// Wrap data portion in an update payload instead of a create payload
//  	updateLinkTypePayload := &app.UpdateWorkItemLinkPayload{
//  		Data: workItemLinkType.Data,
//  	}
//  	newDescription := "Lalala this is a new description for the work item type"
//  	updateLinkTypePayload.Data.Attributes.Description = &newDescription
//  	_, lt := test.UpdateWorkItemLinkOK(s.T(), nil, nil, s.workItemLinkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
//  	require.NotNil(s.T(), lt.Data)
//  	require.NotNil(s.T(), lt.Data.Attributes)
//  	require.NotNil(s.T(), lt.Data.Attributes.Description)
//  	require.Equal(s.T(), newDescription, *lt.Data.Attributes.Description)
//  }
//
//  func (s *WorkItemLinkSuite) TestUpdateWorkItemLinkBadRequest() {
//  	createPayload := s.createDemoLinkType("bug-blocker")
//  	updateLinkTypePayload := &app.UpdateWorkItemLinkPayload{
//  		Data: createPayload.Data,
//  	}
//  	updateLinkTypePayload.Data.Type = "This should be workitemlinktypes" // Causes bad request
//  	test.UpdateWorkItemLinkBadRequest(s.T(), nil, nil, s.workItemLinkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
//  }
//
//  // TestShowWorkItemLinkOK tests if we can fetch the "system" work item link
//  func (s *WorkItemLinkSuite) TestShowWorkItemLinkOK() {
//  	// Create the work item link first and try to read it back in
//  	createPayload := s.createDemoLinkType("bug-blocker")
//  	_, workItemLinkType := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkTypeCtrl, createPayload)
//  	require.NotNil(s.T(), workItemLinkType)
//  	_, readIn := test.ShowWorkItemLinkOK(s.T(), nil, nil, s.workItemLinkTypeCtrl, *workItemLinkType.Data.ID)
//  	require.NotNil(s.T(), readIn)
//  	// Convert to model space and use equal function
//  	expected := models.WorkItemLink{}
//  	actual := models.WorkItemLink{}
//  	require.Nil(s.T(), models.ConvertLinkTypeToModel(workItemLinkType, &expected))
//  	require.Nil(s.T(), models.ConvertLinkTypeToModel(readIn, &actual))
//  	require.True(s.T(), expected.Equal(actual))
//  }
//
//  // TestShowWorkItemLinkNotFound tests if we can fetch a non existing work item link
//  func (s *WorkItemLinkSuite) TestShowWorkItemLinkNotFound() {
//  	test.ShowWorkItemLinkNotFound(s.T(), nil, nil, s.workItemLinkTypeCtrl, "88727441-4a21-4b35-aabe-007f8273cd19")
//  }
//
//  // TestListWorkItemLinkOK tests if we can find the work item links
//  // "bug-blocker" and "related" in the list of work item links
//  func (s *WorkItemLinkSuite) TestListWorkItemLinkOK() {
//  	bugBlockerPayload := s.createDemoLinkType("bug-blocker")
//  	_, bugBlockerType := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkTypeCtrl, bugBlockerPayload)
//  	require.NotNil(s.T(), bugBlockerType)
//
//  	relatedPayload := s.createWorkItemLink("related", "foo.bug", "foo.bug", bugBlockerType.Data.Relationships.LinkCategory.Data.ID)
//  	_, relatedType := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkTypeCtrl, relatedPayload)
//  	require.NotNil(s.T(), relatedType)
//
//  	// Fetch a single work item link
//  	_, linkTypeCollection := test.ListWorkItemLinkOK(s.T(), nil, nil, s.workItemLinkTypeCtrl)
//  	require.NotNil(s.T(), linkTypeCollection)
//  	require.Nil(s.T(), linkTypeCollection.Validate())
//  	// Check the number of found work item links
//  	require.NotNil(s.T(), linkTypeCollection.Data)
//  	require.Condition(s.T(), func() bool {
//  		return (len(linkTypeCollection.Data) >= 2)
//  	}, "At least two work item links must exist (bug-blocker and related), but only %d exist.", len(linkTypeCollection.Data))
//  	// Search for the work item types that must exist at minimum
//  	toBeFound := 2
//  	for i := 0; i < len(linkTypeCollection.Data) && toBeFound > 0; i++ {
//  		if *linkTypeCollection.Data[i].Data.Attributes.Name == "bug-blocker" || *linkTypeCollection.Data[i].Data.Attributes.Name == "related" {
//  			s.T().Log("Found work item link in collection: ", *linkTypeCollection.Data[i].Data.Attributes.Name)
//  			toBeFound--
//  		}
//  	}
//  	require.Exactly(s.T(), 0, toBeFound, "Not all required work item links (bug-blocker and related) where found.")
//  }
//
//  func getWorkItemLinkTestData(t *testing.T) []testSecureAPI {
//  	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
//  	if err != nil {
//  		t.Fatal("Could not parse Key ", err)
//  	}
//  	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
//  	if err != nil {
//  		t.Fatal("Could not parse different private key ", err)
//  	}
//
//  	createWorkItemLinkPayloadString := bytes.NewBuffer([]byte(`
//  		{
//  			"data": {
//  				"type": "workitemlinktypes",
//  				"id": "0270e113-7790-477f-9371-97c37d734d5d",
//  				"attributes": {
//  					"name": "sample",
//  					"description": "A sample work item link",
//  					"version": 0,
//  					"forward_name": "forward string name",
//  					"reverse_name": "reverse string name"
//  				},
//  				"relationships": {
//  					"link_category": {"data": {"type":"workitemlinkcategories", "id": "a75ea296-6378-4578-8573-90f11b8efb00"}},
//  					"source_type": {"data": {"type":"workitemtypes", "id": "system.bug"}},
//  					"target_type": {"data": {"type":"workitemtypes", "id": "system.bug"}}
//  				}
//  			}
//  		}
//  		`))
//  	return []testSecureAPI{
//  		// Create Work Item API with different parameters
//  		{
//  			method:             http.MethodPost,
//  			url:                endpointWorkItemLinks,
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getExpiredAuthHeader(t, privatekey),
//  		}, {
//  			method:             http.MethodPost,
//  			url:                endpointWorkItemLinks,
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getMalformedAuthHeader(t, privatekey),
//  		}, {
//  			method:             http.MethodPost,
//  			url:                endpointWorkItemLinks,
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
//  		}, {
//  			method:             http.MethodPost,
//  			url:                endpointWorkItemLinks,
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           "",
//  		},
//  		// Update Work Item API with different parameters
//  		{
//  			method:             http.MethodPatch,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getExpiredAuthHeader(t, privatekey),
//  		}, {
//  			method:             http.MethodPatch,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getMalformedAuthHeader(t, privatekey),
//  		}, {
//  			method:             http.MethodPatch,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
//  		}, {
//  			method:             http.MethodPatch,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           "",
//  		},
//  		// Delete Work Item API with different parameters
//  		{
//  			method:             http.MethodDelete,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            nil,
//  			jwtToken:           getExpiredAuthHeader(t, privatekey),
//  		}, {
//  			method:             http.MethodDelete,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            nil,
//  			jwtToken:           getMalformedAuthHeader(t, privatekey),
//  		}, {
//  			method:             http.MethodDelete,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            nil,
//  			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
//  		}, {
//  			method:             http.MethodDelete,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            nil,
//  			jwtToken:           "",
//  		},
//  		// Try fetching a random work item link
//  		// We do not have security on GET hence this should return 404 not found
//  		{
//  			method:             http.MethodGet,
//  			url:                endpointWorkItemLinks + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
//  			expectedStatusCode: http.StatusNotFound,
//  			expectedErrorCode:  "not_found",
//  			payload:            nil,
//  			jwtToken:           "",
//  		},
//  	}
//  }
//
//  // This test case will check authorized access to Create/Update/Delete APIs
//  func (s *WorkItemLinkSuite) TestUnauthorizeWorkItemLinkCUD() {
//  	UnauthorizeCreateUpdateDeleteTest(s.T(), getWorkItemLinkTestData, func() *goa.Service {
//  		return goa.New("TestUnauthorizedCreateWorkItemLink-Service")
//  	}, func(service *goa.Service) error {
//  		controller := NewWorkItemLinkController(service, gormapplication.NewGormDB(DB))
//  		app.MountWorkItemLinkController(service, controller)
//  		return nil
//  	})
//  }
//
