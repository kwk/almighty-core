package migration

import (
	"fmt"
	"sync"
	"testing"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

func TestConcurrentMigrations(t *testing.T) {
	resource.Require(t, resource.Database)

	var db1 *gorm.DB

	var err error
	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	db1, err = gorm.Open("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
			configuration.GetPostgresHost(),
			configuration.GetPostgresPort(),
			configuration.GetPostgresUser(),
			configuration.GetPostgresPassword(),
			configuration.GetPostgresSSLMode(),
		))
	if err != nil {
		t.Fatal("Cannot connect to DB", err)
	}
	defer db1.Close()

	var db2 *gorm.DB

	db2, err = gorm.Open("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
			configuration.GetPostgresHost(),
			configuration.GetPostgresPort(),
			configuration.GetPostgresUser(),
			configuration.GetPostgresPassword(),
			configuration.GetPostgresSSLMode(),
		))
	if err != nil {
		t.Fatal("Cannot connect to DB", err)
	}
	defer db2.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := Migrate(db1)
		if err != nil {
			t.Error(err.Error())
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := Migrate(db2)
		if err != nil {
			t.Error(err.Error())
		}
	}()
	wg.Wait()
}
