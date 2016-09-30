package remoteworkitem

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c != false {
		db, err = gorm.Open("postgres",
			fmt.Sprintf("host=%s port=%d user=%s password=%s DB.name=%s sslmode=%s",
				configuration.GetPostgresHost(),
				configuration.GetPostgresPort(),
				configuration.GetPostgresUser(),
				configuration.GetPostgresPassword(),
				configuration.GetPostgresDatabase(),
				configuration.GetPostgresSSLMode(),
			))
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer db.Close()
		// Migrate the schema
		db.AutoMigrate(
			&Tracker{},
			&TrackerQuery{},
			&TrackerItem{})
		db.Model(&TrackerQuery{}).AddForeignKey("tracker_id", "trackers(id)", "RESTRICT", "RESTRICT")
	}
	os.Exit(m.Run())
}

func TestNewScheduler(t *testing.T) {
	resource.Require(t, resource.Database)

	s := NewScheduler(db)
	if s.db != db {
		t.Error("DB not set as an attribute")
	}
	s.Stop()
}

func TestLookupProvider(t *testing.T) {
	resource.Require(t, resource.Database)
	ts1 := trackerSchedule{TrackerType: ProviderGithub}
	tp1 := LookupProvider(ts1)
	if tp1 == nil {
		t.Error("nil provider")
	}
	ts2 := trackerSchedule{TrackerType: ProviderJira}
	tp2 := LookupProvider(ts2)
	if tp2 == nil {
		t.Error("nil provider")
	}
	ts3 := trackerSchedule{TrackerType: "unknown"}
	tp3 := LookupProvider(ts3)
	if tp3 != nil {
		t.Error("non-nil provider")
	}
}
