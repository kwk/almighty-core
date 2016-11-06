package main_test

import (
	"fmt"
	"net/http"
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

// The WorkItemLinkTypeSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type WorkItemLinkTypeSuite struct {
	suite.Suite
	db           *gorm.DB
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	typeCtrl     *WorkitemtypeController
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *WorkItemLinkTypeSuite) SetupSuite() {
	fmt.Println("--- Setting up test suite WorkItemLinkTypeSuite ---")

	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	s.db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())

	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}

	svc := goa.New("WorkItemLinkTypeSuite-Service")
	assert.NotNil(s.T(), svc)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.linkTypeCtrl)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.linkCatCtrl)
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.typeCtrl)
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *WorkItemLinkTypeSuite) TearDownSuite() {
	fmt.Println("--- Tearing down test suite WorkItemLinkTypeSuite ---")
	if s.db != nil {
		s.db.Close()
	}
}

// cleanup removes all DB entries that will be created or have been created
// with this test suite. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *WorkItemLinkTypeSuite) cleanup() {
	db := s.db.Unscoped().Delete(&models.WorkItemLinkType{Name: "bug-blocker"})
	db = db.Unscoped().Delete(&models.WorkItemLinkCategory{Name: "user"})
	db = db.Unscoped().Delete(&models.WorkItemType{Name: "system.bug"})

}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item link types that we will create already exist.
func (s *WorkItemLinkTypeSuite) SetupTest() {
	s.T().Log("--- Running SetupTest ---")
	s.cleanup()
}

// The TearDownTest method will be run after every test in the suite.
func (s *WorkItemLinkTypeSuite) TearDownTest() {
	s.T().Log("--- Running TearDownTest ---")
	s.cleanup()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

func (s *WorkItemLinkTypeSuite) createWorkItemType(Name string) (http.ResponseWriter, *app.WorkItemType) {
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
	return test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, &payload)
}

func (s *WorkItemLinkTypeSuite) createWorkItemLinkCategory(ID string) (http.ResponseWriter, *app.WorkItemLinkCategory) {
	name := "user"
	description := "This work item link category is managed by an admin user."
	// Use the goa generated code to create a work item link category
	payload := app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   ID,
			Type: "workitemlinkcategories",
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
	return test.CreateWorkItemLinkCategoryCreated(s.T(), nil, nil, s.linkCatCtrl, &payload)
}

// createWorkItemLinkTypeBugBlocker defines a work item link type "bug blocker"
func (s *WorkItemLinkTypeSuite) createWorkItemLinkType(ID string, Name string, SourceType string, TargetType string, categoryID string) (http.ResponseWriter, *app.WorkItemLinkType) {
	//   3. Create a work item link type
	description := "Specify that one bug blocks another one."
	linkCatID, _ := satoriuuid.FromString(categoryID)
	id, _ := satoriuuid.FromString(ID)
	lt := models.WorkItemLinkType{
		ID:           id,
		Name:         Name,
		Description:  &description,
		SourceType:   SourceType,
		TargetType:   TargetType,
		ForwardName:  "forward name string",
		ReverseName:  "reverse name string",
		LinkCategory: linkCatID,
	}
	payload := models.ConvertLinkTypeFromModel(&lt)
	fmt.Printf("payload to creat work item link type: \n\n %v\n\n", payload)
	// The create payload is required during creation. Simply copy data over.
	createPayload := app.CreateWorkItemLinkTypePayload{
		Data: payload.Data,
	}
	return test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.linkTypeCtrl, &createPayload)
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemLinkType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(WorkItemLinkTypeSuite))
}

// TestCreateWorkItemLinkType tests if we can create the "system" work item link type
func (s *WorkItemLinkTypeSuite) TestCreateAndDeleteWorkItemLinkType() {
	//   1. Create at least one work item type
	_, workItemType := s.createWorkItemType("system.bug")
	assert.NotNil(s.T(), workItemType)

	//   2. Create a work item link category
	_, workItemLinkCategory := s.createWorkItemLinkCategory("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	assert.NotNil(s.T(), workItemLinkCategory)

	_, workItemLinkType := s.createWorkItemLinkType("40bbdd3d-8b5d-4fd6-ac90-7236b669af04", "bug-blocker", "system.bug", "system.bug", "6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	assert.NotNil(s.T(), workItemLinkType)

	test.DeleteWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, workItemLinkType.Data.ID)
}

// func (s *WorkItemLinkTypeSuite) TestCreateWorkItemLinkTypeBadRequest() {
// 	description := "New description for work item link type."
// 	name := "" // This will lead to a bad parameter error
// 	payload := &app.CreateWorkItemLinkTypePayload{
// 		Data: &app.WorkItemLinkTypeData{
// 			ID:   "88727441-4a21-4b35-aabe-007f8273cdBB",
// 			Type: "workitemlinktypes",
// 			Attributes: &app.WorkItemLinkTypeAttributes{
// 				Name:        &name,
// 				Description: &description,
// 			},
// 		},
// 	}
// 	test.CreateWorkItemLinkTypeBadRequest(s.T(), nil, nil, s.linkTypeCtrl, payload)
// }
//
// func (s *WorkItemLinkTypeSuite) TestDeleteWorkItemLinkTypeNotFound() {
// 	test.DeleteWorkItemLinkTypeNotFound(s.T(), nil, nil, s.linkTypeCtrl, "01f6c751-53f3-401f-be9b-6a9a230db8AA")
// }
//
// func (s *WorkItemLinkTypeSuite) TestUpdateWorkItemLinkTypeNotFound() {
// 	description := "New description for work item link type."
// 	payload := &app.UpdateWorkItemLinkTypePayload{
// 		Data: &app.WorkItemLinkTypeData{
// 			ID:   "88727441-4a21-4b35-aabe-007f8273cd19",
// 			Type: "workitemlinktypes",
// 			Attributes: &app.WorkItemLinkTypeAttributes{
// 				Description: &description,
// 			},
// 		},
// 	}
// 	test.UpdateWorkItemLinkTypeNotFound(s.T(), nil, nil, s.linkTypeCtrl, payload.Data.ID, payload)
// }
//
// func (s *WorkItemLinkTypeSuite) TestUpdateWorkItemLinkTypeOK() {
// 	_, linkCatSystem := s.createWorkItemLinkTypeBugBlocker()
// 	assert.NotNil(s.T(), linkCatSystem)
//
// 	description := "New description for work item link type \"system\"."
// 	updatePayload := &app.UpdateWorkItemLinkTypePayload{}
// 	updatePayload.Data = linkCatSystem.Data
// 	updatePayload.Data.Attributes.Description = &description
//
// 	_, newLinkCat := test.UpdateWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, linkCatSystem.Data.ID, updatePayload)
//
// 	// Test that description was updated and version got incremented
// 	assert.NotNil(s.T(), newLinkCat.Data.Attributes.Description)
// 	assert.Equal(s.T(), description, *newLinkCat.Data.Attributes.Description)
//
// 	assert.NotNil(s.T(), newLinkCat.Data.Attributes.Version)
// 	assert.Equal(s.T(), *linkCatSystem.Data.Attributes.Version+1, *newLinkCat.Data.Attributes.Version)
// }
//
// func (s *WorkItemLinkTypeSuite) TestUpdateWorkItemLinkTypeBadRequest() {
// 	_, linkCatSystem := s.createWorkItemLinkTypeBugBlocker()
// 	assert.NotNil(s.T(), linkCatSystem)
//
// 	description := "New description for work item link type \"system\"."
// 	updatePayload := &app.UpdateWorkItemLinkTypePayload{}
// 	updatePayload.Data = linkCatSystem.Data
// 	updatePayload.Data.Attributes.Description = &description
// 	updatePayload.Data.Type = "this is a wrong type!!!" // "should be workitemlinktypes"
//
// 	_, _ = test.UpdateWorkItemLinkTypeBadRequest(s.T(), nil, nil, s.linkTypeCtrl, linkCatSystem.Data.ID, updatePayload)
// }
//
// // TestShowWorkItemLinkTypeOK tests if we can fetch the "system" work item link type
// func (s *WorkItemLinkTypeSuite) TestShowWorkItemLinkTypeOK() {
// 	// Create the work item link type first and try to read it back in
// 	_, linkCat := s.createWorkItemLinkTypeBugBlocker()
// 	assert.NotNil(s.T(), linkCat)
//
// 	_, linkCat2 := test.ShowWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, linkCat.Data.ID)
//
// 	assert.NotNil(s.T(), linkCat2)
// 	assert.EqualValues(s.T(), linkCat, linkCat2)
// }
//
// // TestShowWorkItemLinkTypeNotFound tests if we can fetch a non existing work item link type
// func (s *WorkItemLinkTypeSuite) TestShowWorkItemLinkTypeNotFound() {
// 	test.ShowWorkItemLinkTypeNotFound(s.T(), nil, nil, s.linkTypeCtrl, "88727441-4a21-4b35-aabe-007f8273cd19")
// }
//
// // TestListWorkItemLinkTypeOK tests if we can find the work item link types
// // "system" and "user" in the list of work item link types
// func (s *WorkItemLinkTypeSuite) TestListWorkItemLinkTypeOK() {
// 	_, linkCatSystem := s.createWorkItemLinkTypeBugBlocker()
// 	assert.NotNil(s.T(), linkCatSystem)
// 	_, linkCatUser := s.createWorkItemLinkTypeUser()
// 	assert.NotNil(s.T(), linkCatUser)
//
// 	// Fetch a single work item link type
// 	_, linkCatCollection := test.ListWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl)
//
// 	assert.NotNil(s.T(), linkCatCollection)
// 	assert.Nil(s.T(), linkCatCollection.Validate())
//
// 	// Check the number of found work item link types
// 	assert.NotNil(s.T(), linkCatCollection.Data)
// 	assert.Condition(s.T(), func() bool {
// 		return (len(linkCatCollection.Data) >= 2)
// 	}, "At least two work item link types must exist (system and user), but only %d exist.", len(linkCatCollection.Data))
//
// 	// Search for the work item types that must exist at minimum
// 	toBeFound := 2
// 	for i := 0; i < len(linkCatCollection.Data) && toBeFound > 0; i++ {
// 		if *linkCatCollection.Data[i].Data.Attributes.Name == "system" || *linkCatCollection.Data[i].Data.Attributes.Name == "user" {
// 			s.T().Log("Found work item link type in collection: ", *linkCatCollection.Data[i].Data.Attributes.Name)
// 			toBeFound--
// 		}
// 	}
// 	assert.Exactly(s.T(), 0, toBeFound, "Not all required work item link types (system and user) where found.")
// }
//
// func getWorkItemLinkTypeTestData(t *testing.T) []testSecureAPI {
// 	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
// 	if err != nil {
// 		t.Fatal("Could not parse Key ", err)
// 	}
// 	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
// 	if err != nil {
// 		t.Fatal("Could not parse different private key ", err)
// 	}
//
// 	createWorkItemLinkTypePayloadString := bytes.NewBuffer([]byte(`
// 		{
// 			"data": {
// 				"attributes": {
// 					"description": "A sample work item link type",
// 					"name": "sample",
// 					"version": 0
// 				},
// 				"id": "6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 				"type": "workitemlinktypes"
// 			}
// 		}
// 		`))
//
// 	return []testSecureAPI{
// 		// Create Work Item API with different parameters
// 		{
// 			method:             http.MethodPost,
// 			url:                endpointWorkItemLinkCategories,
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            createWorkItemLinkTypePayloadString,
// 			jwtToken:           getExpiredAuthHeader(t, privatekey),
// 		}, {
// 			method:             http.MethodPost,
// 			url:                endpointWorkItemLinkCategories,
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            createWorkItemLinkTypePayloadString,
// 			jwtToken:           getMalformedAuthHeader(t, privatekey),
// 		}, {
// 			method:             http.MethodPost,
// 			url:                endpointWorkItemLinkCategories,
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            createWorkItemLinkTypePayloadString,
// 			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
// 		}, {
// 			method:             http.MethodPost,
// 			url:                endpointWorkItemLinkCategories,
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            createWorkItemLinkTypePayloadString,
// 			jwtToken:           "",
// 		},
// 		// Update Work Item API with different parameters
// 		{
// 			method:             http.MethodPut,
// 			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            createWorkItemLinkTypePayloadString,
// 			jwtToken:           getExpiredAuthHeader(t, privatekey),
// 		}, {
// 			method:             http.MethodPut,
// 			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            createWorkItemLinkTypePayloadString,
// 			jwtToken:           getMalformedAuthHeader(t, privatekey),
// 		}, {
// 			method:             http.MethodPut,
// 			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            createWorkItemLinkTypePayloadString,
// 			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
// 		}, {
// 			method:             http.MethodPut,
// 			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            createWorkItemLinkTypePayloadString,
// 			jwtToken:           "",
// 		},
// 		// Delete Work Item API with different parameters
// 		{
// 			method:             http.MethodDelete,
// 			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            nil,
// 			jwtToken:           getExpiredAuthHeader(t, privatekey),
// 		}, {
// 			method:             http.MethodDelete,
// 			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            nil,
// 			jwtToken:           getMalformedAuthHeader(t, privatekey),
// 		}, {
// 			method:             http.MethodDelete,
// 			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            nil,
// 			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
// 		}, {
// 			method:             http.MethodDelete,
// 			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
// 			expectedStatusCode: http.StatusUnauthorized,
// 			expectedErrorCode:  "jwt_security_error",
// 			payload:            nil,
// 			jwtToken:           "",
// 		},
// 		// Try fetching a random work item link type
// 		// We do not have security on GET hence this should return 404 not found
// 		{
// 			method:             http.MethodGet,
// 			url:                endpointWorkItemLinkCategories + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
// 			expectedStatusCode: http.StatusNotFound,
// 			expectedErrorCode:  "not_found",
// 			payload:            nil,
// 			jwtToken:           "",
// 		},
// 	}
// }
//
// // This test case will check authorized access to Create/Update/Delete APIs
// func TestUnauthorizeWorkItemLinkTypeCUD(t *testing.T) {
// 	UnauthorizeCreateUpdateDeleteTest(t, getWorkItemLinkTypeTestData, func() *goa.Service {
// 		return goa.New("TestUnauthorizedCreateWorkItemLinkType-Service")
// 	}, func(service *goa.Service) error {
// 		controller := NewWorkItemLinkTypeController(service, gormapplication.NewGormDB(DB))
// 		app.MountWorkItemLinkTypeController(service, controller)
// 		return nil
// 	})
// }
