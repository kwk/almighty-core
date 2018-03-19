package testfixture

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/query"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

func makeIdentities(fxt *TestFixture) error {
	if fxt.info[kindIdentities] == nil {
		return nil
	}
	fxt.Identities = make([]*account.Identity, fxt.info[kindIdentities].numInstances)
	for i := range fxt.Identities {
		fxt.Identities[i] = &account.Identity{
			Username:     testsupport.CreateRandomValidTestName("John Doe "),
			ProviderType: "test provider", // alternatively: account.KeycloakIDP
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindIdentities); err != nil {
			return errs.WithStack(err)
		}
		err := testsupport.CreateTestIdentityForAccountIdentity(fxt.db, fxt.Identities[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create identity: %+v", fxt.Identities[i])
		}
	}
	return nil
}

func makeWorkItemLinkCategories(fxt *TestFixture) error {
	if fxt.info[kindWorkItemLinkCategories] == nil {
		return nil
	}
	fxt.WorkItemLinkCategories = make([]*link.WorkItemLinkCategory, fxt.info[kindWorkItemLinkCategories].numInstances)
	wilcRepo := link.NewWorkItemLinkCategoryRepository(fxt.db)
	for i := range fxt.WorkItemLinkCategories {
		desc := "some description"
		fxt.WorkItemLinkCategories[i] = &link.WorkItemLinkCategory{
			Name:        testsupport.CreateRandomValidTestName("link category "),
			Description: &desc,
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindWorkItemLinkCategories); err != nil {
			return errs.WithStack(err)
		}
		_, err := wilcRepo.Create(fxt.ctx, fxt.WorkItemLinkCategories[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link category: %+v", fxt.WorkItemLinkCategories[i])
		}
	}
	return nil
}

func makeSpaceTemplates(fxt *TestFixture) error {
	if fxt.info[kindSpaceTemplates] == nil {
		return nil
	}
	fxt.SpaceTemplates = make([]*spacetemplate.SpaceTemplate, fxt.info[kindSpaceTemplates].numInstances)
	spaceTemplateRepo := spacetemplate.NewRepository(fxt.db)
	for i := range fxt.SpaceTemplates {
		fxt.SpaceTemplates[i] = &spacetemplate.SpaceTemplate{
			Name: testsupport.CreateRandomValidTestName("space template "),
		}
		fxt.SpaceTemplates[i].Description = ptr.String("Description for " + fxt.SpaceTemplates[i].Name)

		if err := fxt.runCustomizeEntityFuncs(i, kindSpaceTemplates); err != nil {
			return errs.WithStack(err)
		}
		st, err := spaceTemplateRepo.Create(fxt.ctx, *fxt.SpaceTemplates[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create space template: %+v", fxt.SpaceTemplates[i])
		}
		fxt.SpaceTemplates[i] = st
	}
	return nil
}

func makeSpaces(fxt *TestFixture) error {
	if fxt.info[kindSpaces] == nil {
		return nil
	}
	fxt.Spaces = make([]*space.Space, fxt.info[kindSpaces].numInstances)
	spaceRepo := space.NewRepository(fxt.db)
	for i := range fxt.Spaces {
		fxt.Spaces[i] = &space.Space{
			Name:        testsupport.CreateRandomValidTestName("space "),
			Description: "Some description",
		}
		if !fxt.isolatedCreation {
			fxt.Spaces[i].OwnerID = fxt.Identities[0].ID
			fxt.Spaces[i].SpaceTemplateID = fxt.SpaceTemplates[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindSpaces); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.Spaces[i].OwnerID == uuid.Nil {
				return errs.New("you must specify an owner ID for each space")
			}
			if fxt.Spaces[i].SpaceTemplateID == uuid.Nil {
				return errs.New("you must specify a space template ID for each space")
			}
		}
		_, err := spaceRepo.Create(fxt.ctx, fxt.Spaces[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create space: %+v", fxt.Spaces[i])
		}
	}
	return nil
}

func makeWorkItemLinkTypes(fxt *TestFixture) error {
	if fxt.info[kindWorkItemLinkTypes] == nil {
		return nil
	}
	fxt.WorkItemLinkTypes = make([]*link.WorkItemLinkType, fxt.info[kindWorkItemLinkTypes].numInstances)
	wiltRepo := link.NewWorkItemLinkTypeRepository(fxt.db)
	for i := range fxt.WorkItemLinkTypes {
		desc := "some description"
		fxt.WorkItemLinkTypes[i] = &link.WorkItemLinkType{
			Name:        testsupport.CreateRandomValidTestName("work item link type "),
			Description: &desc,
			Topology:    link.TopologyTree,
			ForwardName: "forward name (e.g. blocks)",
			ReverseName: "reverse name (e.g. blocked by)",
		}
		if !fxt.isolatedCreation {
			fxt.WorkItemLinkTypes[i].SpaceTemplateID = fxt.SpaceTemplates[0].ID
			fxt.WorkItemLinkTypes[i].LinkCategoryID = fxt.WorkItemLinkCategories[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindWorkItemLinkTypes); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.WorkItemLinkTypes[i].SpaceTemplateID == uuid.Nil {
				return errs.New("you must specify a space template for each work item link type")
			}
			if fxt.WorkItemLinkTypes[i].LinkCategoryID == uuid.Nil {
				return errs.New("you must specify a link category for each work item link type")
			}
		}
		_, err := wiltRepo.Create(fxt.ctx, fxt.WorkItemLinkTypes[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link type: %+v", fxt.WorkItemLinkTypes[i])
		}
	}
	return nil
}

func makeIterations(fxt *TestFixture) error {
	if fxt.info[kindIterations] == nil {
		return nil
	}
	fxt.Iterations = make([]*iteration.Iteration, fxt.info[kindIterations].numInstances)
	iterationRepo := iteration.NewIterationRepository(fxt.db)
	for i := range fxt.Iterations {
		desc := "Some description"
		fxt.Iterations[i] = &iteration.Iteration{
			Name:        testsupport.CreateRandomValidTestName("iteration "),
			Description: &desc,
		}
		if !fxt.isolatedCreation {
			fxt.Iterations[i].SpaceID = fxt.Spaces[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindIterations); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.Iterations[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each iteration")
			}
		}
		err := iterationRepo.Create(fxt.ctx, fxt.Iterations[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create iteration: %+v", fxt.Iterations[i])
		}
	}
	return nil
}

func makeAreas(fxt *TestFixture) error {
	if fxt.info[kindAreas] == nil {
		return nil
	}
	fxt.Areas = make([]*area.Area, fxt.info[kindAreas].numInstances)
	areaRepo := area.NewAreaRepository(fxt.db)
	for i := range fxt.Areas {
		fxt.Areas[i] = &area.Area{
			Name: testsupport.CreateRandomValidTestName("area "),
		}
		if !fxt.isolatedCreation {
			fxt.Areas[i].SpaceID = fxt.Spaces[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindAreas); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.Areas[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each area")
			}
		}
		err := areaRepo.Create(fxt.ctx, fxt.Areas[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create area: %+v", fxt.Areas[i])
		}
	}
	return nil
}

func makeCodebases(fxt *TestFixture) error {
	if fxt.info[kindCodebases] == nil {
		return nil
	}
	fxt.Codebases = make([]*codebase.Codebase, fxt.info[kindCodebases].numInstances)
	codebaseRepo := codebase.NewCodebaseRepository(fxt.db)
	for i := range fxt.Codebases {
		stackID := "golang-default"
		fxt.Codebases[i] = &codebase.Codebase{
			Type:              "git",
			StackID:           &stackID,
			LastUsedWorkspace: "my-used-last-workspace",
			URL:               "git@github.com:fabric8-services/fabric8-wit.git",
		}
		if !fxt.isolatedCreation {
			fxt.Codebases[i].SpaceID = fxt.Spaces[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindCodebases); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.Codebases[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each codebase")
			}
		}
		err := codebaseRepo.Create(fxt.ctx, fxt.Codebases[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create codebase: %+v", fxt.Codebases[i])
		}
	}
	return nil
}

func makeWorkItemTypes(fxt *TestFixture) error {
	if fxt.info[kindWorkItemTypes] == nil {
		return nil
	}
	fxt.WorkItemTypes = make([]*workitem.WorkItemType, fxt.info[kindWorkItemTypes].numInstances)
	witRepo := workitem.NewWorkItemTypeRepository(fxt.db)
	for i := range fxt.WorkItemTypes {
		fxt.WorkItemTypes[i] = &workitem.WorkItemType{
			ID:           uuid.NewV4(),
			Name:         testsupport.CreateRandomValidTestName("work item type "),
			Description:  ptr.String("this work item type was automatically generated"),
			Icon:         "fa-bug",
			Extends:      workitem.SystemPlannerItem,
			CanConstruct: true,
			Fields:       workitem.FieldDefinitions{},
		}
		if !fxt.isolatedCreation {
			fxt.WorkItemTypes[i].SpaceTemplateID = fxt.SpaceTemplates[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindWorkItemTypes); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.WorkItemTypes[i].SpaceTemplateID == uuid.Nil {
				return errs.New("you must specify a space template ID for each work item type")
			}
		}
		m := fxt.WorkItemTypes[i]
		wit, err := witRepo.Create(fxt.ctx, m.SpaceTemplateID, &m.ID, &m.Extends, m.Name, m.Description, m.Icon, m.Fields, m.CanConstruct)
		if err != nil {
			return errs.Wrapf(err, "failed to create work item type %+v", fxt.WorkItemTypes[i])
		}
		fxt.WorkItemTypes[i] = wit
	}
	return nil
}

func makeWorkItemTypeGroups(fxt *TestFixture) error {
	if fxt.info[kindWorkItemTypeGroups] == nil {
		return nil
	}
	fxt.WorkItemTypeGroups = make([]*workitem.WorkItemTypeGroup, fxt.info[kindWorkItemTypeGroups].numInstances)
	witgRepo := workitem.NewWorkItemTypeGroupRepository(fxt.db)
	for i := range fxt.WorkItemTypeGroups {
		fxt.WorkItemTypeGroups[i] = &workitem.WorkItemTypeGroup{
			ID:       uuid.NewV4(),
			Name:     testsupport.CreateRandomValidTestName("work item type group "),
			Bucket:   workitem.BucketPortfolio,
			Icon:     "fa fa-suitcase",
			Position: i,
		}
		if !fxt.isolatedCreation {
			fxt.WorkItemTypeGroups[i].TypeList = append(fxt.WorkItemTypeGroups[i].TypeList, fxt.WorkItemTypes[0].ID)
			fxt.WorkItemTypeGroups[i].SpaceTemplateID = fxt.SpaceTemplates[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindWorkItemTypeGroups); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.WorkItemTypes[i].SpaceTemplateID == uuid.Nil {
				return errs.New("you must specify a space template ID for each work item type group")
			}
		}
		witg, err := witgRepo.Create(fxt.ctx, *fxt.WorkItemTypeGroups[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create work item type group %+v", fxt.WorkItemTypeGroups[i])
		}
		fxt.WorkItemTypeGroups[i] = witg
	}
	return nil
}

func makeWorkItems(fxt *TestFixture) error {
	if fxt.info[kindWorkItems] == nil {
		return nil
	}
	fxt.WorkItems = make([]*workitem.WorkItem, fxt.info[kindWorkItems].numInstances)
	wiRepo := workitem.NewWorkItemRepository(fxt.db)
	for i := range fxt.WorkItems {
		fxt.WorkItems[i] = &workitem.WorkItem{
			Fields: map[string]interface{}{
				workitem.SystemTitle: testsupport.CreateRandomValidTestName("work item "),
				workitem.SystemState: workitem.SystemStateNew,
			},
		}
		if !fxt.isolatedCreation {
			fxt.WorkItems[i].SpaceID = fxt.Spaces[0].ID
			fxt.WorkItems[i].Type = fxt.WorkItemTypes[0].ID
			fxt.WorkItems[i].Fields[workitem.SystemCreator] = fxt.Identities[0].ID.String()
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindWorkItems); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.WorkItems[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each work item")
			}
			if fxt.WorkItems[i].Type == uuid.Nil {
				return errs.New("you must specify a work item type ID for each work item")
			}
			_, ok := fxt.WorkItems[i].Fields[workitem.SystemCreator]
			if !ok {
				return errs.Errorf("you must specify a work creator ID for the \"%s\" field in %+v", workitem.SystemCreator, fxt.WorkItems[i].Fields)
			}
		}
		creatorIDStr, ok := fxt.WorkItems[i].Fields[workitem.SystemCreator].(string)
		if !ok {
			return errs.Errorf("failed to convert \"%s\" field to string in %+v: %v", workitem.SystemCreator, fxt.WorkItems[i].Fields, fxt.WorkItems[i].Fields[workitem.SystemCreator])
		}
		creatorID, err := uuid.FromString(creatorIDStr)
		if err != nil {
			return errs.Wrapf(err, "failed to convert \"%s\" field to uuid.UUID: %v", workitem.SystemCreator, fxt.WorkItems[i].Fields[workitem.SystemCreator])
		}

		wi, err := wiRepo.Create(fxt.ctx, fxt.WorkItems[i].SpaceID, fxt.WorkItems[i].Type, fxt.WorkItems[i].Fields, creatorID)
		if err != nil {
			return errs.Wrapf(err, "failed to create work item: %+v", fxt.WorkItems[i])
		}
		fxt.WorkItems[i] = wi
	}
	return nil
}

func makeWorkItemLinks(fxt *TestFixture) error {
	if fxt.info[kindWorkItemLinks] == nil {
		return nil
	}
	fxt.WorkItemLinks = make([]*link.WorkItemLink, fxt.info[kindWorkItemLinks].numInstances)
	wilRepo := link.NewWorkItemLinkRepository(fxt.db)
	for i := range fxt.WorkItemLinks {
		fxt.WorkItemLinks[i] = &link.WorkItemLink{}
		if !fxt.isolatedCreation {
			fxt.WorkItemLinks[i].LinkTypeID = fxt.WorkItemLinkTypes[0].ID
			// this is the logic that ensures, each work item is only appearing
			// in one link
			if fxt.normalLinkCreation {
				fxt.WorkItemLinks[i].SourceID = fxt.WorkItems[2*i].ID
				fxt.WorkItemLinks[i].TargetID = fxt.WorkItems[2*i+1].ID
			}
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindWorkItemLinks); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.WorkItemLinks[i].LinkTypeID == uuid.Nil {
				return errs.New("you must specify a work item link type for each work item link")
			}
			if fxt.WorkItemLinks[i].SourceID == uuid.Nil {
				return errs.New("you must specify a source work item for each work item link")
			}
			if fxt.WorkItemLinks[i].TargetID == uuid.Nil {
				return errs.New("you must specify a target work item for each work item link")
			}
		}
		// default choice for creatorID: take it from the creator of the source work item
		sourceWI, err := workitem.NewWorkItemRepository(fxt.db).LoadByID(fxt.ctx, fxt.WorkItemLinks[i].SourceID)
		if err != nil {
			return errs.Wrapf(err, "failed to load the source work item in order to fetch a creator ID for the link")
		}
		creatorIDStr, ok := sourceWI.Fields[workitem.SystemCreator].(string)
		if !ok {
			return errs.Errorf("failed to fetch the %s field from the source work item %s", workitem.SystemCreator, fxt.WorkItemLinks[i].SourceID)
		}
		creatorID, err := uuid.FromString(creatorIDStr)
		if err != nil {
			return errs.Wrapf(err, "failed to convert the string \"%s\" to a uuid.UUID object", creatorIDStr)
		}

		wilt, err := wilRepo.Create(fxt.ctx, fxt.WorkItemLinks[i].SourceID, fxt.WorkItemLinks[i].TargetID, fxt.WorkItemLinks[i].LinkTypeID, creatorID)
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link: %+v", fxt.WorkItemLinks[i])
		}
		fxt.WorkItemLinks[i] = wilt
	}
	return nil
}

func makeComments(fxt *TestFixture) error {
	if fxt.info[kindComments] == nil {
		return nil
	}
	fxt.Comments = make([]*comment.Comment, fxt.info[kindComments].numInstances)
	commentRepo := comment.NewRepository(fxt.db)
	for i := range fxt.Comments {
		loremIpsum := `Lorem ipsum dolor sitamet, consectetur adipisicing elit, sed do eiusmod
	tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
	quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
	consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
	dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident,
	sunt in culpa qui officia deserunt mollit anim id est laborum.`
		fxt.Comments[i] = &comment.Comment{
			Markup: rendering.SystemMarkupMarkdown,
			Body:   loremIpsum,
		}
		if !fxt.isolatedCreation {
			fxt.Comments[i].ParentID = fxt.WorkItems[0].ID
			fxt.Comments[i].Creator = fxt.Identities[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindComments); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.Comments[i].ParentID == uuid.Nil {
				return errs.New("you must specify a parent work item ID for each comment")
			}
			if fxt.Comments[i].Creator == uuid.Nil {
				return errs.New("you must specify a creator ID for each comment")
			}
		}
		err := commentRepo.Create(fxt.ctx, fxt.Comments[i], fxt.Comments[i].Creator)
		if err != nil {
			return errs.Wrapf(err, "failed to create comment: %+v", fxt.Comments[i])
		}
	}
	return nil
}

func makeLabels(fxt *TestFixture) error {
	if fxt.info[kindLabels] == nil {
		return nil
	}
	fxt.Labels = make([]*label.Label, fxt.info[kindLabels].numInstances)
	labelRrepo := label.NewLabelRepository(fxt.db)

	randColor := func() string {
		colorBits := []string{"0", "1", "2", "3", "4", "5", "6", "a", "b", "c", "d", "e", "f"}
		strArr := make([]string, 6)
		for i := range strArr {
			strArr[i] = colorBits[rand.Intn(len(colorBits))]
		}
		return "#" + strings.Join(strArr, "")
	}
	for i := range fxt.Labels {
		fxt.Labels[i] = &label.Label{
			Name:            testsupport.CreateRandomValidTestName("label "),
			TextColor:       randColor(),
			BackgroundColor: randColor(),
			BorderColor:     randColor(),
		}
		if !fxt.isolatedCreation {
			fxt.Labels[i].SpaceID = fxt.Spaces[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindLabels); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.Labels[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each label")
			}
		}
		err := labelRrepo.Create(fxt.ctx, fxt.Labels[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create label: %+v", fxt.Labels[i])
		}
	}
	return nil
}

func makeTrackers(fxt *TestFixture) error {
	if fxt.info[kindTrackers] == nil {
		return nil
	}
	fxt.Trackers = make([]*remoteworkitem.Tracker, fxt.info[kindTrackers].numInstances)
	trackerRepo := remoteworkitem.NewTrackerRepository(fxt.db)

	for i := range fxt.Trackers {
		fxt.Trackers[i] = &remoteworkitem.Tracker{
			URL:  "https://api.github.com/",
			Type: remoteworkitem.ProviderGithub,
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindTrackers); err != nil {
			return errs.WithStack(err)
		}
		err := trackerRepo.Create(fxt.ctx, fxt.Trackers[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create tracker: %+v", fxt.Trackers[i])
		}
	}
	return nil
}

func makeQueries(fxt *TestFixture) error {
	if fxt.info[kindQueries] == nil {
		return nil
	}
	fxt.Queries = make([]*query.Query, fxt.info[kindQueries].numInstances)
	queryRrepo := query.NewQueryRepository(fxt.db)

	for i := range fxt.Queries {
		fxt.Queries[i] = &query.Query{
			Title:   testsupport.CreateRandomValidTestName("query "),
			Creator: fxt.Identities[0].ID,
		}
		if !fxt.isolatedCreation {
			fxt.Queries[i].Fields = fmt.Sprintf(`{"space": "%s"}`, fxt.Spaces[0].ID)
			fxt.Queries[i].SpaceID = fxt.Spaces[0].ID
		}
		if err := fxt.runCustomizeEntityFuncs(i, kindQueries); err != nil {
			return errs.WithStack(err)
		}
		if fxt.isolatedCreation {
			if fxt.Queries[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each query")
			}
		}
		err := queryRrepo.Create(fxt.ctx, fxt.Queries[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create query: %+v", fxt.Queries[i])
		}
	}
	return nil
}
