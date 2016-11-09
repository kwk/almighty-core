package main

import (
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/models"
	"github.com/goadesign/goa"
)

// WorkItemLinkController implements the work-item-link resource.
type WorkItemLinkController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemLinkController creates a work-item-link controller.
func NewWorkItemLinkController(service *goa.Service, db application.DB) *WorkItemLinkController {
	return &WorkItemLinkController{
		Controller: service.NewController("WorkItemLinkController"),
		db:         db,
	}
}

// Create runs the create action.
func (c *WorkItemLinkController) Create(ctx *app.CreateWorkItemLinkContext) error {
	// WorkItemLinkController_Create: start_implement
	// Convert payload from app to model representation
	model := models.WorkItemLink{}
	in := app.WorkItemLink{
		Data: ctx.Payload.Data,
	}
	err := models.ConvertLinkToModel(&in, &model)
	if err != nil {
		return ctx.ResponseData.Service.Send(ctx.Context, http.StatusBadRequest, goa.ErrBadRequest(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		cat, err := appl.WorkItemLinks().Create(ctx.Context, &model)
		if err != nil {
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusBadRequest, goa.ErrBadRequest(err.Error()))
			default:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
			}
		}
		ctx.ResponseData.Header().Set("Location", app.WorkItemLinkHref(cat.Data.ID))
		return ctx.Created(cat)
	})
	// WorkItemLinkController_Create: end_implement
	return nil
}

// Delete runs the delete action.
func (c *WorkItemLinkController) Delete(ctx *app.DeleteWorkItemLinkContext) error {
	// WorkItemLinkController_Delete: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItemLinks().Delete(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusNotFound, goa.ErrNotFound(err.Error()))
			default:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
			}
		}
		return ctx.OK([]byte{})
	})
	// WorkItemLinkController_Delete: end_implement
	return nil
}

// List runs the list action.
func (c *WorkItemLinkController) List(ctx *app.ListWorkItemLinkContext) error {
	// WorkItemLinkController_List: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemLinks().List(ctx.Context)
		if err != nil {
			return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
		}
		return ctx.OK(result)
	})
	// WorkItemLinkController_List: end_implement
}

// Show runs the show action.
func (c *WorkItemLinkController) Show(ctx *app.ShowWorkItemLinkContext) error {
	// WorkItemLinkController_Show: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		res, err := appl.WorkItemLinks().Load(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusNotFound, goa.ErrNotFound(err.Error()))
			default:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
			}
		}
		return ctx.OK(res)
	})
	// WorkItemLinkController_Show: end_implement
}

// Update runs the update action.
func (c *WorkItemLinkController) Update(ctx *app.UpdateWorkItemLinkContext) error {
	// WorkItemLinkController_Update: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		toSave := app.WorkItemLink{
			Data: ctx.Payload.Data,
		}
		linkType, err := appl.WorkItemLinks().Save(ctx.Context, toSave)
		if err != nil {
			switch err := err.(type) {
			case models.NotFoundError:
				//ctx.ResponseData.Header().Set("Content-Type", "application/vnd.api+json")
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusNotFound, goa.ErrNotFound(err.Error()))
			case models.BadParameterError, models.ConversionError, models.VersionConflictError:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusBadRequest, goa.ErrBadRequest(err.Error()))
			default:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
			}
		}
		return ctx.OK(linkType)
	})
	// WorkItemLinkController_Update: end_implement
}
