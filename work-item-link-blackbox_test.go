package main_test

import (
	"fmt"
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemLinkSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type WorkItemLinkSuite struct {
	suite.Suite
	db           *gorm.DB
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	typeCtrl     *WorkitemtypeController
	linkCtrl     *WorkItemLinkController
	workItemCtrl *WorkitemController
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *WorkItemLinkSuite) SetupSuite() {
	fmt.Println("--- Setting up test suite WorkItemLinkSuite ---")

	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	s.db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())

	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}

	svc := goa.New("WorkItemLinkSuite-Service")
	assert.NotNil(s.T(), svc)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.linkTypeCtrl)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.linkCatCtrl)
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.typeCtrl)
	s.linkCtrl = NewWorkItemLinkController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.linkCtrl)
	s.workItemCtrl = NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.workItemCtrl)
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *WorkItemLinkSuite) TearDownSuite() {
	fmt.Println("--- Tearing down test suite WorkItemLinkSuite ---")
	if s.db != nil {
		s.db.Close()
	}
}

// cleanup removes all DB entries that will be created or have been created
// with this test suite. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *WorkItemLinkSuite) cleanup() {
	db := s.db.Unscoped().Delete(&models.WorkItemLinkType{Name: "bug-blocker"})
	db = db.Unscoped().Delete(&models.WorkItemLinkCategory{Name: "user"})
	db = db.Unscoped().Delete(&models.WorkItemType{Name: "foo.bug"})
}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item links that we will create already exist.
func (s *WorkItemLinkSuite) SetupTest() {
	s.T().Log("--- Running SetupTest ---")
	s.cleanup()
}

// The TearDownTest method will be run after every test in the suite.
func (s *WorkItemLinkSuite) TearDownTest() {
	s.T().Log("--- Running TearDownTest ---")
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

// CreateWorkItemType creates a new work item type
func CreateWorkItemType(Name string) *app.CreateWorkItemTypePayload {
	payload := app.CreateWorkItemTypePayload{
		Fields: map[string]*app.FieldDefinition{
			"name": &app.FieldDefinition{
				Required: true,
				Type: &app.FieldType{
					Kind: "string",
				},
			},
		},
		Name: Name,
	}
	return &payload
}

// CreateWorkItem defines a work item link
func CreateWorkItem(workItemType string) *app.CreateWorkItemPayload {
	return &app.CreateWorkItemPayload{
		Type: workItemType,
		Fields: map[string]interface{}{
			"foo": "bar",
		},
	}
}

// CreateWorkItemLinkType defines a work item link type"
func CreateWorkItemLinkType(Name string, sourceType string, targetType string, categoryID string) *app.CreateWorkItemLinkTypePayload {
	description := "Specify that one bug blocks another one."
	lt := models.WorkItemLinkType{
		Name:           Name,
		Description:    &description,
		SourceTypeName: sourceType,
		TargetTypeName: targetType,
		ForwardName:    "forward name string",
		ReverseName:    "reverse name string",
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
	//   3. Create a work item link
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

// createDemoType creates a demo work item link
func (s *WorkItemLinkSuite) createDemoLink() *app.CreateWorkItemLinkPayload {
	// 1. Create at least one work item type
	workItemTypePayload := CreateWorkItemType("foo.bug")
	_, workItemType := test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, workItemTypePayload)
	assert.NotNil(s.T(), workItemType)

	// 2. Create 2 random work item payloads (TODO add ids)
	createWorkItemPayload := CreateWorkItem("foo.bug")
	_, workItem1 := test.CreateWorkitemCreated(s.T(), nil, nil, s.workItemCtrl, createWorkItemPayload)
	assert.NotNil(s.T(), workItem1)
	_, workItem2 := test.CreateWorkitemCreated(s.T(), nil, nil, s.workItemCtrl, createWorkItemPayload)
	assert.NotNil(s.T(), workItem2)

	// 2. Create a work item link category
	createLinkCategoryPayload := CreateWorkItemLinkCategory("user")
	_, workItemLinkCategory := test.CreateWorkItemLinkCategoryCreated(s.T(), nil, nil, s.linkCatCtrl, createLinkCategoryPayload)
	assert.NotNil(s.T(), workItemLinkCategory)

	// 4. Create work item link type payload
	createLinkTypePayload := CreateWorkItemLinkType("bug-blocker", "foo.bug", "foo.bug", *workItemLinkCategory.Data.ID)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.linkTypeCtrl, createLinkTypePayload)
	assert.NotNil(s.T(), workItemLinkType)

	// 5. Work item link (finally, *phew*)
	createLinkPayload := CreateWorkItemLink(workItem1.ID, workItem2.ID, *workItemLinkType.Data.ID)
	return createLinkPayload
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
	createPayload := s.createDemoLink()
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.linkCtrl, createPayload)
	assert.NotNil(s.T(), workItemLink)

	_ = test.DeleteWorkItemLinkOK(s.T(), nil, nil, s.linkCtrl, *workItemLink.Data.ID)
}

//  func (s *WorkItemLinkSuite) TestCreateWorkItemLinkBadRequest() {
//  	createPayload := s.createDemoLinkType("") // empty name causes bad request
//  	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), nil, nil, s.linkTypeCtrl, createPayload)
//  }
//
//  func (s *WorkItemLinkSuite) TestDeleteWorkItemLinkNotFound() {
//  	test.DeleteWorkItemLinkNotFound(s.T(), nil, nil, s.linkTypeCtrl, "1e9a8b53-73a6-40de-b028-5177add79ffa")
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
//  	test.UpdateWorkItemLinkNotFound(s.T(), nil, nil, s.linkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
//  }
//
//  func (s *WorkItemLinkSuite) TestUpdateWorkItemLinkOK() {
//  	createPayload := s.createDemoLinkType("bug-blocker")
//  	_, workItemLinkType := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.linkTypeCtrl, createPayload)
//  	assert.NotNil(s.T(), workItemLinkType)
//  	// Specify new description for link type that we just created
//  	// Wrap data portion in an update payload instead of a create payload
//  	updateLinkTypePayload := &app.UpdateWorkItemLinkPayload{
//  		Data: workItemLinkType.Data,
//  	}
//  	newDescription := "Lalala this is a new description for the work item type"
//  	updateLinkTypePayload.Data.Attributes.Description = &newDescription
//  	_, lt := test.UpdateWorkItemLinkOK(s.T(), nil, nil, s.linkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
//  	assert.NotNil(s.T(), lt.Data)
//  	assert.NotNil(s.T(), lt.Data.Attributes)
//  	assert.NotNil(s.T(), lt.Data.Attributes.Description)
//  	assert.Equal(s.T(), newDescription, *lt.Data.Attributes.Description)
//  }
//
//  func (s *WorkItemLinkSuite) TestUpdateWorkItemLinkBadRequest() {
//  	createPayload := s.createDemoLinkType("bug-blocker")
//  	updateLinkTypePayload := &app.UpdateWorkItemLinkPayload{
//  		Data: createPayload.Data,
//  	}
//  	updateLinkTypePayload.Data.Type = "This should be workitemlinktypes" // Causes bad request
//  	test.UpdateWorkItemLinkBadRequest(s.T(), nil, nil, s.linkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
//  }
//
//  // TestShowWorkItemLinkOK tests if we can fetch the "system" work item link
//  func (s *WorkItemLinkSuite) TestShowWorkItemLinkOK() {
//  	// Create the work item link first and try to read it back in
//  	createPayload := s.createDemoLinkType("bug-blocker")
//  	_, workItemLinkType := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.linkTypeCtrl, createPayload)
//  	assert.NotNil(s.T(), workItemLinkType)
//  	_, readIn := test.ShowWorkItemLinkOK(s.T(), nil, nil, s.linkTypeCtrl, *workItemLinkType.Data.ID)
//  	assert.NotNil(s.T(), readIn)
//  	// Convert to model space and use equal function
//  	expected := models.WorkItemLink{}
//  	actual := models.WorkItemLink{}
//  	assert.Nil(s.T(), models.ConvertLinkTypeToModel(workItemLinkType, &expected))
//  	assert.Nil(s.T(), models.ConvertLinkTypeToModel(readIn, &actual))
//  	assert.True(s.T(), expected.Equal(actual))
//  }
//
//  // TestShowWorkItemLinkNotFound tests if we can fetch a non existing work item link
//  func (s *WorkItemLinkSuite) TestShowWorkItemLinkNotFound() {
//  	test.ShowWorkItemLinkNotFound(s.T(), nil, nil, s.linkTypeCtrl, "88727441-4a21-4b35-aabe-007f8273cd19")
//  }
//
//  // TestListWorkItemLinkOK tests if we can find the work item links
//  // "bug-blocker" and "related" in the list of work item links
//  func (s *WorkItemLinkSuite) TestListWorkItemLinkOK() {
//  	bugBlockerPayload := s.createDemoLinkType("bug-blocker")
//  	_, bugBlockerType := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.linkTypeCtrl, bugBlockerPayload)
//  	assert.NotNil(s.T(), bugBlockerType)
//
//  	relatedPayload := s.createWorkItemLink("related", "foo.bug", "foo.bug", bugBlockerType.Data.Relationships.LinkCategory.Data.ID)
//  	_, relatedType := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.linkTypeCtrl, relatedPayload)
//  	assert.NotNil(s.T(), relatedType)
//
//  	// Fetch a single work item link
//  	_, linkTypeCollection := test.ListWorkItemLinkOK(s.T(), nil, nil, s.linkTypeCtrl)
//  	assert.NotNil(s.T(), linkTypeCollection)
//  	assert.Nil(s.T(), linkTypeCollection.Validate())
//  	// Check the number of found work item links
//  	assert.NotNil(s.T(), linkTypeCollection.Data)
//  	assert.Condition(s.T(), func() bool {
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
//  	assert.Exactly(s.T(), 0, toBeFound, "Not all required work item links (bug-blocker and related) where found.")
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
//  					"source_type": {"data": {"type":"workitemtypes", "id": "foo.bug"}},
//  					"target_type": {"data": {"type":"workitemtypes", "id": "foo.bug"}}
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
//  			method:             http.MethodPut,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getExpiredAuthHeader(t, privatekey),
//  		}, {
//  			method:             http.MethodPut,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getMalformedAuthHeader(t, privatekey),
//  		}, {
//  			method:             http.MethodPut,
//  			url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
//  			expectedStatusCode: http.StatusUnauthorized,
//  			expectedErrorCode:  "jwt_security_error",
//  			payload:            createWorkItemLinkPayloadString,
//  			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
//  		}, {
//  			method:             http.MethodPut,
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
