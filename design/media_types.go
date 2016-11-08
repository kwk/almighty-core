package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// ALMStatus defines the status of the current running ALM instance
var ALMStatus = a.MediaType("application/vnd.status+json", func() {
	a.Description("The status of the current running instance")
	a.Attributes(func() {
		a.Attribute("commit", d.String, "Commit SHA this build is based on")
		a.Attribute("buildTime", d.String, "The time when built")
		a.Attribute("startTime", d.String, "The time when started")
		a.Attribute("error", d.String, "The error if any")
		a.Required("commit", "buildTime", "startTime")
	})
	a.View("default", func() {
		a.Attribute("commit")
		a.Attribute("buildTime")
		a.Attribute("startTime")
		a.Attribute("error")
	})
})

// AuthToken represents an authentication JWT Token
var AuthToken = a.MediaType("application/vnd.authtoken+json", func() {
	a.TypeName("AuthToken")
	a.Description("JWT Token")
	a.Attributes(func() {
		a.Attribute("token", d.String, "JWT Token")
		a.Required("token")
	})
	a.View("default", func() {
		a.Attribute("token")
	})
})

// workItem is the media type for work items
var workItem = a.MediaType("application/vnd.workitem+json", func() {
	a.TypeName("WorkItem")
	a.Description("A work item hold field values according to a given field type")
	a.Attribute("id", d.String, "unique id per installation")
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control")
	a.Attribute("type", d.String, "Name of the type of this work item")
	a.Attribute("fields", a.HashOf(d.String, d.Any), "The field values, according to the field type")

	a.Required("id")
	a.Required("version")
	a.Required("type")
	a.Required("fields")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("version")
		a.Attribute("type")
		a.Attribute("fields")
	})
})

var pagingLinks = a.Type("pagingLinks", func() {
	a.Attribute("prev", d.String)
	a.Attribute("next", d.String)
	a.Attribute("first", d.String)
	a.Attribute("last", d.String)
})

var meta = a.Type("workItemListResponseMeta", func() {
	a.Attribute("totalCount", d.Integer)

	a.Required("totalCount")
})

// workItemListResponse contains paged results for listing work items and paging links
var workItemListResponse = a.MediaType("application/vnd.workitemlist+json", func() {
	a.TypeName("WorkItemListResponse")
	a.Description("Holds the paginated response to a work item list request")
	a.Attribute("links", pagingLinks)
	a.Attribute("meta", meta)
	a.Attribute("data", a.CollectionOf(workItem))

	a.Required("links")
	a.Required("meta")
	a.Required("data")

	a.View("default", func() {
		a.Attribute("links", func() {
			a.Attribute("prev", d.String)
			a.Attribute("next", d.String)
			a.Attribute("first", d.String)
			a.Attribute("last", d.String)
		})
		a.Attribute("meta", func() {
			a.Attribute("totalCount", d.Number)
		})
		a.Attribute("data")
	})
})

// fieldDefinition defines the possible values for a field in a work item type
var fieldDefinition = a.Type("fieldDefinition", func() {
	a.Description("A fieldDescription aggregates a fieldType and additional field metadata")
	a.Attribute("required", d.Boolean)
	a.Attribute("type", fieldType)

	a.Required("required")
	a.Required("type")

	a.View("default", func() {
		a.Attribute("kind")
	})
})

// fieldType is the datatype of a single field in a work item tepy
var fieldType = a.Type("fieldType", func() {
	a.Description("A fieldType describes the values a particular field can hold")
	a.Attribute("kind", d.String, "The constant indicating the kind of type, for example 'string' or 'enum' or 'instant'")
	a.Attribute("componentType", d.String, "The kind of type of the individual elements for a list type. Required for list types. Must be a simple type, not  enum or list")
	a.Attribute("baseType", d.String, "The kind of type of the enumeration values for an enum type. Required for enum types. Must be a simple type, not  enum or list")
	a.Attribute("values", a.ArrayOf(d.Any), "The possible values for an enum type. The values must be of a type convertible to the base type")

	a.Required("kind")
})

// workItemType is the media type representing a work item type.
var workItemType = a.MediaType("application/vnd.workitemtype+json", func() {
	a.TypeName("WorkItemType")
	a.Description("A work item type describes the values a work item type instance can hold.")
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control")
	a.Attribute("name", d.String, "User Readable Name of this item type")
	a.Attribute("fields", a.HashOf(d.String, fieldDefinition), "Definitions of fields in this work item type")

	a.Required("version")
	a.Required("name")
	a.Required("fields")

	a.View("default", func() {
		a.Attribute("version")
		a.Attribute("name")
		a.Attribute("fields")
	})
	a.View("link", func() {
		a.Attribute("name")
	})

})

// Tracker configuration
var Tracker = a.MediaType("application/vnd.tracker+json", func() {
	a.TypeName("Tracker")
	a.Description("Tracker configuration")
	a.Attribute("id", d.String, "unique id per tracker")
	a.Attribute("url", d.String, "URL of the tracker")
	a.Attribute("type", d.String, "Type of the tracker")

	a.Required("id")
	a.Required("url")
	a.Required("type")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("url")
		a.Attribute("type")
	})
})

// TrackerQuery represents the search query with schedule
var TrackerQuery = a.MediaType("application/vnd.trackerquery+json", func() {
	a.TypeName("TrackerQuery")
	a.Description("Tracker query with schedule")
	a.Attribute("id", d.String, "unique id per installation")
	a.Attribute("query", d.String, "Search query")
	a.Attribute("schedule", d.String, "Schedule for fetch and import")
	a.Attribute("trackerID", d.String, "Tracker ID")

	a.Required("id")
	a.Required("query")
	a.Required("schedule")
	a.Required("trackerID")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("query")
		a.Attribute("schedule")
		a.Attribute("trackerID")
	})
})

// User represents a user object (TODO: add better description)
var User = a.MediaType("application/vnd.user+json", func() {
	a.TypeName("User")
	a.Description("ALM User")
	a.Attribute("fullName", d.String, "The users full name")
	a.Attribute("imageURL", d.String, "The avatar image for the user")

	a.View("default", func() {
		a.Attribute("fullName")
		a.Attribute("imageURL")
	})
})

var searchResponse = a.MediaType("application/vnd.search+json", func() {
	a.TypeName("SearchResponse")
	a.Description("Holds the paginated response to a search request")
	a.Attribute("links", pagingLinks)
	a.Attribute("meta", meta)
	a.Attribute("data", a.CollectionOf(workItem))

	a.Required("links")
	a.Required("meta")
	a.Required("data")

	a.View("default", func() {
		a.Attribute("links", func() {
			a.Attribute("prev", d.String)
			a.Attribute("next", d.String)
			a.Attribute("first", d.String)
			a.Attribute("last", d.String)
		})
		a.Attribute("meta", func() {
			a.Attribute("totalCount", d.Integer)
		})
		a.Attribute("data")
	})
})

//  // JSONAPIAttributesObject see http://jsonapi.org/format/#document-resource-object-attributes
//  var JSONAPIAttributesObject = a.Type("JSONAPIAttributesObject", func() {
//  	//a.TypeName("JSONAPIAttributesObject")
//  	a.Description(`The value of the attributes key MUST be an object (an \"attributes object\").
//  Members of the attributes object ("attributes") represent information about the resource object in
//  which it's defined.
//
//  Although has-one foreign keys (e.g. author_id) are often stored internally alongside other
//  information to be represented in a resource object, these keys SHOULD NOT appear as attributes.`)
//
//  	a.Attribute("attributes", a.HashOf(D.String, D.Any))
//  })
//
//  // JSONAPIResourceObject2 see http://jsonapi.org/format/#document-resource-objects
//  var JSONAPIResourceObject2 = a.Type("JSONAPIResourceObject2", func() {
//  	//a.TypeName("JSONAPIResourceObject2")
//  	a.Description(`Resource objects appear in a JSON API document to represent resources.`)
//  	a.Attribute("type", D.String)
//  	a.Attribute("id", D.UUID, func() {
//  		a.Metadata("struct:tag:jsonapi", "primary")
//  	})
//  	a.Required("type", "id")
//  })
//
//  // JSONAPIResourceObject see http://jsonapi.org/format/#document-resource-objects
//  func JSONAPIResourceObject(typeName string) func() *D.UserTypeDefinition {
//  	t := fmt.Sprintf("JSONAPIResourceObject_%s", typeName)
//  	return func() *D.UserTypeDefinition {
//  		return a.Type(t, func() {
//  			//a.TypeName(t)
//  			a.Description(`Resource objects appear in a JSON API document to represent resources.`)
//  			a.Attribute("id", D.UUID, func() {
//  				a.Metadata("struct:tag:jsonapi", fmt.Sprintf("primary,%s", typeName))
//  			})
//  			a.Required("id")
//  		})
//  	}
//  }
//
//  // JSONAPIMetaObject see http://jsonapi.org/format/#document-meta
//  var JSONAPIMetaObject = a.Type("JSONAPIMetaObject", func() {
//  	//a.TypeName("JSONAPIMetaObject")
//  	a.Description(`Where specified, a meta member can be used to include non-standard meta-information.
//  The value of each meta member MUST be an object (a "meta object").`)
//
//  	a.Attribute("meta", a.HashOf(D.String, D.Any))
//
//  	a.Required("meta")
//  })
//

// JSONAPIErrors is an array of JSONAPI error objects
var JSONAPIErrors = a.MediaType("application/vnd.jsonapierrors+json", func() {
	a.ContentType("application/vnd.api+json")
	a.TypeName("JSONAPIErrors")
	a.Description(``)
	a.Attributes(func() {
		a.Attribute("errors", a.ArrayOf(JSONAPIError))
		a.Required("errors")
	})
	a.View("default", func() {
		a.Attribute("errors")
		a.Required("errors")
	})
})

// WorkItemLinkCategory puts a category on a link between two work items.
// The category is attached to a work item link type.
var WorkItemLinkCategory = a.MediaType("application/vnd.work-item-link-category+json", func() {
	a.ContentType("application/vnd.api+json")
	a.TypeName("WorkItemLinkCategory")
	a.Description(`A link type can have a category like "system", "extension", or "user".
Those categories are handled by this media type.`)
	a.Attributes(func() {
		a.Attribute("data", WorkItemLinkCategoryData)
		a.Required("data")
	})
	a.View("default", func() {
		a.Attribute("data")
		a.Required("data")
	})
})

// WorkItemLinkCategoryArray is a collection of work WorkItemLinkCategoryData objects.
var WorkItemLinkCategoryArray = a.MediaType("application/vnd.work-item-link-category-array+json", func() {
	a.ContentType("application/vnd.api+json")
	a.TypeName("WorkItemLinkCategoryArray")
	a.Description(`An array of work item link categories`)
	a.Attributes(func() {
		a.Attribute("meta", WorkItemLinkCategoryArrayMeta)
		a.Attribute("data", a.ArrayOf(WorkItemLinkCategory))
		a.Required("data")
	})
	a.View("default", func() {
		a.Attribute("data")
		a.Attribute("meta")
		a.Required("data")
	})
})
