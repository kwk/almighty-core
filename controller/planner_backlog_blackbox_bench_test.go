package controller_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

type BenchPlannerBacklogBlackboxREST struct {
	gormbench.DBBenchSuite
	testIdentity account.Identity
	svc          *goa.Service
	ctrl         *PlannerBacklogController
}

var testBench *testing.T

func TestRunPlannerBacklogBlackboxBenchREST(t *testing.T) {
	testBench = t
}

func BenchmarkRunPlannerBacklogBlackboxREST(b *testing.B) {
	resource.Require(b, resource.Database)
	testsupport.Run(b, &BenchPlannerBacklogBlackboxREST{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (rest *BenchPlannerBacklogBlackboxREST) SetupSuite() {
	rest.DBBenchSuite.SetupSuite()
	rest.svc = goa.New("PlannerBacklog-Service")
	rest.ctrl = NewPlannerBacklogController(rest.svc, gormapplication.NewGormDB(rest.DB), rest.Configuration)
}

func (rest *BenchPlannerBacklogBlackboxREST) SetupBenchmark() {
	rest.DBBenchSuite.SetupBenchmark()
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestPlannerBacklogBlackboxREST user", "test provider")
	if err != nil {
		rest.B().Fail()
	}
	rest.testIdentity = *testIdentity
}

func (rest *BenchPlannerBacklogBlackboxREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		spacesRepo := app.Spaces()
		testSpace = &space.Space{
			Name: "PlannerBacklogWorkItems-" + uuid.NewV4().String(),
		}
		_, err := spacesRepo.Create(rest.Ctx, testSpace)
		if err != nil {
			rest.B().Fail()
		}
		log.Info(nil, map[string]interface{}{"space_id": testSpace.ID}, "created space")
		workitemTypesRepo := app.WorkItemTypes()
		workitemType, err := workitemTypesRepo.Create(rest.Ctx, testSpace.ID, nil, &workitem.SystemPlannerItem, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{}, true)
		if err != nil {
			rest.B().Fail()
		}
		log.Info(nil, map[string]interface{}{"wit_id": workitemType.ID}, "created workitem type")

		iterationsRepo := app.Iterations()
		parentIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: testSpace.ID,
			State:   iteration.StateNew,
		}
		iterationsRepo.Create(rest.Ctx, parentIteration)
		log.Info(nil, map[string]interface{}{"parent_iteration_id": parentIteration.ID}, "created parent iteration")

		childIteration := &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: testSpace.ID,
			Path:    append(parentIteration.Path, parentIteration.ID),
			State:   iteration.StateStart,
		}
		iterationsRepo.Create(rest.Ctx, childIteration)
		log.Info(nil, map[string]interface{}{"child_iteration_id": childIteration.ID}, "created child iteration")

		fields := map[string]interface{}{
			workitem.SystemTitle:     "parentIteration Test",
			workitem.SystemState:     "new",
			workitem.SystemIteration: parentIteration.ID.String(),
		}
		app.WorkItems().Create(rest.Ctx, testSpace.ID, workitemType.ID, fields, rest.testIdentity.ID)

		fields2 := map[string]interface{}{
			workitem.SystemTitle:     "childIteration Test",
			workitem.SystemState:     "closed",
			workitem.SystemIteration: childIteration.ID.String(),
		}
		createdWI, err = app.WorkItems().Create(rest.Ctx, testSpace.ID, workitemType.ID, fields2, rest.testIdentity.ID)
		if err != nil {
			rest.B().Fail()
		}
		return nil
	})
	return
}

func (rest *BenchPlannerBacklogBlackboxREST) BenchmarkListPlannerBacklogWorkItemsOK() {
	// given
	testSpace, _, _ := rest.setupPlannerBacklogWorkItems()
	// when
	offset := "0"
	filter := ""
	limit := -1
	// when
	rest.B().ResetTimer()
	rest.B().ReportAllocs()
	for n := 0; n < rest.B().N; n++ {
		if _, workitems := test.ListPlannerBacklogOK(testBench, rest.svc.Context, rest.svc, rest.ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, nil); len(workitems.Data) != 1 {
			rest.B().Fail()
		}
	}
}
