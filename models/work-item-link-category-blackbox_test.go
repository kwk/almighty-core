package models_test

import (
	"testing"

	"time"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemType_Equal Tests equality of two work item link categories
func TestWorkItemLinkCategory_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	uuid, _ := satoriuuid.FromString("0e671e36-871b-43a6-9166-0c4bd573e231")
	description := "An example description"
	a := models.WorkItemLinkCategory{
		ID:          uuid,
		Name:        "Example work item link category",
		Description: &description,
		Version:     0,
	}

	// Test types
	b := convert.DummyEqualer{}
	require.False(t, a.Equal(b))

	// Test lifecycle
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	require.False(t, a.Equal(c))

	// Test version
	d := a
	d.Version += 1
	require.False(t, a.Equal(d))

	// Test name
	e := a
	e.Name = "bar"
	require.False(t, a.Equal(e))

	// Test description
	otherDescription := "bar"
	f := a
	f.Description = &otherDescription
	require.False(t, a.Equal(f))

	// Test equality
	g := a
	require.True(t, a.Equal(g))
}

func TestWorkItemLinkCategory_ConvertLinkCategoryFromModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	m := models.WorkItemLinkCategory{
		ID:          satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:        "Example work item link category",
		Description: &description,
		Version:     0,
	}

	id := m.ID.String()
	expected := app.WorkItemLinkCategory{
		Data: &app.WorkItemLinkCategoryData{
			Type: models.EndpointWorkItemLinkCategories,
			ID:   &id,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &m.Name,
				Description: m.Description,
				Version:     &m.Version,
			},
		},
	}

	actual := models.ConvertLinkCategoryFromModel(&m)
	require.Equal(t, expected.Data.Type, actual.Data.Type)
	require.Equal(t, *expected.Data.ID, *actual.Data.ID)
	require.Equal(t, *expected.Data.Attributes.Name, *actual.Data.Attributes.Name)
	require.Equal(t, *expected.Data.Attributes.Description, *actual.Data.Attributes.Description)
	require.Equal(t, *expected.Data.Attributes.Version, *actual.Data.Attributes.Version)
}
