package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
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
	fmt.Println("--- Setting up test suite WorkItemLinkCategorySuite ---")

	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	s.db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())

	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}

	svc := goa.New("WorkItemLinkCategorySuite-Service")
	assert.NotNil(s.T(), svc)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(s.T(), s.linkCatCtrl)
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *WorkItemLinkCategorySuite) TearDownSuite() {
	fmt.Println("--- Tearing down test suite WorkItemLinkCategorySuite ---")
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
	s.T().Log("--- Running SetupTest ---")
	s.removeWorkItemLinkCategories()
}

// The TearDownTest method will be run after every test in the suite.
func (s *WorkItemLinkCategorySuite) TearDownTest() {
	s.T().Log("--- Running TearDownTest ---")
	s.removeWorkItemLinkCategories()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createWorkItemLinkCategorySystem defines a work item link category "system"
func (s *WorkItemLinkCategorySuite) createWorkItemLinkCategorySystem() (http.ResponseWriter, *app.WorkItemLinkCategory) {
	name := "system"
	description := "This work item link category is reserved for the core system."

	// Use the goa generated code to create a work item link category
	payload := app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   "0e671e36-871b-43a6-9166-0c4bd573e231",
			Type: "workitemlinkcategories",
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

	// Use the goa generated code to create a work item link category
	payload := app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   "bf30167a-9446-42de-82be-6b3815152051",
			Type: "workitemlinkcategories",
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
func (s *WorkItemLinkCategorySuite) TestCreateWorkItemLinkCategory() {
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	assert.NotNil(s.T(), linkCatSystem)
	//assert.Equal(s.T(), "animal", wit.Name)

	_, linkCatUser := s.createWorkItemLinkCategoryUser()
	assert.NotNil(s.T(), linkCatUser)
}

// TestShowWorkItemLinkCategory tests if we can fetch the "system" work item link category
func (s *WorkItemLinkCategorySuite) TestShowWorkItemLinkCategory() {
	// Create the work item link category first and try to read it back in
	_, linkCat := s.createWorkItemLinkCategorySystem()
	assert.NotNil(s.T(), linkCat)

	_, linkCat2 := test.ShowWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl, linkCat.Data.ID)

	assert.NotNil(s.T(), linkCat2)
	assert.EqualValues(s.T(), linkCat, linkCat2)
}

// TestListWorkItemLinkCategory tests if we can find the work item link categories
// "system" and "user" in the list of work item link categories
func (s *WorkItemLinkCategorySuite) TestListWorkItemLinkCategory() {
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	assert.NotNil(s.T(), linkCatSystem)
	_, linkCatUser := s.createWorkItemLinkCategoryUser()
	assert.NotNil(s.T(), linkCatUser)

	// Fetch a single work item link category
	_, linkCatCollection := test.ListWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl)

	assert.NotNil(s.T(), linkCatCollection)
	assert.Nil(s.T(), linkCatCollection.Validate())

	// Check the number of found work item link categories
	assert.NotNil(s.T(), linkCatCollection.Data)
	assert.Condition(s.T(), func() bool {
		return (len(linkCatCollection.Data) >= 2)
	}, "At least two work item link categories must exist (system and user), but only %d exist.", len(linkCatCollection.Data))

	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(linkCatCollection.Data) && toBeFound > 0; i++ {
		if *linkCatCollection.Data[i].Data.Attributes.Name == "system" || *linkCatCollection.Data[i].Data.Attributes.Name == "user" {
			s.T().Log("Found work item link category in collection: ", linkCatCollection.Data[i].Data.Attributes.Name)
			toBeFound--
		}
	}
	assert.Exactly(s.T(), 0, toBeFound, "Not all required work item link categories (system and user) where found.")
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemLinkCategory(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(WorkItemLinkCategorySuite))
}

// Expected strcture of 401 error response
type errorResponseStruct struct {
	Id     string
	Code   string
	Status int
	Detail string
}

// testSecureAPI defines how a Test object is.
type testSecureAPI struct {
	method             string
	url                string
	expectedStatusCode int    // this will be tested against responseRecorder.Code
	expectedErrorCode  string // this will be tested only if response body gets unmarshelled into errorResponseStruct
	payload            *bytes.Buffer
	jwtToken           string
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
			url:                EndpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           GetExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                EndpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           GetMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                EndpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           GetValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                EndpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           "",
		},
		// Update Work Item API with different parameters
		{
			method:             http.MethodPut,
			url:                EndpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           GetExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                EndpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           GetMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                EndpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           GetValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPut,
			url:                EndpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           "",
		},
		// Delete Work Item API with different parameters
		{
			method:             http.MethodDelete,
			url:                EndpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           GetExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                EndpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           GetMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                EndpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           GetValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                EndpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "jwt_security_error",
			payload:            nil,
			jwtToken:           "",
		},
		// Try fetching a random work item link category
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                EndpointWorkItemLinkCategories + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "not_found",
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func TestUnauthorizeWorkItemLinkCategoryCUD(t *testing.T) {
	resource.Require(t, resource.Database)

	// This will be modified after merge PR for "Viper Environment configurations"
	publickey, err := jwt.ParseRSAPublicKeyFromPEM((configuration.GetTokenPublicKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	tokenTests := getWorkItemLinkCategoryTestData(t)

	for _, testObject := range tokenTests {
		// Build a request
		var req *http.Request
		var err error
		if testObject.payload == nil {
			req, err = http.NewRequest(testObject.method, testObject.url, nil)
		} else {
			req, err = http.NewRequest(testObject.method, testObject.url, testObject.payload)
		}
		// req, err := http.NewRequest(testObject.method, testObject.url, testObject.payload)
		if err != nil {
			t.Fatal("could not create a HTTP request")
		}
		// Add Authorization Header
		req.Header.Add("Authorization", testObject.jwtToken)

		rr := httptest.NewRecorder()

		// temperory service for testing the middleware
		service := goa.New("TestUnauthorizedCreateWorkItemLinkCategory-Service")
		assert.NotNil(t, service)

		// if error is thrown during request processing, it will be caught by ErrorHandler middleware
		// this will put error code, status, details in recorder object.
		// e.g> {"id":"AL6spYb2","code":"jwt_security_error","status":401,"detail":"JWT validation failed: crypto/rsa: verification error"}
		service.Use(middleware.ErrorHandler(service, true))

		// append a middleware to service. Use appropriate RSA keys
		jwtMiddleware := goajwt.New(publickey, nil, app.NewJWTSecurity())
		// Adding middleware via "app" is important
		// Because it will check the design and accordingly apply the middleware if mentioned in design
		// But if I use `service.Use(jwtMiddleware)` then middleware is applied for all the requests (without checking design)
		app.UseJWTMiddleware(service, jwtMiddleware)

		controller := NewWorkitemController(service, gormapplication.NewGormDB(DB))
		app.MountWorkitemController(service, controller)

		// Hit the service with own request
		service.Mux.ServeHTTP(rr, req)

		assert.Equal(t, testObject.expectedStatusCode, rr.Code)

		// Below code tries to open Body response which is expected to be a JSON
		// If could not parse it correctly into errorResponseStruct
		// Then it gets logged and continue the test loop
		content := new(errorResponseStruct)
		err = json.Unmarshal(rr.Body.Bytes(), content)
		if err != nil {
			t.Log("Could not parse JSON response: ", rr.Body)
			// safe to continue because we alread checked rr.Code=required_value
			continue
		}
		// Additional checks for 'more' confirmation
		assert.Equal(t, testObject.expectedErrorCode, content.Code)
		assert.Equal(t, testObject.expectedStatusCode, content.Status)
	}
}
