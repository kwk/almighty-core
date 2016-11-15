package main_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemLinkCategorySuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type WorkItemLinkCategorySuite struct {
	suite.Suite
	db          *gorm.DB
	linkCatCtrl *WorkItemLinkCategoryController
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *WorkItemLinkCategorySuite) SetupSuite() {
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

	svc := goa.New("WorkItemLinkCategorySuite-Service")
	require.NotNil(s.T(), svc)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.linkCatCtrl)
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *WorkItemLinkCategorySuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

// removeWorkItemLinkCategories removes all work item link categories from the db that will be created
// during these tests. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *WorkItemLinkCategorySuite) removeWorkItemLinkCategories() {
	s.db.Unscoped().Delete(&models.WorkItemLinkCategory{Name: "system"})
	s.db.Unscoped().Delete(&models.WorkItemLinkCategory{Name: "user"})
}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item link categories that we will create already exist.
func (s *WorkItemLinkCategorySuite) SetupTest() {
	s.removeWorkItemLinkCategories()
}

// The TearDownTest method will be run after every test in the suite.
func (s *WorkItemLinkCategorySuite) TearDownTest() {
	s.removeWorkItemLinkCategories()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createWorkItemLinkCategorySystem defines a work item link category "system"
func (s *WorkItemLinkCategorySuite) createWorkItemLinkCategorySystem() (http.ResponseWriter, *app.WorkItemLinkCategory) {
	name := "system"
	description := "This work item link category is reserved for the core system."
	id := "0e671e36-871b-43a6-9166-0c4bd573e231"

	// Use the goa generated code to create a work item link category
	payload := app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: models.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}

	return test.CreateWorkItemLinkCategoryCreated(s.T(), nil, nil, s.linkCatCtrl, &payload)
}

// createWorkItemLinkCategoryUser defines a work item link category "user"
func (s *WorkItemLinkCategorySuite) createWorkItemLinkCategoryUser() (http.ResponseWriter, *app.WorkItemLinkCategory) {
	name := "user"
	description := "This work item link category is managed by an admin user."
	id := "bf30167a-9446-42de-82be-6b3815152051"

	// Use the goa generated code to create a work item link category
	payload := app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: models.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}

	return test.CreateWorkItemLinkCategoryCreated(s.T(), nil, nil, s.linkCatCtrl, &payload)
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// TestCreateWorkItemLinkCategory tests if we can create the "system" work item link category
func (s *WorkItemLinkCategorySuite) TestCreateAndDeleteWorkItemLinkCategory() {
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCatSystem)

	_, linkCatUser := s.createWorkItemLinkCategoryUser()
	require.NotNil(s.T(), linkCatUser)

	test.DeleteWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl, *linkCatSystem.Data.ID)
}

func (s *WorkItemLinkCategorySuite) TestCreateWorkItemLinkCategoryBadRequest() {
	description := "New description for work item link category."
	name := "" // This will lead to a bad parameter error
	id := "88727441-4a21-4b35-aabe-007f8273cdBB"
	payload := &app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: models.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
	test.CreateWorkItemLinkCategoryBadRequest(s.T(), nil, nil, s.linkCatCtrl, payload)
}

func (s *WorkItemLinkCategorySuite) TestDeleteWorkItemLinkCategoryNotFound() {
	test.DeleteWorkItemLinkCategoryNotFound(s.T(), nil, nil, s.linkCatCtrl, "01f6c751-53f3-401f-be9b-6a9a230db8AA")
}

func (s *WorkItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryNotFound() {
	description := "New description for work item link category."
	id := "88727441-4a21-4b35-aabe-007f8273cd19"
	payload := &app.UpdateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: models.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Description: &description,
			},
		},
	}
	test.UpdateWorkItemLinkCategoryNotFound(s.T(), nil, nil, s.linkCatCtrl, *payload.Data.ID, payload)
}

func (s *WorkItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryOK() {
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCatSystem)

	description := "New description for work item link category \"system\"."
	updatePayload := &app.UpdateWorkItemLinkCategoryPayload{}
	updatePayload.Data = linkCatSystem.Data
	updatePayload.Data.Attributes.Description = &description

	_, newLinkCat := test.UpdateWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl, *linkCatSystem.Data.ID, updatePayload)

	// Test that description was updated and version got incremented
	require.NotNil(s.T(), newLinkCat.Data.Attributes.Description)
	require.Equal(s.T(), description, *newLinkCat.Data.Attributes.Description)

	require.NotNil(s.T(), newLinkCat.Data.Attributes.Version)
	require.Equal(s.T(), *linkCatSystem.Data.Attributes.Version+1, *newLinkCat.Data.Attributes.Version)
}

func (s *WorkItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryBadRequest() {
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCatSystem)

	description := "New description for work item link category \"system\"."
	updatePayload := &app.UpdateWorkItemLinkCategoryPayload{}
	updatePayload.Data = linkCatSystem.Data
	updatePayload.Data.Attributes.Description = &description
	updatePayload.Data.Type = "this is a wrong type!!!" // "should be workitemlinkcategories"

	_, _ = test.UpdateWorkItemLinkCategoryBadRequest(s.T(), nil, nil, s.linkCatCtrl, *linkCatSystem.Data.ID, updatePayload)
}

// TestShowWorkItemLinkCategoryOK tests if we can fetch the "system" work item link category
func (s *WorkItemLinkCategorySuite) TestShowWorkItemLinkCategoryOK() {
	// Create the work item link category first and try to read it back in
	_, linkCat := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCat)

	_, linkCat2 := test.ShowWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl, *linkCat.Data.ID)

	require.NotNil(s.T(), linkCat2)
	require.EqualValues(s.T(), linkCat, linkCat2)
}

// TestShowWorkItemLinkCategoryNotFound tests if we can fetch a non existing work item link category
func (s *WorkItemLinkCategorySuite) TestShowWorkItemLinkCategoryNotFound() {
	test.ShowWorkItemLinkCategoryNotFound(s.T(), nil, nil, s.linkCatCtrl, "88727441-4a21-4b35-aabe-007f8273cd19")
}

// TestListWorkItemLinkCategoryOK tests if we can find the work item link categories
// "system" and "user" in the list of work item link categories
func (s *WorkItemLinkCategorySuite) TestListWorkItemLinkCategoryOK() {
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCatSystem)
	_, linkCatUser := s.createWorkItemLinkCategoryUser()
	require.NotNil(s.T(), linkCatUser)

	// Fetch a single work item link category
	_, linkCatCollection := test.ListWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl)

	require.NotNil(s.T(), linkCatCollection)
	require.Nil(s.T(), linkCatCollection.Validate())

	// Check the number of found work item link categories
	require.NotNil(s.T(), linkCatCollection.Data)
	require.Condition(s.T(), func() bool {
		return (len(linkCatCollection.Data) >= 2)
	}, "At least two work item link categories must exist (system and user), but only %d exist.", len(linkCatCollection.Data))

	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(linkCatCollection.Data) && toBeFound > 0; i++ {
		if *linkCatCollection.Data[i].Attributes.Name == "system" || *linkCatCollection.Data[i].Attributes.Name == "user" {
			s.T().Log("Found work item link category in collection: ", *linkCatCollection.Data[i].Attributes.Name)
			toBeFound--
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required work item link categories (system and user) where found.")
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemLinkCategory(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(WorkItemLinkCategorySuite))
}

func getWorkItemLinkCategoryTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	if err != nil {
		t.Fatal("Could not parse different private key ", err)
	}

	createWorkItemLinkCategoryPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"attributes": {
					"description": "A sample work item link category",
					"name": "sample",
					"version": 0
				},
				"id": "6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				"type": "workitemlinkcategories"
			}
		}
		`))

	return []testSecureAPI{
		// Create Work Item API with different parameters
		{
			method:             http.MethodPost,
			url:                endpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           "",
		},
		// Update Work Item API with different parameters
		{
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           "",
		},
		// Delete Work Item API with different parameters
		{
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           "",
		},
		// Try fetching a random work item link category
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                endpointWorkItemLinkCategories + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "not_found",
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *WorkItemLinkCategorySuite) TestUnauthorizeWorkItemLinkCategoryCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), getWorkItemLinkCategoryTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemLinkCategory-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemLinkCategoryController(service, gormapplication.NewGormDB(DB))
		app.MountWorkItemLinkCategoryController(service, controller)
		return nil
	})
}
