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

// TestWorkItemType_Equal Tests equality of two work item link types
func TestWorkItemLinkType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	uuid, _ := satoriuuid.FromString("0e671e36-871b-43a6-9166-0c4bd573e231")
	uuid2, _ := satoriuuid.FromString("0e671e36-871b-43a6-9166-0c4bd573eAAA")
	description := "An example description"
	const a = models.WorkItemLinkType{
		ID:             uuid,
		Name:           "Example work item link category",
		Description:    &description,
		Version:        0,
		SourceTypeName: "system.bug",
		SourceType:     models.WorkItemType{Name: "system.bug"},
		TargetTypeName: "systen.userstory",
		TargetType:     models.WorkItemType{Name: "system.userstory"},
		ForwardName:    "blocks",
		ReverseName:    "blocked by",
		LinkCategoryID: uuid2,
		LinkCategory:   models.WorkItemLinkCategory{},
	}

	// Test types
	c := convert.DummyEqualer{}
	assert.False(t, a.Equal(c))

	// Test equality
	b := a
	assert.True(t, a.Equal(b))

	// Test lifecycle
	b = a
	b.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	assert.False(t, a.Equal(b))

	// Test ID
	b = a
	b.ID = "CCC71e36-871b-43a6-9166-0c4bd573eCCC"
	assert.False(t, a.Equal(b))

	// Test Version
	b = a
	b.Version += 1
	assert.False(t, a.Equal(b))

	// Test Name
	b = a
	b.Name = "bar"
	assert.False(t, a.Equal(b))

	// Test Description
	otherDescription := "bar"
	b = a
	b.Description = &otherDescription
	assert.False(t, a.Equal(b))

	// Test SourceType
	b = a
	b.SourceType.Name = "foobar"
	assert.False(t, a.Equal(b))

	// Test TargetType
	b = a
	b.TargetType.Name = "foobar"
	assert.False(t, a.Equal(b))

	// Test SourceTypeName
	b = a
	b.SourceTypeName.Name = "foobar"
	assert.False(t, a.Equal(b))

	// Test TargetTypeName
	b = a
	b.TargetTypeName.Name = "foobar"
	assert.False(t, a.Equal(b))

	// Test ForwardName
	b = a
	b.ForwardName = "go, go, go!"
	assert.False(t, a.Equal(b))

	// Test ReverseName
	b = a
	b.ReverseName = "backup, backup!"
	assert.False(t, a.Equal(b))

}
