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
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemTypeTestSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type WorkItemTypeSuite struct {
	suite.Suite
	db       *gorm.DB
	typeCtrl *WorkitemtypeController
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *WorkItemTypeSuite) SetupSuite() {
	fmt.Println("--- Setting up test suite WorkItemTypeSuite ---")

	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	s.db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())

	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}

	svc := goa.New("WorkItemTypeSuite-Service")
	assert.NotNil(s.T(), svc)
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(s.db))
	assert.NotNil(s.T(), s.typeCtrl)

	// Make sure the database is populated with the correct types (e.g. system.bug etc.)
	if configuration.GetPopulateCommonTypes() {
		if err := models.Transactional(s.db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, models.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *WorkItemTypeSuite) TearDownSuite() {
	fmt.Println("--- Tearing down test suite WorkItemTypeSuite ---")
	if s.db != nil {
		s.db.Close()
	}
}

// removeWorkItemTypes removes all work item types from the db that will be created
// during these tests. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *WorkItemTypeSuite) removeWorkItemTypes() {

	s.db.Unscoped().Delete(&models.WorkItemType{Name: "person"})
	s.db.Unscoped().Delete(&models.WorkItemType{Name: "animal"})
}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that non of the work item types that we will create already exist.
func (s *WorkItemTypeSuite) SetupTest() {
	s.T().Log("--- Running SetupTest ---")
	s.removeWorkItemTypes()
}

// The TearDownTest method will be run after every test in the suite.
func (s *WorkItemTypeSuite) TearDownTest() {
	s.T().Log("--- Running TearDownTest ---")
	s.removeWorkItemTypes()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createWorkItemTypeAnimal defines a work item type "animal" that consists of
// two fields ("animal-type" and "color"). The type is mandatory but the color is not.
func (s *WorkItemTypeSuite) createWorkItemTypeAnimal() (http.ResponseWriter, *app.WorkItemType) {

	// Create an enumeration of animal names
	typeStrings := []string{"elephant", "blue whale", "Tyrannosaurus rex"}

	// Convert string slice to slice of interface{} in O(n) time.
	typeEnum := make([]interface{}, len(typeStrings))
	for i := range typeStrings {
		typeEnum[i] = typeStrings[i]
	}

	// Create the type for "animal-type" field based on the enum above
	stString := "string"
	typeFieldDef := app.FieldDefinition{
		Required: true,
		Type: &app.FieldType{
			BaseType: &stString,
			Kind:     "enum",
			Values:   typeEnum,
		},
	}

	// Create the type for the "color" field
	colorFieldDef := app.FieldDefinition{
		Required: false,
		Type: &app.FieldType{
			Kind: "string",
		},
	}

	// Use the goa generated code to create a work item type
	payload := app.CreateWorkItemTypePayload{
		Fields: map[string]*app.FieldDefinition{
			"animal_type": &typeFieldDef,
			"color":       &colorFieldDef,
		},
		Name: "animal",
	}

	return test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, &payload)
}

// createWorkItemTypePerson defines a work item type "person" that consists of
// a required "name" field.
func (s *WorkItemTypeSuite) createWorkItemTypePerson() (http.ResponseWriter, *app.WorkItemType) {

	// Create the type for the "color" field
	nameFieldDef := app.FieldDefinition{
		Required: true,
		Type: &app.FieldType{
			Kind: "string",
		},
	}

	// Use the goa generated code to create a work item type
	payload := app.CreateWorkItemTypePayload{
		Fields: map[string]*app.FieldDefinition{
			"name": &nameFieldDef,
		},
		Name: "person",
	}

	return test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, &payload)
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// TestCreateWorkItemType tests if we can create two work item types: "animal" and "person"
func (s *WorkItemTypeSuite) TestCreateWorkItemType() {
	_, wit := s.createWorkItemTypeAnimal()
	assert.NotNil(s.T(), wit)
	assert.Equal(s.T(), "animal", wit.Name)

	_, wit = s.createWorkItemTypePerson()
	assert.NotNil(s.T(), wit)
	assert.Equal(s.T(), "person", wit.Name)
}

// TestShowWorkItemType tests if we can fetch the work item type "animal".
func (s *WorkItemTypeSuite) TestShowWorkItemType() {
	// Create the work item type first and try to read it back in
	_, wit := s.createWorkItemTypeAnimal()
	assert.NotNil(s.T(), wit)

	_, wit2 := test.ShowWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, wit.Name)

	assert.NotNil(s.T(), wit2)
	assert.EqualValues(s.T(), wit, wit2)
}

// TestListWorkItemType tests if we can find the work item types
// "person" and "animal" in the list of work item types
func (s *WorkItemTypeSuite) TestListWorkItemType() {
	// Create the work item type first and try to read it back in
	_, witAnimal := s.createWorkItemTypeAnimal()
	assert.NotNil(s.T(), witAnimal)
	_, witPerson := s.createWorkItemTypePerson()
	assert.NotNil(s.T(), witPerson)

	// Fetch a single work item type
	// Paging in the format <start>,<limit>"
	page := "0,-1"
	_, witCollection := test.ListWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, &page)

	assert.NotNil(s.T(), witCollection)
	assert.Nil(s.T(), witCollection.Validate())

	// Check the number of found work item types
	assert.Condition(s.T(), func() bool {
		return (len(witCollection) >= 2)
	}, "At least two work item types must exist (animal and person), but only %d exist.", len(witCollection))

	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(witCollection) && toBeFound > 0; i++ {
		if witCollection[i].Name == "person" || witCollection[i].Name == "animal" {
			s.T().Log("Found work item type in collection: ", witCollection[i].Name)
			toBeFound--
		}
	}
	assert.Exactly(s.T(), 0, toBeFound, "Not all required work item types (animal and person) where found.")
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(WorkItemTypeSuite))
}

func getWorkItemTypeTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))

	createWITPayloadString := bytes.NewBuffer([]byte(`{"fields": {"system.administrator": {"Required": true,"Type": {"Kind": "string"}}},"name": "Epic"}`))

	return []testSecureAPI{
		// Create Work Item API with different parameters
		{
			method:             "POST",
			url:                "/api/workitemtypes",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWITPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             "POST",
			url:                "/api/workitemtypes",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWITPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             "POST",
			url:                "/api/workitemtypes",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWITPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             "POST",
			url:                "/api/workitemtypes",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWITPayloadString,
			jwtToken:           "",
		},
		// Try fetching a random work Item Type
		// We do not have security on GET hence this should return 404 not found
		{
			method:             "GET",
			url:                "/api/workitems/someRandomTestWIT8712",
			expectedStatusCode: 404,
			expectedErrorCode:  "not_found",
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func TestUnauthorizeWorkItemTypeCreate(t *testing.T) {
	resource.Require(t, resource.Database)

	// This will be modified after merge PR for "Viper Environment configurations"
	publickey, err := jwt.ParseRSAPublicKeyFromPEM((configuration.GetTokenPublicKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	tokenTests := getWorkItemTypeTestData(t)

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
		service := goa.New("TestUnauthorizedCreateWI-Service")
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

		controller := NewWorkitemtypeController(service, gormapplication.NewGormDB(DB))
		app.MountWorkitemtypeController(service, controller)

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
