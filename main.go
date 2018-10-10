package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // blank import is used here for simplicity
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sagikazarmark/go-gin-gorm-opencensus/internal"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/trace"
)

func main() {
	// Create prometheus exporter
	pe, err := prometheus.NewExporter(prometheus.Options{
		Registry: prom.DefaultGatherer.(*prom.Registry),
	})
	if err != nil {
		panic(err)
	}

	// Sample every trace for the sake of the example.
	// Note: do not use this in production.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	// Create jaeger exporter
	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: os.Getenv("JAEGER_AGENT_ENDPOINT"),
		Endpoint:      os.Getenv("JAEGER_ENDPOINT"),
		ServiceName:   "go-gin-gorm-opencensus",
	})
	if err != nil {
		panic(err)
	}

	// Register jaeger as a Trace Exporter
	trace.RegisterExporter(je)

	// Connect to database
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	// Run migrations and fixtures
	db.AutoMigrate(internal.Person{})
	err = internal.Fixtures(db)
	if err != nil {
		panic(err)
	}

	// Initialize Gin engine
	r := gin.Default()

	// Add routes
	r.POST("/people", internal.CreatePerson(db))
	r.GET("/hello/:firstName", internal.Hello(db))
	r.GET("/metrics", gin.HandlerFunc(func(c *gin.Context) {
		pe.ServeHTTP(c.Writer, c.Request)
	}))

	// Listen and serve on 0.0.0.0:8080
	r.Run()
}
