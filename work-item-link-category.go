package main

import (
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/models"
	"github.com/goadesign/goa"
	//satoriuuid "github.com/satori/go.uuid"
)

// WorkItemLinkCategoryController implements the work-item-link-category resource.
type WorkItemLinkCategoryController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemLinkCategoryController creates a WorkItemLinkCategoryController.
func NewWorkItemLinkCategoryController(service *goa.Service, db application.DB) *WorkItemLinkCategoryController {
	return &WorkItemLinkCategoryController{
		Controller: service.NewController("WorkItemLinkCategoryController"),
		db:         db,
	}
}

// Create runs the create action.
func (c *WorkItemLinkCategoryController) Create(ctx *app.CreateWorkItemLinkCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		cat, err := appl.WorkItemLinkCategories().Create(ctx.Context, ctx.Payload.Data.Attributes.Name, ctx.Payload.Data.Attributes.Description)
		if err != nil {
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusBadRequest, goa.ErrBadRequest(err.Error()))
			default:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
			}
		}
		ctx.ResponseData.Header().Set("Location", app.WorkItemLinkCategoryHref(cat.Data.ID))
		return ctx.Created(cat)
	})
}

// Show runs the show action.
func (c *WorkItemLinkCategoryController) Show(ctx *app.ShowWorkItemLinkCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		res, err := appl.WorkItemLinkCategories().Load(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				//ctx.ResponseData.Header().Set("Content-Type", "application/vnd.api+json")
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusNotFound, goa.ErrNotFound(err.Error()))
			default:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
			}
		}
		return ctx.OK(res)
	})
}

// List runs the list action.
func (c *WorkItemLinkCategoryController) List(ctx *app.ListWorkItemLinkCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemLinkCategories().List(ctx.Context)
		if err != nil {
			return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
		}
		return ctx.OK(result)
	})
}

// Delete runs the delete action.
func (c *WorkItemLinkCategoryController) Delete(ctx *app.DeleteWorkItemLinkCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItemLinkCategories().Delete(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				//ctx.ResponseData.Header().Set("Content-Type", "application/vnd.api+json")
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusNotFound, goa.ErrNotFound(err.Error()))
			default:
				return ctx.ResponseData.Service.Send(ctx.Context, http.StatusInternalServerError, goa.ErrNotFound(err.Error()))
			}
		}
		return ctx.OK([]byte{})
	})
}

// Update runs the update action.
func (c *WorkItemLinkCategoryController) Update(ctx *app.UpdateWorkItemLinkCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		toSave := app.WorkItemLinkCategory{
			Data: ctx.Payload.Data,
		}
		linkCategory, err := appl.WorkItemLinkCategories().Save(ctx.Context, toSave)
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
		return ctx.OK(linkCategory)
	})
}
