package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
)

// WorkItemLinkController implements the work-item-link resource.
type WorkItemLinkController struct {
	*goa.Controller
}

// NewWorkItemLinkController creates a work-item-link controller.
func NewWorkItemLinkController(service *goa.Service) *WorkItemLinkController {
	return &WorkItemLinkController{Controller: service.NewController("WorkItemLinkController")}
}

// Create runs the create action.
func (c *WorkItemLinkController) Create(ctx *app.CreateWorkItemLinkContext) error {
	// WorkItemLinkController_Create: start_implement

	// Put your logic here

	// WorkItemLinkController_Create: end_implement
	return nil
}

// Delete runs the delete action.
func (c *WorkItemLinkController) Delete(ctx *app.DeleteWorkItemLinkContext) error {
	// WorkItemLinkController_Delete: start_implement

	// Put your logic here

	// WorkItemLinkController_Delete: end_implement
	return nil
}

// List runs the list action.
func (c *WorkItemLinkController) List(ctx *app.ListWorkItemLinkContext) error {
	// WorkItemLinkController_List: start_implement

	// Put your logic here

	// WorkItemLinkController_List: end_implement
	res := &app.WorkItemLinkArray{}
	return ctx.OK(res)
}

// Show runs the show action.
func (c *WorkItemLinkController) Show(ctx *app.ShowWorkItemLinkContext) error {
	// WorkItemLinkController_Show: start_implement

	// Put your logic here

	// WorkItemLinkController_Show: end_implement
	res := &app.WorkItemLink{}
	return ctx.OK(res)
}

// Update runs the update action.
func (c *WorkItemLinkController) Update(ctx *app.UpdateWorkItemLinkContext) error {
	// WorkItemLinkController_Update: start_implement

	// Put your logic here

	// WorkItemLinkController_Update: end_implement
	res := &app.WorkItemLink{}
	return ctx.OK(res)
}
