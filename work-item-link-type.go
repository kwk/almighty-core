package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
)

// WorkItemLinkTypeController implements the work-item-link-type resource.
type WorkItemLinkTypeController struct {
	*goa.Controller
}

// NewWorkItemLinkTypeController creates a work-item-link-type controller.
func NewWorkItemLinkTypeController(service *goa.Service) *WorkItemLinkTypeController {
	return &WorkItemLinkTypeController{Controller: service.NewController("WorkItemLinkTypeController")}
}

// Create runs the create action.
func (c *WorkItemLinkTypeController) Create(ctx *app.CreateWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Create: start_implement

	// Put your logic here

	// WorkItemLinkTypeController_Create: end_implement
	return nil
}

// Delete runs the delete action.
func (c *WorkItemLinkTypeController) Delete(ctx *app.DeleteWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Delete: start_implement

	// Put your logic here

	// WorkItemLinkTypeController_Delete: end_implement
	return nil
}

// List runs the list action.
func (c *WorkItemLinkTypeController) List(ctx *app.ListWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_List: start_implement

	// Put your logic here

	// WorkItemLinkTypeController_List: end_implement
	res := &app.WorkItemLinkTypeArray{}
	return ctx.OK(res)
}

// Show runs the show action.
func (c *WorkItemLinkTypeController) Show(ctx *app.ShowWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Show: start_implement

	// Put your logic here

	// WorkItemLinkTypeController_Show: end_implement
	res := &app.WorkItemLinkType{}
	return ctx.OK(res)
}

// Update runs the update action.
func (c *WorkItemLinkTypeController) Update(ctx *app.UpdateWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Update: start_implement

	// Put your logic here

	// WorkItemLinkTypeController_Update: end_implement
	res := &app.WorkItemLinkType{}
	return ctx.OK(res)
}
