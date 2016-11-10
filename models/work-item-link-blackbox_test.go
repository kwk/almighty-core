package models_test

import (
	"testing"

	"time"

	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemLink_Equal Tests equality of two work item links
func TestWorkItemLink_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := models.WorkItemLink{
		ID:         satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		SourceID:   "1",
		Source:     models.WorkItem{ID: 1},
		TargetID:   "2",
		Target:     models.WorkItem{ID: 2},
		LinkTypeID: satoriuuid.FromStringOrNil("966e982c-615c-4879-961f-56e912cbc4f2"),
		LinkType:   models.WorkItemLinkType{ID: satoriuuid.FromStringOrNil("966e982c-615c-4879-961f-56e912cbc4f2")},
	}

	// Test equality
	b := a
	require.True(t, a.Equal(b))

	// Test types
	c := convert.DummyEqualer{}
	require.False(t, a.Equal(c))

	// Test lifecycle
	b = a
	b.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	require.False(t, a.Equal(b))

	// Test ID
	b = a
	b.ID = satoriuuid.FromStringOrNil("10616dae-0a28-4de5-9d79-c831dbcfd039")
	require.False(t, a.Equal(b))

	// Test Version
	b = a
	b.Version += 1
	require.False(t, a.Equal(b))

	// Test SourceID
	b = a
	b.SourceID = "1292387473"
	require.False(t, a.Equal(b))

	// Test TargetID
	b = a
	b.TargetID = "93092303290"
	require.False(t, a.Equal(b))

	// Test Source
	b = a
	b.Source = models.WorkItem{Type: "foobar"}
	require.False(t, a.Equal(b))

	// Test Target
	b = a
	b.Target = models.WorkItem{Type: "foobar"}
	require.False(t, a.Equal(b))

	// Test LinkTypeID
	b = a
	b.LinkTypeID = satoriuuid.FromStringOrNil("10a41146-3868-47cd-84ae-f96ea4c9d797")
	require.False(t, a.Equal(b))

	// Test LinkType
	b = a
	b.LinkType = models.WorkItemLinkType{ID: satoriuuid.FromStringOrNil("5a54e430-09ea-4ddb-a694-3b318ef2f6fc")}
	require.False(t, a.Equal(b))
}

func TestWorkItemLinkCheckValidForCreation(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := models.WorkItemLink{
		ID:         satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		SourceID:   "1",
		Source:     models.WorkItem{ID: 1},
		TargetID:   "2",
		Target:     models.WorkItem{ID: 2},
		LinkTypeID: satoriuuid.FromStringOrNil("966e982c-615c-4879-961f-56e912cbc4f2"),
		LinkType:   models.WorkItemLinkType{ID: satoriuuid.FromStringOrNil("966e982c-615c-4879-961f-56e912cbc4f2")},
	}

	// Check valid
	b := a
	require.Nil(t, b.CheckValidForCreation())

	// Check empty SourceID
	b = a
	b.SourceID = ""
	require.NotNil(t, b.CheckValidForCreation())

	// Check empty TargetID
	b = a
	b.TargetID = ""
	require.NotNil(t, b.CheckValidForCreation())

	// Check empty LinkTypeID
	b = a
	b.LinkTypeID = satoriuuid.Nil
	require.NotNil(t, b.CheckValidForCreation())
}
