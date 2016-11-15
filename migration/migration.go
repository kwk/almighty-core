package migration

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/models"
	"github.com/jinzhu/gorm"
	"golang.org/x/net/context"
)

// AdvisoryLockID is a random number that should be used within the application
// by anybody who wants to modify the "version" table.
const AdvisoryLockID = 42

// fn defines the type of function that can be part of a migration steps
type fn func(tx *sql.Tx) error

// steps defines a collection of all the functions that make up a version
type steps []fn

// migrations defines all a collection of all the steps
type migrations []steps

// Migrate executes the required migration of the database on startup.
// For each successful migration, an entry will be written into the "version"
// table, that states when a certain version was reached.
func Migrate(db *sql.DB) error {
	var err error

	if db == nil {
		return fmt.Errorf("Database handle is nil\n")
	}

	m := getMigrations()

	var tx *sql.Tx
	for nextVersion := int64(0); nextVersion < int64(len(m)) && err == nil; nextVersion++ {

		tx, err = db.Begin()
		if err != nil {
			return fmt.Errorf("Failed to start transaction: %s\n", err)
		}

		err = migrateToNextVersion(tx, &nextVersion, m)

		if err != nil {
			oldErr := err
			log.Printf("Rolling back transaction due to: %s\n", err)
			if err = tx.Rollback(); err != nil {
				return fmt.Errorf("Error while rolling back transaction: %s\n", err)
			}
			return oldErr
		}

		if err = tx.Commit(); err != nil {
			return fmt.Errorf("Error during transaction commit: %s\n", err)
		}

	}

	if err != nil {
		return fmt.Errorf("Migration failed with error: %s\n", err)
	}

	return nil
}

// getMigrations returns the migrations all the migrations we have.
// Add your own migration to the end of this function.
// IMPORTANT: ALWAYS APPEND AT THE END AND DON'T CHANGE THE ORDER OF MIGRATIONS!
func getMigrations() migrations {
	m := migrations{}

	// Version 0
	m = append(m, steps{executeSQLFile("000-bootstrap.sql")})

	// Version 1
	m = append(m, steps{executeSQLFile("001-common.sql")})

	// Version 2
	m = append(m, steps{executeSQLFile("002-tracker-items.sql")})

	// Version 3
	m = append(m, steps{executeSQLFile("003-login.sql")})

	// Version 4
	m = append(m, steps{executeSQLFile("004-drop-tracker-query-id.sql")})

	// Version 5
	m = append(m, steps{executeSQLFile("005-add-search-index.sql")})

	// Version 6
	m = append(m, steps{executeSQLFile("005-work-item-links.sql")})

	// Version N
	//
	// In order to add an upgrade, simply append an array of MigrationFunc to the
	// the end of the "migrations" slice. The version numbers are determined by
	// the index in the array. The following code in comments show how you can
	// do a migration in 3 steps. If one of the steps fails, the others are not
	// executed.
	// If something goes wrong during the migration, all you need to do is return
	// an error that is not nil.

	/*
		m = append(m, steps{
			func(db *sql.Tx) error {
				// Execute random go code
				return nil
			},
			executeSQLFile("YOUR_OWN_FILE.sql"),
			func(db *sql.Tx) error {
				// Execute random go code
				return nil
			},
		})
	*/

	return m
}

// executeSQLFile loads the given filename from the packaged SQL files and
// executes it on the given database
func executeSQLFile(filename string) fn {
	return func(db *sql.Tx) error {
		data, err := Asset(filename)
		if err != nil {
			return err
		}
		_, err = db.Exec(string(data))
		return err
	}
}

// migrateToNextVersion migrates the database to the nextVersion.
// If the database is already at nextVersion or higher, the nextVersion
// will be set to the actual next version.
func migrateToNextVersion(tx *sql.Tx, nextVersion *int64, m migrations) error {
	// Obtain exclusive transaction level advisory that doesn't depend on any table.
	// Once obtained, the lock is held for the remainder of the current transaction.
	// (There is no UNLOCK TABLE command; locks are always released at transaction end.)
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", AdvisoryLockID); err != nil {
		return fmt.Errorf("Failed to acquire lock: %s\n", err)
	}

	// Determine current version and adjust the outmost loop
	// iterator variable "version"
	currentVersion, err := getCurrentVersion(tx)
	if err != nil {
		return err
	}
	*nextVersion = currentVersion + 1
	if *nextVersion >= int64(len(m)) {
		// No further updates to apply (this is NOT an error)
		log.Printf("Current version %d. Nothing to update.", currentVersion)
		return nil
	}

	log.Printf("Attempt to update DB to version %d\n", *nextVersion)

	// Apply all the updates of the next version
	for j := range m[*nextVersion] {
		if err := m[*nextVersion][j](tx); err != nil {
			return fmt.Errorf("Failed to execute migration of step %d of version %d: %s\n", j, *nextVersion, err)
		}
	}

	if _, err := tx.Exec("INSERT INTO version(version) VALUES($1)", *nextVersion); err != nil {
		return fmt.Errorf("Failed to update DB to version %d: %s\n", *nextVersion, err)
	}

	log.Printf("Successfully updated DB to version %d\n", *nextVersion)
	return nil
}

// getCurrentVersion returns the highest version from the version
// table or -1 if that table does not exist.
//
// Returning -1 simplifies the logic of the migration process because
// the next version is always the current version + 1 which results
// in -1 + 1 = 0 which is exactly what we want as the first version.
func getCurrentVersion(db *sql.Tx) (int64, error) {
	row := db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_catalog='postgres' AND table_name='version')")

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return -1, fmt.Errorf("Failed to scan if table \"version\" exists: %s\n", err)
	}

	if !exists {
		// table doesn't exist
		return -1, nil
	}

	row = db.QueryRow("SELECT max(version) as current FROM version")

	var current int64 = -1
	if err := row.Scan(&current); err != nil {
		return -1, fmt.Errorf("Failed to scan max version in table \"version\": %s\n", err)
	}

	return current, nil
}

// PopulateCommonTypes makes sure the database is populated with the correct types (e.g. system.bug etc.)
func PopulateCommonTypes(ctx context.Context, db *gorm.DB, witr application.WorkItemTypeRepository) error {
	// FIXME: Need to add this conditionally
	// q := `ALTER TABLE "tracker_queries" ADD CONSTRAINT "tracker_fk" FOREIGN KEY ("tracker") REFERENCES "trackers" ON DELETE CASCADE`
	// db.Exec(q)

	if err := createOrUpdateSystemUserstory(ctx, witr, db); err != nil {
		return err
	}
	if err := createOrUpdateSystemValueProposition(ctx, witr, db); err != nil {
		return err
	}
	if err := createOrUpdateSystemFundamental(ctx, witr, db); err != nil {
		return err
	}
	if err := createOrUpdateSystemExperience(ctx, witr, db); err != nil {
		return err
	}
	if err := createOrUpdateSystemFeature(ctx, witr, db); err != nil {
		return err
	}
	if err := createOrUpdateSystemBug(ctx, witr, db); err != nil {
		return err
	}
	return nil
}

func createOrUpdateSystemUserstory(ctx context.Context, witr application.WorkItemTypeRepository, db *gorm.DB) error {
	return createOrUpdateCommon(models.SystemUserStory, ctx, witr, db)
}

func createOrUpdateSystemValueProposition(ctx context.Context, witr application.WorkItemTypeRepository, db *gorm.DB) error {
	return createOrUpdateCommon(models.SystemValueProposition, ctx, witr, db)
}

func createOrUpdateSystemFundamental(ctx context.Context, witr application.WorkItemTypeRepository, db *gorm.DB) error {
	return createOrUpdateCommon(models.SystemFundamental, ctx, witr, db)
}

func createOrUpdateSystemExperience(ctx context.Context, witr application.WorkItemTypeRepository, db *gorm.DB) error {
	return createOrUpdateCommon(models.SystemExperience, ctx, witr, db)
}

func createOrUpdateSystemFeature(ctx context.Context, witr application.WorkItemTypeRepository, db *gorm.DB) error {
	return createOrUpdateCommon(models.SystemFeature, ctx, witr, db)
}

func createOrUpdateSystemBug(ctx context.Context, witr application.WorkItemTypeRepository, db *gorm.DB) error {
	return createOrUpdateCommon(models.SystemBug, ctx, witr, db)
}

func createOrUpdateCommon(typeName string, ctx context.Context, witr application.WorkItemTypeRepository, db *gorm.DB) error {
	stString := "string"
	workItemTypeFields := map[string]app.FieldDefinition{
		models.SystemTitle:        app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: true},
		models.SystemDescription:  app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: false},
		models.SystemCreator:      app.FieldDefinition{Type: &app.FieldType{Kind: "user"}, Required: true},
		models.SystemAssignee:     app.FieldDefinition{Type: &app.FieldType{Kind: "user"}, Required: false},
		models.SystemRemoteItemID: app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: false},
		models.SystemState: app.FieldDefinition{
			Type: &app.FieldType{
				BaseType: &stString,
				Kind:     "enum",
				Values: []interface{}{
					models.SystemStateNew,
					models.SystemStateOpen,
					models.SystemStateInProgress,
					models.SystemStateResolved,
					models.SystemStateClosed,
				},
			},
			Required: true,
		},
	}

	_, err := witr.Load(ctx, typeName)
	switch err.(type) {
	case models.NotFoundError:
		_, err := witr.Create(ctx, nil, typeName, workItemTypeFields)
		if err != nil {
			return err
		}
	case nil:
		fmt.Println("Work item type exists, will update/overwrite the fields only ")
		convertedFields, err := models.TEMPConvertFieldTypesToModel(workItemTypeFields)
		if err != nil {
			return err
		}
		jsonArray, err := json.Marshal(convertedFields)
		jsonString := string(jsonArray[:])
		q := fmt.Sprintf(`UPDATE work_item_types SET fields='%s' where name = '%s'`, jsonString, typeName)
		db.Exec(q)
	}
	return nil
}
