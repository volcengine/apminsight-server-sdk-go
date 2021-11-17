package gorm_v1

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Test_example(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()
	defer func() {
		tracer.Stop()
	}()

	db, err := gorm.Open(mysql.Open("root:123456@tcp(127.0.0.1:3306)/byteapm?charset=utf8"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db, err = WrapDB(aitracer.MySQL, "127.0.0.1:3306", "byteapm", db, tracer)
	if err != nil {
		panic(err)
	}

	type User struct {
		ID           uint
		Name         string
		Email        *string
		Age          uint8
		Birthday     time.Time
		MemberNumber sql.NullString
		ActivatedAt  sql.NullTime
		CreatedAt    time.Time
		UpdatedAt    time.Time
	}

	ctx := context.Background()
	span, ctx := tracer.StartServerSpanFromContext(ctx, "gorm_example_test", aitracer.ServerResourceAs("gorm"))
	defer span.Finish()
	{
		user := User{Name: "Jinzhu", Age: 18, Birthday: time.Now()}
		db.WithContext(ctx).Create(&user) // pass pointer of data to Create
	}
	{
		user := User{}
		db.WithContext(ctx).First(&user)
	}
	{
		user := User{}
		db.WithContext(ctx).First(&user)

		user.Name = "jinzhu 2"
		user.Age = 100
		db.WithContext(ctx).Save(&user)
	}
	{
		db.WithContext(ctx).Delete(&User{}, 10)

		db.WithContext(ctx).Delete(&User{}, "10")

		var users []User
		db.WithContext(ctx).Delete(&users, []int{1, 2, 3})
	}

}
