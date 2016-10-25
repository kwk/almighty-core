package models_test

import (
	"testing"

	"time"

	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
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
	assert.False(t, a.Equal(b))

	// Test lifecycle
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	assert.False(t, a.Equal(c))

	// Test version
	d := a
	d.Version += 1
	assert.False(t, a.Equal(d))

	// Test name
	e := a
	e.Name = "bar"
	assert.False(t, a.Equal(e))

	// Test description
	otherDescription := "bar"
	f := a
	f.Description = &otherDescription
	assert.False(t, a.Equal(f))

	// Test equality
	g := a
	assert.True(t, a.Equal(g))
}
