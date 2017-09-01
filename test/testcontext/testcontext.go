package testcontext

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// A TestContext object is the result of a call to
//  NewContext()
// or
//  NewContextIsolated()
//
// Don't create one on your own!
type TestContext struct {
	info             map[kind]*createInfo
	db               *gorm.DB
	isolatedCreation bool
	ctx              context.Context

	// Use this test reference in deferred creation function in order let the
	// test context creation fail.
	T *testing.T

	Identities             []*account.Identity          // Itentities (if any) that were created for this test context.
	Iterations             []*iteration.Iteration       // Iterations (if any) that were created for this test context.
	Areas                  []*area.Area                 // Areas (if any) that were created for this test context.
	Spaces                 []*space.Space               // Spaces (if any) that were created for this test context.
	Codebases              []*codebase.Codebase         // Codebases (if any) that were created for this test context.
	WorkItems              []*workitem.WorkItem         // Work items (if any) that were created for this test context.
	Comments               []*comment.Comment           // Comments (if any) that were created for this test context.
	WorkItemTypes          []*workitem.WorkItemType     // Work item types (if any) that were created for this test context.
	WorkItemLinkTypes      []*link.WorkItemLinkType     // Work item link types (if any) that were created for this test context.
	WorkItemLinkCategories []*link.WorkItemLinkCategory // Work item link categories (if any) that were created for this test context.
	WorkItemLinks          []*link.WorkItemLink         // Work item links (if any) that were created for this test context.
}

// A RecipeFunction ... TODO(kwk): document me
type RecipeFunction func(ctx *TestContext)

// A DeferredCreationFunc acts as a generic callback to the various entity
// creation functions (e.g. Identities(), Spaces(), etc.). The current test
// context is given with the ctx argument and the position of the object that
// will be created next is indicated by the index idx. That index can be used to
// look up e.g. a space with
//     s := ctx.Spaces[idx]
// That space s will be a ready-to-create space object on that you can modify to
// your liking.
//
// Notice that when you lookup objects in the test context, you can only safely
// access those object on which the entity depends, because those are guaranteed
// to be already created. For example when you try to access a work item type
// from a deferred space creation callback, it will not be very useful:
//     NewContext(t, db, WorkItemTypes(1), Spaces(1, func(ctx *TestContext, idx int){
//         witID := ctx.WorkItems[0].ID // this will not give you a uuid.Nil
//     }))
// On the other hand, you can safely lookup the space ID when you're in the
// deferred work item creation callback:
//     NewContext(t, db, WorkItemTypes(1), Spaces(1, func(ctx *TestContext, idx int){
//         witID := ctx.WorkItems[0].ID // this will not give you a uuid.Nil
//     }))
//
// Notice that you can do all kinds of distribution related functions in a
// deferred creation function. For example, you can control which identity owns
// a space or define what work item type each work item shall have. If not
// otherwise specified (e.g. as for WorkItemLinks()) we use a straight forward
// approach. So for example if you write
//     NewContext(t, db, Identities(7), Spaces(100))
// then we will create 10 identites and 100 spaces and the owner of all spaces
// will be identified with the ID of the first identity:
//     ctx.Identities[0].ID
// If you want a different distribution, you can create your own deferred
// creation function (see Identities() for an example).
//
// If you for some error reason you want your test context creation to fail you
// can use the ctx.T test instance:
//      NewContext(t, db, Identities(100, func(ctx *TestContext, idx int){
//          ctx.T.Fatal("some test failure reason")
//      }))
type DeferredCreationFunc func(ctx *TestContext, idx int)

// NewContext will create a test context by executing the recipies from the
// given recipe functions. If recipeFuncs is empty, nothing will happen. If
// there's an error during the setup, the given test t will fail.
//
// For example
//     NewContext(t, db, Comments(100))
// will create a work item (and everything required in order to create it) and
// author 100 comments for it. They will all be created by the same user if you
// don't tell the system to do it differently. For example, to create 100
// comments from 100 different users we can do the following:
//      NewContext(t, db, Identities(100), Comments(100, func(ctx *TestContext, idx int){
//          ctx.Comments[idx].Creator = ctx.Identities[idx].ID
//      }))
// That will create 100 identities and 100 comments and for each comment we're
// using the ID of one of the identities that have been created earlier. There's
// one important observation to make with this example: there's an order to how
// entities get created in the test context. That order is basically defined by
// the number of dependencies that each entity has. For example an identity has
// no dependency, so it will be created first and then can be accessed safely by
// any of the other entity creation functions. A comment for example depends on
// a work item which itself depends on a work item type and a space. The NewContext
// function does take of recursively resolving those dependcies first.
//
// If you just want to create 100 identities and 100 work items but don't care
// about resolving the dependencies automatically you can create the entities in
// isolation:
//      NewContextIsolated(t, db, Identities(100), Comments(100, func(ctx *TestContext, idx int){
//          ctx.Comments[idx].Creator = ctx.Identities[idx].ID
//          ctx.Comments[idx].ParentID = someExistingWorkItemID
//      }))
// Notice that I manually have to specify the ParentID of the work comment then
// because we cannot automatically resolve to which work item we will attach the
// comment.
func NewContext(t *testing.T, db *gorm.DB, recipeFuncs ...RecipeFunction) *TestContext {
	return newContext(t, db, false, recipeFuncs...)
}

// NewContextIsolated will create a test context by executing the recipies from
// the given recipe functions. If recipeFuncs is empty, nothing will happen. If
// there's an error during the setup, the given test t will fail.
//
// The difference to the normal NewContext function is that we will only create
// those object that where specified in the recipeFuncs. We will not create any
// object that is normally demanded by an object. For example, if you call
//     NewContext(t, db, WorkItems(1))
// you would (apart from other objects) get at least one work item AND a work
// item type because that is needed to create a work item. With
//     NewContextIsolated(t, db, Comments(2), WorkItems(1))
// on the other hand, we will only create a work item, two comments for it, and
// nothing more. And for sure your test will fail if you do that because you
// need to specify a space ID and a work item type ID for the created work item:
//     NewContextIsolated(t, db, Comments(2), WorkItems(1, func(ctx *TestContext, idx int){
//       ctx.WorkItems[idx].SpaceID = someExistingSpaceID
//       ctx.WorkItems[idx].WorkItemType = someExistingWorkItemTypeID
//     }))
func NewContextIsolated(t *testing.T, db *gorm.DB, setupFuncs ...RecipeFunction) *TestContext {
	return newContext(t, db, true, setupFuncs...)
}

// Identities tells the test context to create at least n identity objects.
//
// If called multiple times with differently n's, the biggest n wins. All
// deferred creation functions fns from all calls will be respected when
// creating the test context.
//
// Here's an example how you can create 42 identites and give them a numbered
// user name like "John Doe 0", "John Doe 1", and so forth:
//    Identities(42, func(ctx *TestContext, idx int){
//        ctx.Identities[idx].Username = "Jane Doe " + strconv.FormatInt(idx, 10)
//    })
// Notice that the index idx goes from 0 to n-1 and that you have to manually
// lookup the object from the test context. The identity object referenced by
//    ctx.Identities[idx]
// is guaranteed to be ready to be used for creation. That means, you don't
// necessarily have to touch it to avoid unique key violation for example. This
// is totally optional.
func Identities(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindIdentities, fns...)
	})
}

// Spaces tells the test context to create at least n space objects. See also
// the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Identities(1)
// but with NewContextIsolated(), no other objects will be created.
func Spaces(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindSpaces, fns...)
		if !ctx.isolatedCreation {
			Identities(1)(ctx)
		}
	})
}

// Iterations tells the test context to create at least n iteration objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
// but with NewContextIsolated(), no other objects will be created.
func Iterations(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindIterations, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
		}
	})
}

// Areas tells the test context to create at least n area objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
// but with NewContextIsolated(), no other objects will be created.
func Areas(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindAreas, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
		}
	})
}

// Codebases tells the test context to create at least n codebase objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
// but with NewContextIsolated(), no other objects will be created.
func Codebases(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindCodebases, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
		}
	})
}

// WorkItems tells the test context to create at least n work item objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
//     WorkItemTypes(1)
//     Identities(1)
// but with NewContextIsolated(), no other objects will be created.
//
// Notice that the Number field of a work item is only set after the work item
// has been created, so any changes you make to
//     ctx.WorkItems[idx].Number
// will have no effect.
func WorkItems(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItems, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx) // for the space ID
			WorkItemTypes(1)(ctx)
			Identities(1)(ctx) // for the creator ID
		}
	})
}

// Comments tells the test context to create at least n comment objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Identities(1)
//     WorkItems(1)
// but with NewContextIsolated(), no other objects will be created.
func Comments(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindComments, fns...)
		if !ctx.isolatedCreation {
			Identities(1)(ctx) // for the creator
			WorkItems(1)(ctx)
		}
	})
}

// WorkItemTypes tells the test context to create at least n work item type
// objects. See also the Identities() function for more general information on n
// and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
// but with NewContextIsolated(), no other objects will be created.
//
// The work item type that we create for each of the n instances is always the
// same and it tries to be compatible with the planner item work item type by
// specifying the same fields.
func WorkItemTypes(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItemTypes, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
		}
	})
}

// WorkItemLinkTypes tells the test context to create at least n work item link
// type objects. See also the Identities() function for more general information
// on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
//     WorkItemLinkCategories(1)
// but with NewContextIsolated(), no other objects will be created.
//
// We've created these helper functions that you should have a look at if you
// want to implement your own re-usable deferred creation functions:
//     TopologyNetwork()
//     TopologyDirectedNetwork()
//     TopologyDependency()
//     TopologyTree()
//     Topology(topology string) // programmatically set the topology
// The topology functions above are neat because you don't have to write a full
// callback function yourself.
//
// By default a call to
//     WorkItemLinkTypes(1)
// equals
//     WorkItemLinkTypes(1, TopologyTree())
// because we automatically set the topology for each link type to be "tree".
func WorkItemLinkTypes(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItemLinkTypes, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
			WorkItemLinkCategories(1)(ctx)
		}
	})
}

// Topology ensures that all created link types will have the given topology
// type.
func Topology(topology string) DeferredCreationFunc {
	return DeferredCreationFunc(func(ctx *TestContext, idx int) {
		ctx.WorkItemLinkTypes[idx].Topology = topology
	})
}

// TopologyNetwork ensures that all created link types will have the "network"
// topology type.
func TopologyNetwork() DeferredCreationFunc {
	return Topology(link.TopologyNetwork)
}

// TopologyDirectedNetwork ensures that all created link types will have the
// "directed network" topology type.
func TopologyDirectedNetwork() DeferredCreationFunc {
	return Topology(link.TopologyDirectedNetwork)
}

// TopologyDependency ensures that all created link types will have the
// "dependency" topology type.
func TopologyDependency() DeferredCreationFunc {
	return Topology(link.TopologyDependency)
}

// TopologyTree ensures that all created link types will have the "tree"
// topology type.
func TopologyTree() DeferredCreationFunc {
	return Topology(link.TopologyTree)
}

// WorkItemLinkCategories tells the test context to create at least n work item
// link category objects. See also the Identities() function for more general
// information on n and fns.
//
// No other objects will be created.
func WorkItemLinkCategories(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItemLinkCategories, fns...)
	})
}

// WorkItemLinks tells the test context to create at least n work item link
// objects. See also the Identities() function for more general information
// on n and fns.
//
// When called in NewContext() this function will call also call
//     WorkItemLinkTypes(1)
//     WorkItems(2*n)
// but with NewContextIsolated(), no other objects will be created.
//
// Notice, that we will create two times the number of work items of your
// requested links. The way those links will be created can for sure be
// influenced using a deferred creation function; but by default we create each
// link between two distinct work items. That means, no link will include the
// same work item.
func WorkItemLinks(n int, fns ...DeferredCreationFunc) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItemLinks, fns...)
		if !ctx.isolatedCreation {
			WorkItemLinkTypes(1)(ctx)
			WorkItems(2 * n)(ctx)
		}
	})
}

type kind string

const (
	kindIdentities             kind = "identities"
	kindIterations             kind = "iterations"
	kindAreas                  kind = "areas"
	kindSpaces                 kind = "spaces"
	kindCodebases              kind = "codebases"
	kindWorkItems              kind = "work_items"
	kindComments               kind = "comments"
	kindWorkItemTypes          kind = "work_item_types"
	kindWorkItemLinkTypes      kind = "work_item_link_types"
	kindWorkItemLinkCategories kind = "work_item_link_categories"
	kindWorkItemLinks          kind = "work_item_links"
)

type createInfo struct {
	numInstances        int
	deferredCreateFuncs []DeferredCreationFunc
}

func (ctx *TestContext) runDeferredCreationFunctions(idx int, k kind) {
	if ctx.info[k] == nil {
		ctx.T.Fatalf("the creation info for kind %s is nil (this should not happen)", k)
	}
	for _, dfn := range ctx.info[k].deferredCreateFuncs {
		dfn(ctx, idx)
	}
}

func (ctx *TestContext) setupInfo(n int, k kind, fns ...DeferredCreationFunc) {
	require.True(ctx.T, n > 0, "the number of objects to create must always be greater than zero")
	if _, ok := ctx.info[k]; !ok {
		ctx.info[k] = &createInfo{}
	}
	maxN := n
	if maxN < ctx.info[k].numInstances {
		maxN = ctx.info[k].numInstances
	}
	ctx.info[k].numInstances = maxN
	ctx.info[k].deferredCreateFuncs = append(ctx.info[k].deferredCreateFuncs, fns...)
}

func newContext(t *testing.T, db *gorm.DB, isolatedCreation bool, recipeFuncs ...RecipeFunction) *TestContext {
	ctx := TestContext{
		T:                t,
		info:             map[kind]*createInfo{},
		db:               db,
		isolatedCreation: isolatedCreation,
		ctx:              context.Background(),
	}

	for _, fn := range recipeFuncs {
		fn(&ctx)
	}

	// actually make the objects that DON'T have any dependencies
	makeIdentities(&ctx)
	makeWorkItemLinkCategories(&ctx)

	// actually make the objects that DO have any dependencies
	makeSpaces(&ctx)
	makeWorkItemLinkTypes(&ctx)
	makeCodebases(&ctx)
	makeWorkItemTypes(&ctx)
	makeIterations(&ctx)
	makeAreas(&ctx)
	makeWorkItems(&ctx)
	makeComments(&ctx)
	makeWorkItemLinks(&ctx)

	return &ctx
}

func makeIdentities(ctx *TestContext) {
	if ctx.info[kindIdentities] == nil {
		return
	}
	ctx.Identities = make([]*account.Identity, ctx.info[kindIdentities].numInstances)
	for i := range ctx.Identities {
		id := uuid.NewV4()
		ctx.Identities[i] = &account.Identity{
			ID:           id,
			Username:     "John Doe " + id.String(),
			ProviderType: "test provider",
		}

		ctx.runDeferredCreationFunctions(i, kindIdentities)

		err := testsupport.CreateTestIdentityForAccountIdentity(ctx.db, ctx.Identities[i])
		require.Nil(ctx.T, err, "failed to create identity: %+v", ctx.Identities[i])
	}
}

func makeWorkItemLinkCategories(ctx *TestContext) {
	if ctx.info[kindWorkItemLinkCategories] == nil {
		return
	}
	ctx.WorkItemLinkCategories = make([]*link.WorkItemLinkCategory, ctx.info[kindWorkItemLinkCategories].numInstances)
	for i := range ctx.WorkItemLinkCategories {
		id := uuid.NewV4()
		desc := "some description"
		ctx.WorkItemLinkCategories[i] = &link.WorkItemLinkCategory{
			ID:          id,
			Name:        "link category " + id.String(),
			Description: &desc,
		}
		ctx.runDeferredCreationFunctions(i, kindWorkItemLinkCategories)
		_, err := link.NewWorkItemLinkCategoryRepository(ctx.db).Create(ctx.ctx, ctx.WorkItemLinkCategories[i])
		require.Nil(ctx.T, err, "failed to create work item link category: %+v", ctx.WorkItemLinkCategories[i])
	}
}

func makeSpaces(ctx *TestContext) {
	if ctx.info[kindSpaces] == nil {
		return
	}
	ctx.Spaces = make([]*space.Space, ctx.info[kindSpaces].numInstances)
	for i := range ctx.Spaces {
		id := uuid.NewV4()
		ctx.Spaces[i] = &space.Space{
			ID:          id,
			Name:        "space " + id.String(),
			Description: "Some description",
		}
		if !ctx.isolatedCreation {
			ctx.Spaces[i].OwnerId = ctx.Identities[0].ID
		}
		ctx.runDeferredCreationFunctions(i, kindSpaces)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Spaces[i].OwnerId, "you must specify an owner ID for each space")
		}
		_, err := space.NewRepository(ctx.db).Create(ctx.ctx, ctx.Spaces[i])
		require.Nil(ctx.T, err, "failed to create space: %+v", ctx.Spaces[i])
	}
}

func makeWorkItemLinkTypes(ctx *TestContext) {
	if ctx.info[kindWorkItemLinkTypes] == nil {
		return
	}
	ctx.WorkItemLinkTypes = make([]*link.WorkItemLinkType, ctx.info[kindWorkItemLinkTypes].numInstances)
	for i := range ctx.WorkItemLinkTypes {
		id := uuid.NewV4()
		desc := "some description"
		ctx.WorkItemLinkTypes[i] = &link.WorkItemLinkType{
			ID:          id,
			Name:        "work item link type " + id.String(),
			Description: &desc,
			Topology:    link.TopologyTree,
			ForwardName: "forward name (e.g. blocks)",
			ReverseName: "reverse name (e.g. blocked by)",
		}
		if !ctx.isolatedCreation {
			ctx.WorkItemLinkTypes[i].SpaceID = ctx.Spaces[0].ID
			ctx.WorkItemLinkTypes[i].LinkCategoryID = ctx.WorkItemLinkCategories[0].ID
		}
		ctx.runDeferredCreationFunctions(i, kindWorkItemLinkTypes)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinkTypes[i].SpaceID, "you must specify a space for each work item link type")
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinkTypes[i].LinkCategoryID, "you must specify a link category for each work item link type")
		}
		_, err := link.NewWorkItemLinkTypeRepository(ctx.db).Create(ctx.ctx, ctx.WorkItemLinkTypes[i])
		require.Nil(ctx.T, err, "failed to create work item link type: %+v", ctx.WorkItemLinkTypes[i])
	}
}

func makeIterations(ctx *TestContext) {
	if ctx.info[kindIterations] == nil {
		return
	}
	ctx.Iterations = make([]*iteration.Iteration, ctx.info[kindIterations].numInstances)
	for i := range ctx.Iterations {
		desc := "Some description"
		id := uuid.NewV4()
		ctx.Iterations[i] = &iteration.Iteration{
			ID:          id,
			Name:        "iteration " + id.String(),
			Description: &desc,
		}
		if !ctx.isolatedCreation {
			ctx.Iterations[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runDeferredCreationFunctions(i, kindIterations)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Iterations[i].SpaceID, "you must specify a space ID for each iteration")
		}
		err := iteration.NewIterationRepository(ctx.db).Create(ctx.ctx, ctx.Iterations[i])
		require.Nil(ctx.T, err, "failed to create iteration: %+v", ctx.Iterations[i])
	}
}

func makeAreas(ctx *TestContext) {
	if ctx.info[kindAreas] == nil {
		return
	}
	ctx.Areas = make([]*area.Area, ctx.info[kindAreas].numInstances)
	for i := range ctx.Areas {
		id := uuid.NewV4()
		ctx.Areas[i] = &area.Area{
			ID:   id,
			Name: "area " + id.String(),
		}
		if !ctx.isolatedCreation {
			ctx.Areas[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runDeferredCreationFunctions(i, kindAreas)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Areas[i].SpaceID, "you must specify a space ID for each area")
		}
		err := area.NewAreaRepository(ctx.db).Create(ctx.ctx, ctx.Areas[i])
		require.Nil(ctx.T, err, "failed to create area: %+v", ctx.Areas[i])
	}
}

func makeCodebases(ctx *TestContext) {
	if ctx.info[kindCodebases] == nil {
		return
	}
	ctx.Codebases = make([]*codebase.Codebase, ctx.info[kindCodebases].numInstances)
	for i := range ctx.Codebases {
		id := uuid.NewV4()
		stackID := "golang-default"
		ctx.Codebases[i] = &codebase.Codebase{
			ID:                id,
			Type:              "git",
			StackID:           &stackID,
			LastUsedWorkspace: "my-used-last-workspace",
			URL:               "git@github.com:fabric8-services/fabric8-wit.git",
		}
		if !ctx.isolatedCreation {
			ctx.Codebases[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runDeferredCreationFunctions(i, kindCodebases)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Codebases[i].SpaceID, "you must specify a space ID for each codebase")
		}
		err := codebase.NewCodebaseRepository(ctx.db).Create(ctx.ctx, ctx.Codebases[i])
		require.Nil(ctx.T, err, "failed to create codebase: %+v", ctx.Codebases[i])
	}
}

func makeWorkItemTypes(ctx *TestContext) {
	if ctx.info[kindWorkItemTypes] == nil {
		return
	}
	ctx.WorkItemTypes = make([]*workitem.WorkItemType, ctx.info[kindWorkItemTypes].numInstances)
	for i := range ctx.WorkItemTypes {
		desc := "this work item type was automatically generated"
		id := uuid.NewV4()
		ctx.WorkItemTypes[i] = &workitem.WorkItemType{
			ID:          id,
			Name:        "work item type " + id.String(),
			Description: &desc,
			Icon:        "fa-bug",
			Fields: map[string]workitem.FieldDefinition{
				workitem.SystemTitle:        {Type: workitem.SimpleType{Kind: "string"}, Required: true, Label: "Title", Description: "The title text of the work item"},
				workitem.SystemDescription:  {Type: workitem.SimpleType{Kind: "markup"}, Required: false, Label: "Description", Description: "A descriptive text of the work item"},
				workitem.SystemCreator:      {Type: workitem.SimpleType{Kind: "user"}, Required: true, Label: "Creator", Description: "The user that created the work item"},
				workitem.SystemRemoteItemID: {Type: workitem.SimpleType{Kind: "string"}, Required: false, Label: "Remote item", Description: "The ID of the remote work item"},
				workitem.SystemCreatedAt:    {Type: workitem.SimpleType{Kind: "instant"}, Required: false, Label: "Created at", Description: "The date and time when the work item was created"},
				workitem.SystemUpdatedAt:    {Type: workitem.SimpleType{Kind: "instant"}, Required: false, Label: "Updated at", Description: "The date and time when the work item was last updated"},
				workitem.SystemOrder:        {Type: workitem.SimpleType{Kind: "float"}, Required: false, Label: "Execution Order", Description: "Execution Order of the workitem."},
				workitem.SystemIteration:    {Type: workitem.SimpleType{Kind: "iteration"}, Required: false, Label: "Iteration", Description: "The iteration to which the work item belongs"},
				workitem.SystemArea:         {Type: workitem.SimpleType{Kind: "area"}, Required: false, Label: "Area", Description: "The area to which the work item belongs"},
				workitem.SystemCodebase:     {Type: workitem.SimpleType{Kind: "codebase"}, Required: false, Label: "Codebase", Description: "Contains codebase attributes to which this WI belongs to"},
				workitem.SystemAssignees: {
					Type: &workitem.ListType{
						SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
						ComponentType: workitem.SimpleType{Kind: workitem.KindUser}},
					Required:    false,
					Label:       "Assignees",
					Description: "The users that are assigned to the work item",
				},
				workitem.SystemState: {
					Type: &workitem.EnumType{
						SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
						BaseType:   workitem.SimpleType{Kind: workitem.KindString},
						Values: []interface{}{
							workitem.SystemStateNew,
							workitem.SystemStateOpen,
							workitem.SystemStateInProgress,
							workitem.SystemStateResolved,
							workitem.SystemStateClosed,
						},
					},

					Required:    true,
					Label:       "State",
					Description: "The state of the work item",
				},
			},
		}
		if !ctx.isolatedCreation {
			ctx.WorkItemTypes[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runDeferredCreationFunctions(i, kindWorkItemTypes)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemTypes[i].SpaceID, "you must specify a space ID for each work item type")
		}
		_, err := workitem.NewWorkItemTypeRepository(ctx.db).CreateFromModel(ctx.ctx, ctx.WorkItemTypes[i])
		require.Nil(ctx.T, err)
	}
}

func makeWorkItems(ctx *TestContext) {
	if ctx.info[kindWorkItems] == nil {
		return
	}
	ctx.WorkItems = make([]*workitem.WorkItem, ctx.info[kindWorkItems].numInstances)
	for i := range ctx.WorkItems {
		id := uuid.NewV4()
		ctx.WorkItems[i] = &workitem.WorkItem{
			ID: id,
			Fields: map[string]interface{}{
				workitem.SystemTitle: fmt.Sprintf("work item %s", id),
				workitem.SystemState: workitem.SystemStateNew,
			},
		}
		if !ctx.isolatedCreation {
			ctx.WorkItems[i].SpaceID = ctx.Spaces[0].ID
			ctx.WorkItems[i].Type = ctx.WorkItemTypes[0].ID
			ctx.WorkItems[i].Fields[workitem.SystemCreator] = ctx.Identities[0].ID.String()
		}
		ctx.runDeferredCreationFunctions(i, kindWorkItems)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItems[i].SpaceID, "you must specify a space ID for each work item")
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItems[i].Type, "you must specify a work item type ID for each work item")
			_, ok := ctx.WorkItems[i].Fields[workitem.SystemCreator]
			require.True(ctx.T, ok, "you must specify a work creator ID for the \"%s\" field in %+v", workitem.SystemCreator, ctx.WorkItems[i].Fields)
		}
		creatorIDStr, ok := ctx.WorkItems[i].Fields[workitem.SystemCreator].(string)
		require.True(ctx.T, ok, "failed to convert \"%s\" field to string", workitem.SystemCreator)
		creatorID, err := uuid.FromString(creatorIDStr)
		require.Nil(ctx.T, err, "failed to convert \"%s\" field to uuid.UUID", workitem.SystemCreator)

		err = workitem.NewWorkItemRepository(ctx.db).CreateFromModel(ctx.ctx, ctx.WorkItems[i], creatorID)
		require.Nil(ctx.T, err, "failed to create work item: %+v", ctx.WorkItems[i])
	}
}

func makeWorkItemLinks(ctx *TestContext) {
	if ctx.info[kindWorkItemLinks] == nil {
		return
	}
	ctx.WorkItemLinks = make([]*link.WorkItemLink, ctx.info[kindWorkItemLinks].numInstances)
	for i := range ctx.WorkItemLinks {
		id := uuid.NewV4()
		ctx.WorkItemLinks[i] = &link.WorkItemLink{
			ID: id,
		}
		if !ctx.isolatedCreation {
			ctx.WorkItemLinks[i].LinkTypeID = ctx.WorkItemLinkTypes[0].ID
			// this is the logic that ensures, each work item is only appearing
			// in one link
			ctx.WorkItemLinks[i].SourceID = ctx.WorkItems[2*i].ID
			ctx.WorkItemLinks[i].TargetID = ctx.WorkItems[2*i+1].ID
		}
		ctx.runDeferredCreationFunctions(i, kindWorkItemLinks)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinks[i].LinkTypeID, "you must specify a work item link type for each work item link")
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinks[i].SourceID, "you must specify a source work item for each work item link")
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinks[i].TargetID, "you must specify a target work item for each work item link")
		}
		// default choice for creatorID: take it from the creator of the source work item
		sourceWI, err := workitem.NewWorkItemRepository(ctx.db).LoadByID(ctx.ctx, ctx.WorkItemLinks[i].SourceID)
		require.Nil(ctx.T, err, "failed to load the source work item in order to fetch a creator ID for the link")
		creatorIDStr, ok := sourceWI.Fields[workitem.SystemCreator].(string)
		require.True(ctx.T, ok, "failed to fetch the %s field from the source work item %s", workitem.SystemCreator, ctx.WorkItemLinks[i].SourceID)
		creatorID, err := uuid.FromString(creatorIDStr)
		require.Nil(ctx.T, err, "failed to convert the string \"%s\" to a uuid.UUID object", creatorIDStr)

		err = link.NewWorkItemLinkRepository(ctx.db).CreateFromModel(ctx.ctx, ctx.WorkItemLinks[i], creatorID)
		require.Nil(ctx.T, err, "failed to create work item link: %+v", ctx.WorkItemLinks[i])
	}
}

func makeComments(ctx *TestContext) {
	if ctx.info[kindComments] == nil {
		return
	}
	ctx.Comments = make([]*comment.Comment, ctx.info[kindComments].numInstances)
	for i := range ctx.Comments {
		id := uuid.NewV4()
		buf := `Lorem ipsum dolor sitamet, consectetur adipisicing elit, sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident,
sunt in culpa qui officia deserunt mollit anim id est laborum.`
		ctx.Comments[i] = &comment.Comment{
			ID:     id,
			Markup: rendering.SystemMarkupMarkdown,
			Body:   buf,
		}
		if !ctx.isolatedCreation {
			ctx.Comments[i].ParentID = ctx.WorkItems[0].ID
			ctx.Comments[i].Creator = ctx.Identities[0].ID
		}
		ctx.runDeferredCreationFunctions(i, kindComments)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Comments[i].ParentID, "you must specify a parent work item ID for each comment")
			require.NotEqual(ctx.T, uuid.Nil, ctx.Comments[i].Creator, "you must specify a creator ID for each comment")
		}
		err := comment.NewRepository(ctx.db).Create(ctx.ctx, ctx.Comments[i], ctx.Comments[i].Creator)
		require.Nil(ctx.T, err, "failed to create comment: %+v", ctx.Comments[i])
	}
}
